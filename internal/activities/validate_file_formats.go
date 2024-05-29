package activities

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/fformat"
)

const ValidateFileFormatsName = "validate-file-formats"

type ValidateFileFormatsParams struct {
	ContentPath string
}

type ValidateFileFormatsResult struct{}

type ValidateFileFormats struct{}

func NewValidateFileFormats() *ValidateFileFormats {
	return &ValidateFileFormats{}
}

func (a *ValidateFileFormats) Execute(
	ctx context.Context,
	params *ValidateFileFormatsParams,
) (*ValidateFileFormatsResult, error) {
	sf := fformat.NewSiegfriedEmbed()
	// TODO(daniel): make allowed list configurable.
	allowed := map[string]struct{}{
		"fmt/95":    {},
		"x-fmt/16":  {},
		"x-fmt/21":  {},
		"x-fmt/22":  {},
		"x-fmt/62":  {},
		"x-fmt/111": {},
		"x-fmt/282": {},
		"x-fmt/283": {},
		"fmt/354":   {},
		"fmt/476":   {},
		"fmt/477":   {},
		"fmt/478":   {},
		"x-fmt/18":  {},
		"fmt/161":   {},
		"fmt/1196":  {},
		"fmt/1777":  {},
		"fmt/353":   {},
		"x-fmt/392": {},
		"fmt/1":     {},
		"fmt/2":     {},
		"fmt/6":     {},
		"fmt/141":   {},
		"fmt/569":   {},
		"fmt/199":   {},
		"fmt/101":   {},
		"fmt/142":   {},
		"x-fmt/280": {},
		"fmt/1014":  {},
		"fmt/1012":  {},
		"fmt/654":   {},
		"fmt/1013":  {},
		"fmt/1011":  {},
		"fmt/653":   {},
	}

	var invalidErrors error

	err := filepath.WalkDir(params.ContentPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ff, err := sf.Identify(p)
		if err != nil {
			return err
		}
		if _, exists := allowed[ff.ID]; !exists {
			invalidErrors = errors.Join(
				invalidErrors,
				fmt.Errorf("file format not allowed %q for file %q", ff.ID, p),
			)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("check content file formats: %v", err)
	}

	if invalidErrors != nil {
		return nil, invalidErrors
	}

	return &ValidateFileFormatsResult{}, nil
}
