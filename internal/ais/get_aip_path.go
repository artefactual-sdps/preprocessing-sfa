package ais

import (
	"context"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/amss"
)

const GetAIPPathActivityName = "get-aip-path"

type (
	GetAIPPathActivity struct {
		AMSSClient amss.Service
	}
	GetAIPPathActivityParams struct {
		AIPUUID string
	}
	GetAIPPathActivityResult struct {
		Path string
	}
)

func NewGetAIPPathActivity(amssClient amss.Service) *GetAIPPathActivity {
	return &GetAIPPathActivity{AMSSClient: amssClient}
}

func (a *GetAIPPathActivity) Execute(
	ctx context.Context,
	params *GetAIPPathActivityParams,
) (*GetAIPPathActivityResult, error) {
	path, err := a.AMSSClient.GetAIPPath(ctx, params.AIPUUID)
	if err != nil {
		return nil, err
	}

	return &GetAIPPathActivityResult{Path: path}, nil
}
