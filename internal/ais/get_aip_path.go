package ais

import (
	"context"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/amss"
)

const GetAIPPathActivityName = "get-aip-path"

type (
	GetAIPPathActivity struct {
		amssSvc amss.Service
	}
	GetAIPPathActivityParams struct {
		AIPUUID string
	}
	GetAIPPathActivityResult struct {
		Path string
	}
)

func NewGetAIPPathActivity(amssSvc amss.Service) *GetAIPPathActivity {
	return &GetAIPPathActivity{amssSvc: amssSvc}
}

func (a *GetAIPPathActivity) Execute(
	ctx context.Context,
	params *GetAIPPathActivityParams,
) (*GetAIPPathActivityResult, error) {
	path, err := a.amssSvc.GetAIPPath(ctx, params.AIPUUID)
	if err != nil {
		return nil, err
	}

	return &GetAIPPathActivityResult{Path: path}, nil
}
