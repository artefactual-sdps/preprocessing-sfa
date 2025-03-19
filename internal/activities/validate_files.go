package activities

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"

	"go.artefactual.dev/tools/temporal"

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

type fileFormats map[string]*fformat.FileFormat

func NewValidateFiles(idr fformat.Identifier, vdrs ...fvalidate.Validator) *ValidateFiles {
	return &ValidateFiles{
		identifier: idr,
		validators: vdrs,
	}
}

// Execute validates SIP files against a file format specification. The
// only format validator currently implemented verapdf for PDF/A.
func (a *ValidateFiles) Execute(ctx context.Context, params *ValidateFilesParams) (*ValidateFilesResult, error) {
	formats, err := a.identifyFormats(ctx, params.SIP)
	if err != nil {
		return nil, fmt.Errorf("identifyFormats: %v", err)
	}

	failures, err := a.validateFiles(params.SIP, formats)
	if err != nil {
		return nil, fmt.Errorf("validateFiles: %v", err)
	}

	return &ValidateFilesResult{Failures: failures}, nil
}

func (a *ValidateFiles) identifyFormats(ctx context.Context, sip sip.SIP) (fileFormats, error) {
	logger := temporal.GetLogger(ctx)
	formats := make(fileFormats)
	err := filepath.WalkDir(sip.ContentPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if ctx.Err() != nil {
			return errors.New("context cancelled")
		}

		if d.IsDir() {
			return nil
		}

		ff, err := a.identifier.Identify(path)
		if err != nil {
			logger.Info("format identification failed", "path", path)
		} else {
			formats[path] = ff
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return formats, nil
}

func (a *ValidateFiles) validateFiles(
	sip sip.SIP,
	files fileFormats,
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

func validate(v fvalidate.Validator, path string, files fileFormats) (string, error) {
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
