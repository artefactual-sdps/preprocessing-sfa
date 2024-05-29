package sip_test

import (
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

func TestNewSIP(t *testing.T) {
	t.Parallel()

	vecteurAIP := fs.NewDir(t, "", fs.WithDir("content"), fs.WithDir("additional"))
	vecteurSIP := fs.NewDir(t, "", fs.WithDir("content"), fs.WithDir("header"))

	tests := []struct {
		name    string
		path    string
		wantSIP sip.SIP
		wantErr string
	}{
		{
			name: "Creates a new SIP (type: VecteurAIP)",
			path: vecteurAIP.Path(),
			wantSIP: sip.SIP{
				Type:         enums.SIPTypeVecteurAIP,
				Path:         vecteurAIP.Path(),
				ContentPath:  filepath.Join(vecteurAIP.Path(), "content", "content"),
				MetadataPath: filepath.Join(vecteurAIP.Path(), "additional", "UpdatedAreldaMetadata.xml"),
				XSDPath:      filepath.Join(vecteurAIP.Path(), "content", "header", "xsd", "arelda.xsd"),
			},
		},
		{
			name: "Creates a new SIP (type: VecteurSIP)",
			path: vecteurSIP.Path(),
			wantSIP: sip.SIP{
				Type:         enums.SIPTypeVecteurSIP,
				Path:         vecteurSIP.Path(),
				ContentPath:  filepath.Join(vecteurSIP.Path(), "content"),
				MetadataPath: filepath.Join(vecteurSIP.Path(), "header", "metadata.xml"),
				XSDPath:      filepath.Join(vecteurSIP.Path(), "header", "xsd", "arelda.xsd"),
			},
		},
		{
			name:    "Fails with a non existing path",
			wantErr: "NewSIP: stat : no such file or directory",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, err := sip.NewSIP(tt.path)

			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("error is nil, expecting: %q", tt.wantErr)
				} else {
					assert.ErrorContains(t, err, tt.wantErr)
				}

				return
			}

			assert.NilError(t, err)
			assert.DeepEqual(t, s, tt.wantSIP)
		})
	}
}
