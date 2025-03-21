package fformat

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/richardlehane/siegfried"
	"github.com/richardlehane/siegfried/pkg/config"
	"github.com/richardlehane/siegfried/pkg/static"
)

// SiegfriedEmbed is an implementation of Siegfried based on the library dist.
// It should be the fastest implementation because it loads just once.
type siegfriedEmbed struct {
	embed     *siegfried.Siegfried
	version   string
	signature string
}

var _ Identifier = (*siegfriedEmbed)(nil)

func NewSiegfriedEmbed() *siegfriedEmbed {
	v := config.Version()
	return &siegfriedEmbed{
		embed:     static.New(),
		version:   fmt.Sprintf("%d.%d.%d", v[0], v[1], v[2]),
		signature: config.SignatureBase(),
	}
}

// Identify runs the Siegfried PRONOM file identifier on the file at path and
// returns a FileFormat pointer or an error.
func (sf *siegfriedEmbed) Identify(path string) (*FileFormat, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ids, err := sf.embed.Identify(f, f.Name(), "")
	if err != nil {
		return nil, err
	}
	if len(ids) > 1 {
		return nil, fmt.Errorf("multiple file formats matched: %q", path)
	}

	// Loop through Siegfried identifier result key-value pairs
	var res FileFormat
	for _, kv := range sf.embed.Label(ids[0]) {
		switch kv[0] {
		case "namespace":
			res.Namespace = kv[1]
		case "id":
			res.ID = kv[1]
		case "format":
			res.CommonName = kv[1]
		case "version":
			res.Version = kv[1]
		case "mime":
			res.MIMEType = kv[1]
		case "basis":
			res.Basis = kv[1]
		case "warning":
			res.Warning = kv[1]
		}
	}

	return &res, nil
}

func (s siegfriedEmbed) Version() string {
	return s.version
}
