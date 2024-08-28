package activities

import (
	"context"
	"os"
	"path/filepath"

	"github.com/google/uuid"

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

	for _, subpath := range subpaths {
		object := premis.Object{
			IdType:       "UUID",
			IdValue:      uuid.New().String(),
			OriginalName: premis.OriginalNameForSubpath(params.SIP, subpath),
		}

		err = premis.AppendObjectXML(doc, object)
		if err != nil {
			return nil, err
		}
	}

	err = doc.WriteToFile(params.PREMISFilePath)
	if err != nil {
		return nil, err
	}

	return &AddPREMISObjectsResult{}, nil
}
