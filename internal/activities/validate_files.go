package activities

import (
	"context"
	"fmt"
	"slices"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/fformat"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/fvalidate"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

const ValidateFilesName = "validate-files"

type (
	ValidateFiles struct {
		identifier fformat.Identifier
		validators []fvalidate.Validator
	}
	ValidateFilesParams struct {
		SIP sip.SIP
	}
	ValidateFilesResult struct {
		Failures []string
	}
)

func NewValidateFiles(idr fformat.Identifier, vdrs ...fvalidate.Validator) *ValidateFiles {
	return &ValidateFiles{
		identifier: idr,
		validators: vdrs,
	}
}

// Execute validates SIP files against a file format specification. The
// only format validator currently implemented verapdf for PDF/A.
func (a *ValidateFiles) Execute(ctx context.Context, params *ValidateFilesParams) (*ValidateFilesResult, error) {
	formats, err := fformat.IdentifyFormats(ctx, a.identifier, params.SIP)
	if err != nil {
		return nil, fmt.Errorf("identifyFormats: %v", err)
	}

	failures, err := a.validateFiles(params.SIP, formats)
	if err != nil {
		return nil, fmt.Errorf("validateFiles: %v", err)
	}

	return &ValidateFilesResult{Failures: failures}, nil
}

func (a *ValidateFiles) validateFiles(
	sip sip.SIP,
	files fformat.FileFormats,
) ([]string, error) {
	var failures []string
	for _, v := range a.validators {
		out, err := validate(v, sip.ContentPath, files)
		if err != nil {
			return nil, err
		}
		if out != "" {
			failures = append(failures, out)
		}
	}

	return failures, nil
}

func validate(v fvalidate.Validator, path string, files fformat.FileFormats) (string, error) {
	var canValidate bool
	allowedIds := v.FormatIDs()

	for _, f := range files {
		if slices.Contains(allowedIds, f.ID) {
			canValidate = true
			break
		}
	}

	if !canValidate {
		return "", nil
	}

	out, err := v.Validate(path)
	if err != nil {
		return "", err
	}

	return out, nil
}
