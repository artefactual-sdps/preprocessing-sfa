package amss

import (
	"context"
	"io"
)

type Service interface {
	GetAIPPath(ctx context.Context, aipUUID string) (string, error)
	DownloadAIPFile(ctx context.Context, aipUUID, path string, writer io.Writer) error
}
