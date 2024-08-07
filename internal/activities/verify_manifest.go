package activities

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/antchfx/xmlquery"
	goset "github.com/deckarep/golang-set/v2"

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

func (a *VerifyManifest) Execute(ctx context.Context, params *VerifyManifestParams) (*VerifyManifestResult, error) {
	manifestFiles, err := manifestFiles(params.SIP)
	if err != nil {
		return nil, fmt.Errorf("verify manifest: parse manifest: %v", err)
	}

	sipFiles, err := sipFiles(params.SIP.ContentPath)
	if err != nil {
		return nil, fmt.Errorf("verify manifest: get SIP contents: %v", err)
	}

	var failures []string
	for _, p := range manifestFiles.Difference(sipFiles).ToSlice() {
		failures = append(failures, fmt.Sprintf("Missing file: %s", p))
	}
	for _, p := range sipFiles.Difference(manifestFiles).ToSlice() {
		failures = append(failures, fmt.Sprintf("Unexpected file: %s", p))
	}

	return &VerifyManifestResult{Failures: failures}, nil
}

func manifestFiles(s sip.SIP) (goset.Set[string], error) {
	f, err := os.Open(s.ManifestPath)
	if err != nil {
		return nil, fmt.Errorf("open: %v", err)
	}

	doc, err := xmlquery.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("parse document: %v", err)
	}

	contentDir, err := xmlquery.Query(doc, "//paket/inhaltsverzeichnis/ordner/name[text()='content']/..")
	if err != nil || contentDir == nil {
		return nil, fmt.Errorf("missing content path: %v", err)
	}

	dossiers := contentDir.SelectElements("ordner")
	if dossiers == nil {
		return nil, errors.New("no dossiers in content path")
	}

	paths := goset.NewSet[string]()
	for _, d := range dossiers {
		for _, f := range d.SelectElements("datei") {
			paths.Add(filepath.Join(
				d.SelectElement("name").InnerText(),
				f.SelectElement("name").InnerText(),
			))
		}
	}

	return paths, nil
}

func sipFiles(dir string) (goset.Set[string], error) {
	paths := goset.NewSet[string]()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			p, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			paths.Add(p)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}
