package activities

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/manifest"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/premis"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

const AddPREMISObjectsName = "add-premis-objects"

type AddPREMISObjectsParams struct {
	SIP            sip.SIP
	PREMISFilePath string
}

type AddPREMISObjectsResult struct{}

type AddPREMISObjectsActivity struct{}

func NewAddPREMISObjects() *AddPREMISObjectsActivity {
	return &AddPREMISObjectsActivity{}
}

func (md *AddPREMISObjectsActivity) Execute(
	ctx context.Context,
	params *AddPREMISObjectsParams,
) (*AddPREMISObjectsResult, error) {
	// Get subpaths of files in transfer.
	subpaths, err := premis.FilesWithinDirectory(params.SIP.ContentPath)
	if err != nil {
		return nil, err
	}

	// Create parent directory, if necessary.
	mdPath := filepath.Dir(params.PREMISFilePath)
	if err := os.MkdirAll(mdPath, 0o700); err != nil {
		return nil, err
	}

	doc, err := premis.ParseOrInitialize(params.PREMISFilePath)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(params.SIP.ManifestPath)
	if err != nil {
		return nil, fmt.Errorf("open manifest file: %v", err)
	}
	defer f.Close()

	manifestFiles, err := manifest.Files(f)
	if err != nil {
		return nil, fmt.Errorf("parse manifest: %v", err)
	}

	for _, subpath := range subpaths {
		var id string
		if f, ok := manifestFiles[subpathToManifestPath(subpath)]; ok {
			id = f.ID
		}

		object := premis.Object{
			IdType:       "local",
			IdValue:      id,
			OriginalName: premis.OriginalNameForSubpath(params.SIP, subpath),
		}

		err = premis.AppendObjectXML(doc, object)
		if err != nil {
			return nil, err
		}
	}

	doc.Indent(2)
	err = doc.WriteToFile(params.PREMISFilePath)
	if err != nil {
		return nil, err
	}

	return &AddPREMISObjectsResult{}, nil
}

func subpathToManifestPath(path string) string {
	return filepath.Join("content", path)
}
