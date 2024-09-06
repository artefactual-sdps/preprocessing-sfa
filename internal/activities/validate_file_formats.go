package activities

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/fformat"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

const ValidateFileFormatsName = "validate-file-formats"

type ValidateFileFormatsParams struct {
	SIP sip.SIP
}

type ValidateFileFormatsResult struct {
	Failures []string
}

type ValidateFileFormats struct{}

func NewValidateFileFormats() *ValidateFileFormats {
	return &ValidateFileFormats{}
}

func (a *ValidateFileFormats) Execute(
	ctx context.Context,
	params *ValidateFileFormatsParams,
) (*ValidateFileFormatsResult, error) {
	var failures []string

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

	err := filepath.WalkDir(params.SIP.ContentPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		ff, err := sf.Identify(p)
		if err != nil {
			return fmt.Errorf("identify format: %v", err)
		}

		if _, ok := allowed[ff.ID]; !ok {
			failures = append(failures, fmt.Sprintf("file format %q not allowed: %q", ff.ID, p))
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("ValidateFileFormats: %v", err)
	}

	return &ValidateFileFormatsResult{Failures: failures}, nil
}
