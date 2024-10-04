package activities

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"go.artefactual.dev/tools/temporal"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/fformat"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

const ValidateFileFormatsName = "validate-file-formats"

type (
	ValidateFileFormats struct {
		cfg fformat.Config
	}
	ValidateFileFormatsParams struct {
		SIP sip.SIP
	}
	ValidateFileFormatsResult struct {
		Failures []string
	}
)

type formatList map[string]struct{}

func NewValidateFileFormats(cfg fformat.Config) *ValidateFileFormats {
	return &ValidateFileFormats{cfg: cfg}
}

func (a *ValidateFileFormats) Execute(
	ctx context.Context,
	params *ValidateFileFormatsParams,
) (*ValidateFileFormatsResult, error) {
	var failures []string
	logger := temporal.GetLogger(ctx)

	if a.cfg.AllowlistPath == "" {
		logger.Info("ValidateFileFormats: No file format allowlist path set, skipping file format validation")

		return nil, nil
	}

	f, err := os.Open(a.cfg.AllowlistPath)
	if err != nil {
		return nil, fmt.Errorf("ValidateFileFormats: %v", err)
	}
	defer f.Close()

	allowed, err := parseFormatList(f)
	if err != nil {
		return nil, fmt.Errorf("ValidateFileFormats: load allowed formats: %v", err)
	}

	sf := fformat.NewSiegfriedEmbed()
	err = filepath.WalkDir(params.SIP.ContentPath, func(p string, d fs.DirEntry, err error) error {
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

func parseFormatList(r io.Reader) (formatList, error) {
	var i int
	formats := make(formatList)

	cr := csv.NewReader(r)
	for {
		row, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("invalid CSV: %v", err)
		}

		// Skip the first row.
		if i > 0 {
			// Get the file format identifier from the first column of each row
			// and ignore subsequent columns.
			formats[row[0]] = struct{}{}
		}

		i++
	}

	if len(formats) == 0 {
		return nil, fmt.Errorf("no allowed file formats")
	}

	return formats, nil
}
