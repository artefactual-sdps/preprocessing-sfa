package amss

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.artefactual.dev/ssclient"
)

const GetAIPPathActivityName = "get-aip-path"

type (
	GetAIPPathActivity struct {
		packages *ssclient.PackagesService
	}
	GetAIPPathActivityParams struct {
		AIPUUID uuid.UUID
	}
	GetAIPPathActivityResult struct {
		Path string
	}
)

func NewGetAIPPathActivity(packages *ssclient.PackagesService) *GetAIPPathActivity {
	return &GetAIPPathActivity{packages: packages}
}

func (a *GetAIPPathActivity) Execute(
	ctx context.Context,
	params *GetAIPPathActivityParams,
) (*GetAIPPathActivityResult, error) {
	pkg, err := a.packages.Get(ctx, params.AIPUUID)
	if err != nil {
		return nil, fmt.Errorf("GetAIPPath: get package: %w", err)
	}
	if pkg == nil {
		return nil, fmt.Errorf("GetAIPPath: package not found")
	}

	path := pkg.GetCurrentPath()
	if path == nil {
		return nil, fmt.Errorf("GetAIPPath: current_path not found in response")
	}

	return &GetAIPPathActivityResult{Path: *path}, nil
}
