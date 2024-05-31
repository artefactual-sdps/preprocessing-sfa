package activities

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"go.artefactual.dev/tools/fsutil"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

const TransformSIPName = "transform-sip"

type TransformSIPParams struct {
	SIP sip.SIP
}

type TransformSIPResult struct{}

type TransformSIP struct{}

func NewTransformSIP() *TransformSIP {
	return &TransformSIP{}
}

func (a *TransformSIP) Execute(ctx context.Context, params *TransformSIPParams) (*TransformSIPResult, error) {
	// Create metadata directory.
	mdPath := filepath.Join(params.SIP.Path, "metadata")
	if err := os.MkdirAll(mdPath, 0o700); err != nil {
		return nil, err
	}

	// Move metadata file.
	err := fsutil.Move(params.SIP.MetadataPath, filepath.Join(mdPath, filepath.Base(params.SIP.MetadataPath)))
	if err != nil {
		return nil, err
	}

	// Move Prozess_Digitalisierung_PREMIS.xml files.
	err = filepath.WalkDir(params.SIP.ContentPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() == "Prozess_Digitalisierung_PREMIS.xml" {
			// Adding the parent dir to the filename reduces the likelihood of filename conflicts.
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

	// Create objects directory.
	objectsPath := filepath.Join(params.SIP.Path, "objects")
	if err = os.MkdirAll(objectsPath, 0o700); err != nil {
		return nil, err
	}

	// Move all entries from content to objects folder.
	entries, err := os.ReadDir(params.SIP.ContentPath)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		err := fsutil.Move(
			filepath.Join(params.SIP.ContentPath, entry.Name()),
			filepath.Join(objectsPath, entry.Name()),
		)
		if err != nil {
			return nil, err
		}
	}

	// Remove previous top-level directories.
	for _, path := range params.SIP.TopLevelPaths {
		if removeErr := os.RemoveAll(path); err != nil {
			err = errors.Join(err, removeErr)
		}
	}
	if err != nil {
		return nil, err
	}

	return &TransformSIPResult{}, nil
}
