package activities

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

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

	// Check for empty directories and invalid (Archivematica incompatible) file/directory names.
	paths := make(map[string]int)

	err := filepath.WalkDir(params.SIP.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(params.SIP.Path, path)
		if err != nil {
			return err
		}

		if !validateName(d.Name()) {
			failures = append(failures, fmt.Sprintf("Name %q contains invalid character", relativePath))
		}

		if path != params.SIP.Path {
			// Initialize this directory's total number of immediate children.
			if d.IsDir() {
				paths[relativePath] = 0
			}

			// Add to parent's total number of immediate children.
			paths[filepath.Dir(relativePath)] = paths[filepath.Dir(relativePath)] + 1
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("ValidateStructure: check for empty directories: %v", err)
	}

	// Report any empty subdirectories.
	emptyDirFound := false

	for path, children := range paths {
		if children == 0 {
			failures = append(failures, fmt.Sprintf("An empty directory has been found - %s", path))
			emptyDirFound = true
		}
	}

	if emptyDirFound {
		failures = append(
			failures,
			"Please remove the empty directories and update the metadata manifest accordingly",
		)
	}

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

// validateName makes sure only valid characters exist in name.
func validateName(name string) bool {
	const validChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_.()"

	for i := range len(name) {
		if !strings.Contains(validChars, string(name[i])) {
			return false
		}
	}

	return true
}
