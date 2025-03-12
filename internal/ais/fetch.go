package ais

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/amss"
	"go.artefactual.dev/tools/temporal"
)

const FetchActivityName = "fetch-amss-file"

type (
	FetchActivityParams struct {
		AIPUUID      string
		RelativePath string
		Destination  string
	}
	FetchActivityResult struct{}
	FetchActivity       struct {
		amssSvc amss.Service
	}
)

func NewFetchActivity(amssclient amss.Service) *FetchActivity {
	return &FetchActivity{amssSvc: amssclient}
}

func (a *FetchActivity) Execute(ctx context.Context, params *FetchActivityParams) (*FetchActivityResult, error) {
	h := temporal.StartAutoHeartbeat(ctx)
	defer h.Stop()

	if err := os.MkdirAll(filepath.Dir(params.Destination), 0o700); err != nil {
		return nil, fmt.Errorf("FetchActivity: create directory: %w", err)
	}

	file, err := os.OpenFile(params.Destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600) // #nosec G304
	if err != nil {
		return nil, fmt.Errorf("FetchActivity: create file: %w", err)
	}
	defer file.Close()

	err = a.amssSvc.DownloadAIPFile(ctx, params.AIPUUID, params.RelativePath, file)
	if err != nil {
		return nil, fmt.Errorf("FetchActivity: download file: %w", err)
	}

	return &FetchActivityResult{}, nil
}
