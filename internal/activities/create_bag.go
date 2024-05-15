package activities

import (
	"context"

	go_bagit "github.com/nyudlts/go-bagit"
)

const CreateBagName = "create-bag"

type CreateBagActivity struct{}

func NewCreateBagActivity() *CreateBagActivity {
	return &CreateBagActivity{}
}

type CreateBagParams struct {
	Path string
}

type CreateBagResult struct{}

func (a *CreateBagActivity) Execute(
	ctx context.Context,
	params *CreateBagParams,
) (*CreateBagResult, error) {
	_, err := go_bagit.CreateBag(params.Path, "sha512", 1)
	if err != nil {
		return nil, err
	}

	return &CreateBagResult{}, nil
}
