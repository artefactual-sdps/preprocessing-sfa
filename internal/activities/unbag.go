package activities

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.artefactual.dev/tools/fsutil"
)

const UnbagName = "unbag"

type (
	Unbag struct{}

	UnbagParams struct {
		Path string
	}

	UnbagResult struct {
		Path string
	}
)

func NewUnbag() *Unbag {
	return &Unbag{}
}

func (a *Unbag) Execute(ctx context.Context, params *UnbagParams) (*UnbagResult, error) {
	if _, err := os.Stat(filepath.Join(params.Path, "bagit.txt")); err != nil {
		// Do nothing if not a bag (bagit.txt doesn't exist).
		return &UnbagResult{Path: params.Path}, nil
	}
	if _, err := os.Stat(filepath.Join(params.Path, "data")); err != nil {
		return nil, errors.New("missing data directory")
	}

	entries, err := os.ReadDir(params.Path)
	if err != nil {
		return nil, fmt.Errorf("read dir: %v", err)
	}

	// Delete everything except the data directory.
	for _, e := range entries {
		if e.Name() != "data" {
			if err := os.RemoveAll(filepath.Join(params.Path, e.Name())); err != nil {
				return nil, fmt.Errorf("delete: %v", err)
			}
		}
	}

	// Move the data directory contents to the SIP root.
	entries, err = os.ReadDir(filepath.Join(params.Path, "data"))
	if err != nil {
		return nil, fmt.Errorf("read data dir: %v", err)
	}

	for _, e := range entries {
		if err = fsutil.Move(
			filepath.Join(params.Path, "data", e.Name()),
			filepath.Join(params.Path, e.Name()),
		); err != nil {
			return nil, fmt.Errorf("move: %v", err)
		}
	}

	// Delete the empty data directory.
	if err := os.Remove(filepath.Join(params.Path, "data")); err != nil {
		return nil, fmt.Errorf("remove data dir: %v", err)
	}

	return &UnbagResult{Path: params.Path}, nil
}
