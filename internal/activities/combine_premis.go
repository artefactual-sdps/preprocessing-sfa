package activities

import (
	"context"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/beevik/etree"
	"github.com/otiai10/copy"
	"github.com/terminalstatic/go-xsd-validate"
)

const CombinePREMISName = "combine-premis"

type CombinePREMISParams struct {
	Path string
}

type CombinePREMISResult struct{}

type CombinePREMIS struct{}

func NewCombinePREMIS() *CombinePREMIS {
	return &CombinePREMIS{}
}

func (a *CombinePREMIS) Execute(ctx context.Context, params *CombinePREMISParams) (*CombinePREMISResult, error) {
	// Get transfer's PREMIS file paths.
	file_paths, err := CombinePREMISGetPaths(params.Path)
	if err != nil {
		return nil, err
	}

	// Copy empty PREMIS file into metadata directory.
	source_filepath := "empty_premis.xml"
	dest_filepath := path.Join(params.Path, "metadata/premis.xml")

	err = copy.Copy(source_filepath, dest_filepath)
	if err != nil {
		return nil, err
	}

	// Write elements from transfer's PREMIS files to combined PREMIS file.
	combined_premis_filepath := path.Join(params.Path, "metadata/premis.xml")

	for _, file_path := range file_paths {
		err := CombinePREMISCopy(file_path, combined_premis_filepath)
		if err != nil {
			return nil, err
		}
	}

	return &CombinePREMISResult{}, nil
}

func CombinePREMISGetPaths(transfer_dir string) ([]string, error) {
	objects_dir := filepath.Join(transfer_dir, "objects")
	dir_items, err := os.ReadDir(objects_dir)
	if err != nil {
		return nil, err
	}

	file_paths := []string{}
	for _, dir_item := range dir_items {
		if dir_item.IsDir() {
			subdir := path.Join(objects_dir, dir_item.Name())

			sub_items, err := os.ReadDir(subdir)
			if err != nil {
				return nil, err
			}

			for _, subdir_item := range sub_items {
				if !subdir_item.IsDir() {
					if strings.HasSuffix(strings.ToLower(subdir_item.Name()), "_premis.xml") {
						file_paths = append(file_paths, path.Join(subdir, subdir_item.Name()))
					}
				}
			}
		}
	}

	return file_paths, nil
}

func CombinePREMISCopy(source_filepath, destination_filepath string) error {
	// Parse source document and get root PREMIS element.
	source_doc := etree.NewDocument()

	if err := source_doc.ReadFromFile(source_filepath); err != nil {
		return err
	}

	source_premis_element := source_doc.FindElement("/premis")
	if source_premis_element == nil {
		return errors.New("no root premis element found in source document")
	}

	// Read source child PREMIS elements.
	source_premis_object_elements := source_premis_element.FindElements("object")
	source_premis_event_elements := source_premis_element.FindElements("event")
	source_premis_agent_elements := source_premis_element.FindElements("agent")

	// Parse destination document and get root PREMIS element.
	dest_doc := etree.NewDocument()
	if err := dest_doc.ReadFromFile(destination_filepath); err != nil {
		return err
	}

	dest_premis_element := dest_doc.FindElement("/premis")
	if dest_premis_element == nil {
		return errors.New("no root premis element found in destination document")
	}

	// Update PREMIS originalname child elements of PREMIS object elements.
	dirName := filepath.Base(filepath.Dir(source_filepath))
	for _, premis_object_element := range source_premis_object_elements {
		objectname_element := premis_object_element.FindElement("originalName")
		if objectname_element != nil {
			objectname_element.SetText("objects/" + dirName + "/" + objectname_element.Text())
		}
	}

	// Write destination child PREMIS elements.
	CombinePREMISAddChildElements(dest_premis_element, source_premis_object_elements)
	CombinePREMISAddChildElements(dest_premis_element, source_premis_event_elements)
	CombinePREMISAddChildElements(dest_premis_element, source_premis_agent_elements)

	dest_doc.Indent(2)
	err := dest_doc.WriteToFile(destination_filepath)
	if err != nil {
		return err
	}

	// Validate final XML.
	err = CombinePREMISValidateXML(destination_filepath, "premis-v2-2.xsd")
	if err != nil {
		return err
	}

	return nil
}

func CombinePREMISAddChildElements(parent_element *etree.Element, new_child_elements []*etree.Element) {
	for _, child_element := range new_child_elements {
		child_element.Space = "premis"
		for _, element := range child_element.FindElements("//*") {
			element.Space = "premis"
		}
		parent_element.AddChild(child_element)
	}
}

func CombinePREMISValidateXML(xml_filepath string, xsd_filepath string) error {
	err := xsdvalidate.Init()
	if err != nil {
		return err
	}
	defer xsdvalidate.Cleanup()

	// Prepare XSD handler.
	xsdHandler, err := xsdvalidate.NewXsdHandlerUrl(xsd_filepath, xsdvalidate.ParsErrDefault)
	if err != nil {
		return err
	}

	// Prepare XML handler.
	xmlFile, err := os.Open(filepath.Clean(xml_filepath))
	if err != nil {
		return err
	}
	defer xmlFile.Close()

	inXml, err := io.ReadAll(xmlFile)
	if err != nil {
		return err
	}

	xmlhandler, err := xsdvalidate.NewXmlHandlerMem(inXml, xsdvalidate.ParsErrDefault)
	if err != nil {
		return err
	}
	defer xmlhandler.Free()

	// Validate the XML document against the XSD.
	err = xsdHandler.Validate(xmlhandler, xsdvalidate.ValidErrDefault)
	if err != nil {
		return err
	}

	return nil
}
