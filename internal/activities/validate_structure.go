package activities

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/fsutil"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

const ValidateStructureName = "validate-structure"

type ValidateStructureParams struct {
	SIP sip.SIP
}

type ValidateStructureResult struct {
	Failures []string
}

type ValidateStructure struct{}

func NewValidateStructure() *ValidateStructure {
	return &ValidateStructure{}
}

func (a *ValidateStructure) Execute(
	ctx context.Context,
	params *ValidateStructureParams,
) (*ValidateStructureResult, error) {
	var failures []string

	// Check existence of the content directory.
	hasContentDir := true
	if !fsutil.FileExists(params.SIP.ContentPath) {
		failures = append(failures, "Content folder is missing")
		hasContentDir = false
	}

	// Check existence of the XSD directory.
	if !fsutil.FileExists(params.SIP.XSDPath) {
		failures = append(failures, "XSD folder is missing")
	}

	// Check existence of metadata file.
	if !fsutil.FileExists(params.SIP.MetadataPath) {
		failures = append(failures, fmt.Sprintf(
			"%s is missing", filepath.Base(params.SIP.MetadataPath),
		))
	}

	// Check existence of UpdatedAreldaMetadata file (AIPs only).
	if params.SIP.IsAIP() && !fsutil.FileExists(params.SIP.UpdatedAreldaMDPath) {
		failures = append(failures, fmt.Sprintf(
			"%s is missing", filepath.Base(params.SIP.UpdatedAreldaMDPath),
		))
	}

	// Check existence of logical metadata file (AIPs only).
	if params.SIP.IsAIP() && !fsutil.FileExists(params.SIP.LogicalMDPath) {
		failures = append(failures, fmt.Sprintf("%s is missing", filepath.Base(params.SIP.LogicalMDPath)))
	}

	// Check for unexpected top-level directories.
	sipBase := params.SIP.Path
	extras, err := extraNodes(sipBase, params.SIP.Path, params.SIP.TopLevelPaths, true)
	if err != nil {
		return nil, fmt.Errorf("ValidateStructure: check for unexpected dirs: %v", err)
	}
	failures = append(failures, extras...)

	// Check for unexpected files in the content directory.
	if hasContentDir {
		extras, err := extraNodes(sipBase, params.SIP.ContentPath, []string{}, false)
		if err != nil {
			return nil, fmt.Errorf("ValidateStructure: check for unexpected files: %v", err)
		}
		failures = append(failures, extras...)
	}

	// Check that digitized packages only have one dossier in the content dir.
	if params.SIP.Type == enums.SIPTypeDigitizedSIP || params.SIP.Type == enums.SIPTypeDigitizedAIP && hasContentDir {
		entries, err := os.ReadDir(params.SIP.ContentPath)
		if err != nil {
			return nil, fmt.Errorf("ValidateStructure: check for unexpected dossiers: %v", err)
		}

		dirs := 0
		for _, e := range entries {
			if e.IsDir() {
				dirs += 1
			}
			if dirs > 1 {
				break
			}
		}

		if dirs > 1 {
			failures = append(failures, "More than one dossier in the content directory")
		}
	}

	return &ValidateStructureResult{Failures: failures}, nil
}

func extraNodes(sipBase, path string, expected []string, matchDir bool) ([]string, error) {
	var extras []string

	ftype := "file"
	if matchDir {
		ftype = "directory"
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("can't read directory: %v", err)
	}

	for _, entry := range entries {
		fp := filepath.Join(path, entry.Name())
		if entry.IsDir() == matchDir && !slices.Contains(expected, fp) {
			rel, err := filepath.Rel(sipBase, fp)
			if err != nil {
				return nil, fmt.Errorf("can't determine relative path: %v", err)
			}
			rel = filepath.Join(filepath.Base(sipBase), rel)
			extras = append(extras, fmt.Sprintf("Unexpected %s: %q", ftype, rel))
		}
	}

	return extras, nil
}
