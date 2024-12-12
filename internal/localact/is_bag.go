package localact

import (
	"context"
	"path/filepath"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/fsutil"
)

type (
	IsBagParams struct {
		Path string
	}

	IsBagResult struct {
		IsBag bool
	}
)

func IsBag(ctx context.Context, params *IsBagParams) (*IsBagResult, error) {
	return &IsBagResult{
		IsBag: fsutil.FileExists(filepath.Join(params.Path, "bagit.txt")),
	}, nil
}
