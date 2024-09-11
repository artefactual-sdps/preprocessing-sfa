package activities

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"go.artefactual.dev/tools/fsutil"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/pips"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

const TransformSIPName = "transform-sip"

type TransformSIPParams struct {
	SIP sip.SIP
}

type TransformSIPResult struct {
	PIP pips.PIP
}

type TransformSIP struct{}

func NewTransformSIP() *TransformSIP {
	return &TransformSIP{}
}

func (a *TransformSIP) Execute(ctx context.Context, params *TransformSIPParams) (*TransformSIPResult, error) {
	// Create a metadata directory.
	mdPath := filepath.Join(params.SIP.Path, "metadata")
	if err := os.MkdirAll(mdPath, 0o700); err != nil {
		return nil, err
	}

	// Move Prozess_Digitalisierung_PREMIS.xml files to the metadata directory.
	err := filepath.WalkDir(params.SIP.ContentPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() == "Prozess_Digitalisierung_PREMIS.xml" {
			// Adding the parent dir to the filename reduces the likelihood of
			// filename conflicts.
			dir := filepath.Base(filepath.Dir(p))
			dest := filepath.Join(mdPath, fmt.Sprintf("Prozess_Digitalisierung_PREMIS_%s.xml", dir))
			err := fsutil.Move(p, dest)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Move UpdatedAreldaMetatdata.xml to the metadata directory (Digitized AIP
	// only)
	if params.SIP.Type == enums.SIPTypeDigitizedAIP {
		err = fsutil.Move(
			params.SIP.UpdatedAreldaMDPath,
			filepath.Join(mdPath, filepath.Base(params.SIP.UpdatedAreldaMDPath)),
		)
		if err != nil {
			return nil, err
		}
	}

	// Create objects and [sip-name] sub-directories.
	objectsPath := filepath.Join(params.SIP.Path, "objects", filepath.Base(params.SIP.Path))
	if err = os.MkdirAll(objectsPath, 0o700); err != nil {
		return nil, err
	}

	// Move the content directory into the objects directory.
	err = fsutil.Move(params.SIP.ContentPath, filepath.Join(objectsPath, "content"))
	if err != nil {
		return nil, err
	}

	// Create a header directory in the objects folder.
	headerPath := filepath.Join(objectsPath, "header")
	if err = os.MkdirAll(headerPath, 0o700); err != nil {
		return nil, err
	}

	// Move the metadata.xml file into the header directory.
	err = fsutil.Move(params.SIP.MetadataPath, filepath.Join(headerPath, filepath.Base(params.SIP.MetadataPath)))
	if err != nil {
		return nil, err
	}

	// Remove the old top-level directories.
	for _, path := range params.SIP.TopLevelPaths {
		if removeErr := os.RemoveAll(path); err != nil {
			err = errors.Join(err, removeErr)
		}
	}
	if err != nil {
		return nil, err
	}

	// Set all the file modes.
	if err = fsutil.SetFileModes(params.SIP.Path, 0o700, 0o600); err != nil {
		return nil, err
	}

	return &TransformSIPResult{PIP: pips.NewFromSIP(params.SIP)}, nil
}
