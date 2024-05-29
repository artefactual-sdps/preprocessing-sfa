package activities

import (
	"context"
	"errors"
	"os/exec"
)

const ValidateMetadataName = "validate-metadata"

type ValidateMetadataParams struct {
	MetadataPath string
}

type ValidateMetadataResult struct {
	Out string
}

type ValidateMetadata struct{}

func NewValidateMetadata() *ValidateMetadata {
	return &ValidateMetadata{}
}

func (a *ValidateMetadata) Execute(
	ctx context.Context,
	params *ValidateMetadataParams,
) (*ValidateMetadataResult, error) {
	e, err := exec.Command("python3", "xsdval.py", params.MetadataPath, "arelda.xsd").CombinedOutput() // #nosec G204
	if err != nil {
		return nil, err
	}

	if string(e) != "Is metadata.xml valid:  True\n" {
		return nil, errors.New("Failed to validate metadata files: " + string(e))
	}
	return &ValidateMetadataResult{}, nil
}
