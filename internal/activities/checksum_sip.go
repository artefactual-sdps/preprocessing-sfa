package activities

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

const ChecksumSIPName = "checksum-sip"

type (
	ChecksumSIPParams struct {
		Path string
	}
	ChecksumSIPResult struct {
		Hash string
	}
	ChecksumSIP struct{}
)

func NewChecksumSIP() *ChecksumSIP {
	return &ChecksumSIP{}
}

func (a *ChecksumSIP) Execute(ctx context.Context, params *ChecksumSIPParams) (*ChecksumSIPResult, error) {
	f, err := os.Open(params.Path)
	if err != nil {
		return nil, fmt.Errorf("ChecksumSIP: open SIP: %v", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, fmt.Errorf("ChecksumSIP: calculate checksum: %v", err)
	}

	return &ChecksumSIPResult{Hash: hex.EncodeToString(h.Sum(nil))}, nil
}
