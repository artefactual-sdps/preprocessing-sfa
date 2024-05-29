package activities

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

const ValidateStructureName = "validate-structure"

type ValidateStructureParams struct {
	SIP sip.SIP
}

type ValidateStructureResult struct{}

type ValidateStructure struct{}

func NewValidateStructure() *ValidateStructure {
	return &ValidateStructure{}
}

func (a *ValidateStructure) Execute(
	ctx context.Context,
	params *ValidateStructureParams,
) (*ValidateStructureResult, error) {
	var e error

	if _, err := os.Stat(params.SIP.ContentPath); err != nil {
		e = errors.Join(e, fmt.Errorf("content folder: %v", err))
	}
	if _, err := os.Stat(params.SIP.MetadataPath); err != nil {
		e = errors.Join(e, fmt.Errorf("metadata file: %v", err))
	}
	if _, err := os.Stat(params.SIP.XSDPath); err != nil {
		e = errors.Join(e, fmt.Errorf("XSD file: %v", err))
	}

	entries, err := os.ReadDir(params.SIP.ContentPath)
	if err != nil {
		e = errors.Join(e, fmt.Errorf("read content folder: %v", err))
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			e = errors.Join(e, fmt.Errorf(
				"unexpected file: %q",
				filepath.Join(params.SIP.ContentPath, entry.Name()),
			))
		}
	}

	if e != nil {
		return nil, e
	}

	return &ValidateStructureResult{}, nil
}
