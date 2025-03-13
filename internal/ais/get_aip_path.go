package ais

import (
	"context"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/amss"
)

const GetAIPPathActivityName = "get-aip-path"

type (
	GetAIPPathActivity struct {
		amssClient amss.Client
	}
	GetAIPPathActivityParams struct {
		AIPUUID string
	}
	GetAIPPathActivityResult struct {
		Path string
	}
)

func NewGetAIPPathActivity(amssClient amss.Client) *GetAIPPathActivity {
	return &GetAIPPathActivity{amssClient: amssClient}
}

func (a *GetAIPPathActivity) Execute(
	ctx context.Context,
	params *GetAIPPathActivityParams,
) (*GetAIPPathActivityResult, error) {
	path, err := a.amssClient.GetAIPPath(ctx, params.AIPUUID)
	if err != nil {
		return nil, err
	}

	return &GetAIPPathActivityResult{Path: path}, nil
}
