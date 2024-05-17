package activities

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/fsutil"
)

const TransformVecteurAIPName = "transform-vecteur-aip"

type TransformVecteurAIPActivity struct{}

func NewTransformVecteurAIPActivity() *TransformVecteurAIPActivity {
	return &TransformVecteurAIPActivity{}
}

type TransformVecteurAIPParams struct {
	Path string
}

type TransformVecteurAIPResult struct{}

func (a *TransformVecteurAIPActivity) Execute(
	ctx context.Context,
	params *TransformVecteurAIPParams,
) (*TransformVecteurAIPResult, error) {
	// Rename additional folder to metadata.
	mdPath := filepath.Join(params.Path, "metadata")
	err := fsutil.Move(filepath.Join(params.Path, "additional"), mdPath)
	if err != nil {
		return nil, err
	}

	// Move Prozess_Digitalisierung_PREMIS.xml files to metadata folder.
	contentPath := filepath.Join(params.Path, "content", "content")
	err = filepath.WalkDir(contentPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() == "Prozess_Digitalisierung_PREMIS.xml" {
			// Adding the parent dir to the filename reduces the likelihood of filename
			// conflicts and prevents deletion when all PREMIS files are deleted after.
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
	objectsPath := filepath.Join(params.Path, "objects")
	if err = os.Mkdir(objectsPath, 0o750); err != nil {
		return nil, err
	}

	// Move all entries from content/content to root folder.
	entries, err := os.ReadDir(contentPath)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		err := fsutil.Move(filepath.Join(contentPath, entry.Name()), filepath.Join(objectsPath, entry.Name()))
		if err != nil {
			return nil, err
		}
	}

	// Remove content folders.
	err = os.RemoveAll(filepath.Join(params.Path, "content"))
	if err != nil {
		return nil, err
	}

	return &TransformVecteurAIPResult{}, nil
}
