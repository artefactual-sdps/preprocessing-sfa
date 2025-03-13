package amss

import (
	"context"
	"io"
)

// Client is a minimal representation of the Archivematica Storage Client API endpoints with which preprocessing-sfa
// interacts.
type Client interface {
	GetAIPPath(ctx context.Context, aipUUID string) (string, error)
	DownloadAIPFile(ctx context.Context, aipUUID, path string, writer io.Writer) error
}
