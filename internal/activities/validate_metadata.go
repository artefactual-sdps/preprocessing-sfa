package activities

import (
	"context"
	"fmt"
	"os/exec"
)

const ValidateMetadataName = "validate-metadata"

type ValidateMetadataParams struct {
	MetadataPath string
}

type ValidateMetadataResult struct {
	Failures []string
}

type ValidateMetadata struct{}

func NewValidateMetadata() *ValidateMetadata {
	return &ValidateMetadata{}
}

func (a *ValidateMetadata) Execute(
	ctx context.Context,
	params *ValidateMetadataParams,
) (*ValidateMetadataResult, error) {
	var failures []string
	e, err := exec.Command("python3", "xsdval.py", params.MetadataPath, "arelda.xsd").CombinedOutput() // #nosec G204
	if err != nil {
		return nil, err
	}

	if string(e) != "Is metadata.xml valid:  True\n" {
		failures = append(failures, fmt.Sprintf(
			"%s does not match expected metadata requirements",
			params.MetadataPath,
		))
	}
	return &ValidateMetadataResult{Failures: failures}, nil
}
