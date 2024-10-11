package activities

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

const ValidateMetadataName = "validate-metadata"

type ValidateMetadataParams struct {
	SIP sip.SIP
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
	c := exec.Command("python3", "xsdval.py", params.SIP.ManifestPath, "arelda.xsd") // #nosec G204
	out, err := c.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("run xsdval.py: %s", out)
	}

	// The xsdval.py script always outputs the name "metadata.xml" in the
	// following string regardless of the actual manifest filename.
	if string(out) != "Is metadata.xml valid:  True\n" {
		p, err := filepath.Rel(params.SIP.Path, params.SIP.ManifestPath)
		if err != nil {
			p = params.SIP.ManifestPath
		}
		failures = append(failures, fmt.Sprintf("%q does not match expected metadata requirements", p))
	}
	return &ValidateMetadataResult{Failures: failures}, nil
}
