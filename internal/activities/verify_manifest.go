package activities

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/antchfx/xmlquery"
	goset "github.com/deckarep/golang-set/v2"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

const VerifyManifestName = "verify-manifest"

type (
	VerifyManifest       struct{}
	VerifyManifestParams struct {
		SIP sip.SIP
	}
	VerifyManifestResult struct {
		Failures []string
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

	sipFiles, err := sipFiles(params.SIP)
	if err != nil {
		return nil, fmt.Errorf("verify manifest: get SIP contents: %v", err)
	}

	var failures []string

	if s := manifestFiles.Difference(sipFiles).ToSlice(); len(s) > 0 {
		slices.Sort(s)
		for _, p := range s {
			failures = append(failures, fmt.Sprintf("Missing file: %s", p))
		}
	}

	if s := sipFiles.Difference(manifestFiles).ToSlice(); len(s) > 0 {
		slices.Sort(s)
		for _, p := range s {
			failures = append(failures, fmt.Sprintf("Unexpected file: %s", p))
		}
	}

	return &VerifyManifestResult{Failures: failures}, nil
}

// manifestFiles returns the set of all files paths listed in a SIP's manifest.
func manifestFiles(s sip.SIP) (goset.Set[string], error) {
	f, err := os.Open(s.ManifestPath)
	if err != nil {
		return nil, fmt.Errorf("open: %v", err)
	}

	doc, err := xmlquery.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("parse document: %v", err)
	}

	manifest, err := xmlquery.Query(doc, "//paket/inhaltsverzeichnis")
	if err != nil || manifest == nil {
		return nil, fmt.Errorf("missing inhaltsverzeichnis entry: %v", err)
	}

	root := ""
	if s.Type == enums.SIPTypeDigitizedAIP {
		root = "content"
	}

	return walkNode(manifest, root), nil
}

// walkNode recursively walks node's xpath tree and returns the set of all file
// (excluding directories) paths found.
func walkNode(node *xmlquery.Node, path string) goset.Set[string] {
	paths := goset.NewSet[string]()

	for _, n := range node.SelectElements("ordner") {
		name := n.SelectElement("name").InnerText()
		paths = paths.Union(walkNode(n, filepath.Join(path, name)))
	}

	for _, n := range node.SelectElements("datei") {
		name := n.SelectElement("name").InnerText()
		paths.Add(filepath.Join(path, name))
	}

	return paths
}

// sipFiles recursively walks dir's tree and returns the set of all file
// (excluding directories) paths found.
func sipFiles(s sip.SIP) (goset.Set[string], error) {
	root := s.Path
	if s.Type == enums.SIPTypeDigitizedAIP {
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

		// Digitized SIP and born-digital SIPs don't include metadata.xml in the
		// manifest, so ignore the file here.
		if s.Type != enums.SIPTypeDigitizedAIP && p == "header/metadata.xml" {
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
