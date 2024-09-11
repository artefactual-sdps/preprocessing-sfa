package manifest

import (
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"slices"
)

type File struct {
	ID       string
	Checksum Checksum
}

type Checksum struct {
	Algorithm string
	Hash      string
}

var relevantElements = []string{
	"paket",
	"inhaltsverzeichnis",
	"ordner",
	"datei",
	"name",
	"pruefalgorithmus",
	"pruefsumme",
}

// Files parses r and returns a map of file paths to checksums.
func Files(r io.Reader) (map[string]*File, error) {
	var (
		file *File
		path string
	)

	// openElems is a stack representing open elements. It has an arbitrarily
	// large capacity to avoid unnecessary copies of the underlying array.
	openElems := make([]string, 100)
	files := make(map[string]*File)

	// decoder is an XML stream parser reading from r.
	decoder := xml.NewDecoder(r)
	for {
		tok, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("parse: %w", err)
		}

		switch elem := tok.(type) {
		case xml.StartElement:
			e := elem.Name.Local
			switch {
			case slices.Contains(relevantElements, e):
				if e == "datei" {
					var id string
					for _, a := range elem.Attr {
						if a.Name.Local == "id" {
							id = a.Value
							break
						}
					}
					file = &File{ID: id} // Create a new file instance.
				}

				// Add element to openElems stack.
				openElems = append(openElems, e)
			default:
				if err := decoder.Skip(); err != nil {
					return nil, fmt.Errorf("skip irrelevant element %s: %v", e, err)
				}
			}
		case xml.EndElement:
			if e := elem.Name.Local; e == openElems[len(openElems)-1] {
				switch e {
				case "datei":
					files[path] = file
					file = nil // Close file instance.
					fallthrough
				case "ordner":
					path = filepath.Dir(path) // Remove name from path.
				case "inhaltsverzeichnis":
					return files, nil // Stop parsing.
				}

				openElems = openElems[:len(openElems)-1] // Close element.
			}
		case xml.CharData:
			switch openElems[len(openElems)-1] {
			case "name":
				// Add ordner or datei name to file path.
				path = filepath.Join(path, string(elem))
			case "pruefalgorithmus":
				file.Checksum.Algorithm = string(elem)
			case "pruefsumme":
				file.Checksum.Hash = string(elem)
			}
		}
	}

	return files, nil
}
