package ais

import "context"

type GetAIPPathActivityParams struct {
	AMSSClient *AMSSClient
	AIPUUID    string
}

type GetAIPPathActivityResult struct {
	Path string
}

func GetAIPPathActivity(ctx context.Context, params *GetAIPPathActivityParams) (*GetAIPPathActivityResult, error) {
	path, err := params.AMSSClient.GetAIPPath(ctx, params.AIPUUID)
	if err != nil {
		return nil, err
	}
	return &GetAIPPathActivityResult{Path: path}, nil
}
