package fformat_test

import (
	"context"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/fformat"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

func TestIdentifyFormats(t *testing.T) {
	t.Parallel()

	testDir := fs.NewDir(t, "",
		fs.WithFile("something.txt", "text"),
		fs.WithFile("else.json", "{}"),
	)

	testSIP := sip.SIP{
		Type:        enums.SIPTypeBornDigitalAIP,
		ContentPath: testDir.Path(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fileformats, err := fformat.IdentifyFormats(ctx, fformat.NewSiegfriedEmbed(), testSIP)
	assert.NilError(t, err)

	assert.Equal(t, len(fileformats), 2)
	assert.Equal(t, fileformats[filepath.Join(testDir.Path(), "something.txt")].ID, "x-fmt/111")
	assert.Equal(t, fileformats[filepath.Join(testDir.Path(), "else.json")].ID, "fmt/817")
}
