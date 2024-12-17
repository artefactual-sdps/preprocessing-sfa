package activities

import (
	"context"
	"crypto/md5"    // #nosec: 501 -- not used for security.
	"crypto/sha1"   // #nosec: 501 -- not used for security.
	"crypto/sha256" // #nosec: 501 -- not used for security.
	"crypto/sha512" // #nosec: 501 -- not used for security.
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	goset "github.com/deckarep/golang-set/v2"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/manifest"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

const VerifyManifestName = "verify-manifest"

type (
	VerifyManifest       struct{}
	VerifyManifestParams struct {
		SIP sip.SIP
	}
	VerifyManifestResult struct {
		Failed           bool
		ChecksumFailures []string
		MissingFiles     []string
		UnexpectedFiles  []string
	}
)

func NewVerifyManifest() *VerifyManifest {
	return &VerifyManifest{}
}

// Execute parses a SIP's manifest and verifies it against the actual files in
// the SIP directory. Any missing or unexpected files on disk are reported as
// failures.
func (a *VerifyManifest) Execute(ctx context.Context, params *VerifyManifestParams) (*VerifyManifestResult, error) {
	manifestFiles, err := manifestFiles(params.SIP)
	if err != nil {
		return nil, fmt.Errorf("verify manifest: parse manifest: %v", err)
	}
	manifestSet := goset.NewSetFromMapKeys(manifestFiles)

	sipFiles, err := sipFiles(params.SIP)
	if err != nil {
		return nil, fmt.Errorf("verify manifest: get SIP contents: %v", err)
	}

	badChecksums, err := verifyChecksums(manifestFiles, sipFiles, params.SIP.Path)
	if err != nil {
		return nil, fmt.Errorf("verify checksums: %v", err)
	}

	sipBase := filepath.Base(params.SIP.Path)
	missing := missingFiles(sipBase, manifestSet, sipFiles)
	unexpected := unexpectedFiles(sipBase, manifestSet, sipFiles)

	return &VerifyManifestResult{
		Failed:           len(missing) > 0 || len(unexpected) > 0 || len(badChecksums) > 0,
		ChecksumFailures: badChecksums,
		MissingFiles:     missing,
		UnexpectedFiles:  unexpected,
	}, nil
}

// manifestFiles parses the SIP manifest and returns a map of file paths
// (relative to the SIP root directory) to files.
func manifestFiles(s sip.SIP) (map[string]*manifest.File, error) {
	f, err := os.Open(s.ManifestPath)
	if err != nil {
		return nil, fmt.Errorf("open: %v", err)
	}

	files, err := manifest.Files(f)
	if err != nil {
		return nil, err
	}

	// Prefix "content/" to AIP file paths.
	if s.IsAIP() {
		m := make(map[string]*manifest.File, len(files))
		for k, v := range files {
			m[filepath.Join("content", k)] = v
		}
		files = m
	}

	return files, nil
}

// sipFiles recursively walks dir's tree and returns the set of all file
// (excluding directory) paths found.
func sipFiles(s sip.SIP) (goset.Set[string], error) {
	root := s.Path
	if s.IsAIP() {
		root = filepath.Join(s.Path, "content")
	}

	paths := goset.NewSet[string]()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		p, err := filepath.Rel(s.Path, path)
		if err != nil {
			return err
		}

		// SIPs don't include metadata.xml in the manifest, so ignore the file
		// here.
		if s.IsSIP() && p == "header/metadata.xml" {
			return nil
		}

		paths.Add(p)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

// missingFiles returns the list of all files that are in manifest but not
// filesys.
func missingFiles(base string, manifest, filesys goset.Set[string]) []string {
	var missing []string
	if s := manifest.Difference(filesys).ToSlice(); len(s) > 0 {
		slices.Sort(s)
		for _, p := range s {
			fp := filepath.Join(base, p)
			missing = append(missing, fmt.Sprintf("Missing file: %s", fp))
		}
	}
	return missing
}

// unexpectedFiles returns the list of all files that are in filesys but not
// manifest.
func unexpectedFiles(base string, manifest, filesys goset.Set[string]) []string {
	var unexpected []string
	if s := filesys.Difference(manifest).ToSlice(); len(s) > 0 {
		slices.Sort(s)
		for _, p := range s {
			fp := filepath.Join(base, p)
			unexpected = append(unexpected, fmt.Sprintf("Unexpected file: %s", fp))
		}
	}
	return unexpected
}

// verifyChecksums checks that each manifestFiles file checksum matches the
// checksum generated from the actual file contents. If a file is on the
// manifest but missing from the filesystem, or vice versa, it will be skipped
// with no validation message.  The root is the absolute path to the root
// directory of the SIP, and is prefixed to each relative file path in the
// manifest to create an absolute path the file.
func verifyChecksums(
	manifestFiles map[string]*manifest.File,
	sipFiles goset.Set[string],
	root string,
) ([]string, error) {
	var failures []string

	for path, file := range manifestFiles {
		// Check if file exists on filesystem.
		if !sipFiles.Contains(path) {
			continue
		}

		// Attempt to generate hash from filesystem file contents.
		hashResult, err := generateHash(filepath.Join(root, path), file.Checksum.Algorithm)
		if err != nil {
			return nil, err
		}

		// Compare hash to expected value.
		if hashResult != file.Checksum.Hash {
			failures = append(
				failures,
				fmt.Sprintf(
					"Checksum mismatch for %q (expected: %q, got: %q)",
					path,
					file.Checksum.Hash,
					hashResult,
				),
			)
		}
	}
	slices.Sort(failures)

	return failures, nil
}

// Return a hexadecimal encoded hash string generated from the contents
// of the file at path.
func generateHash(path, alg string) (string, error) {
	var h hash.Hash

	switch alg {
	case "MD5":
		h = md5.New() // #nosec: G401 -- not used for security.
	case "SHA-1":
		h = sha1.New() // #nosec: G401 -- not used for security.
	case "SHA-256":
		h = sha256.New() // #nosec: G401 -- not used for security.
	case "SHA-512":
		h = sha512.New() // #nosec: G401 -- not used for security.
	default:
		return "", fmt.Errorf("hash algorithm %q is not supported", alg)
	}

	f, err := os.Open(path) // #nosec: G304 -- trusted path.
	if err != nil {
		return "", fmt.Errorf("open file: %v", err)
	}
	defer f.Close()

	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("copy contents: %v", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
