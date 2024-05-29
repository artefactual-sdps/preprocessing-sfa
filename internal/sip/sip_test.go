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

	aipPath := fs.NewDir(t, "", fs.WithDir("content"), fs.WithDir("additional")).Path()
	sipPath := fs.NewDir(t, "", fs.WithDir("content"), fs.WithDir("header")).Path()

	tests := []struct {
		name    string
		path    string
		wantSIP sip.SIP
		wantErr string
	}{
		{
			name: "Creates a new SIP (type: VecteurAIP)",
			path: aipPath,
			wantSIP: sip.SIP{
				Type:         enums.SIPTypeVecteurAIP,
				Path:         aipPath,
				ContentPath:  filepath.Join(aipPath, "content", "content"),
				MetadataPath: filepath.Join(aipPath, "additional", "UpdatedAreldaMetadata.xml"),
				XSDPath:      filepath.Join(aipPath, "content", "header", "xsd", "arelda.xsd"),
				RemovePaths:  []string{filepath.Join(aipPath, "content"), filepath.Join(aipPath, "additional")},
			},
		},
		{
			name: "Creates a new SIP (type: VecteurSIP)",
			path: sipPath,
			wantSIP: sip.SIP{
				Type:         enums.SIPTypeVecteurSIP,
				Path:         sipPath,
				ContentPath:  filepath.Join(sipPath, "content"),
				MetadataPath: filepath.Join(sipPath, "header", "metadata.xml"),
				XSDPath:      filepath.Join(sipPath, "header", "xsd", "arelda.xsd"),
				RemovePaths:  []string{filepath.Join(sipPath, "content"), filepath.Join(sipPath, "header")},
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
