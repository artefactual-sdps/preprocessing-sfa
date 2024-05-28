package activities

import (
	"context"
	"os"
	"path/filepath"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
)

const IdentifyTransferName = "identify-transfer"

type IdentifyTransferParams struct {
	Path string
}
type IdentifyTransferResult struct {
	Type enums.TransferType
}

type IdentifyTransfer struct{}

func NewIdentifyTransfer() *IdentifyTransfer {
	return &IdentifyTransfer{}
}

func (a *IdentifyTransfer) Execute(
	ctx context.Context,
	params *IdentifyTransferParams,
) (*IdentifyTransferResult, error) {
	res := &IdentifyTransferResult{Type: enums.TransferTypeVecteurAIP}
	if _, err := os.Stat(filepath.Join(params.Path, "additional")); os.IsNotExist(err) {
		res.Type = enums.TransferTypeVecteurSIP
	}

	return res, nil
}
