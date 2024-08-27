package ais

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"go.artefactual.dev/tools/temporal"
)

const FetchActivityName = "fetch-amss-file"

type (
	FetchActivityParams struct {
		AIPUUID      uuid.UUID
		RelativePath string
		Destination  string
	}
	FetchActivityResult struct{}
	FetchActivity       struct {
		amssclient *AMSSClient
	}
)

func NewFetchActivity(amssclient *AMSSClient) *FetchActivity {
	return &FetchActivity{amssclient: amssclient}
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

	err = a.amssclient.DownloadAIPFile(ctx, params.AIPUUID.String(), params.RelativePath, file)
	if err != nil {
		return nil, fmt.Errorf("FetchActivity: download file: %w", err)
	}

	return &FetchActivityResult{}, nil
}
