package sip_test

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

func TestNew(t *testing.T) {
	t.Parallel()

	digitizedAIP := fs.NewDir(t, "",
		fs.WithDir("content"),
		fs.WithDir("additional"),
	)

	digitizedSIP := fs.NewDir(t, "SIP_20201201_Vecteur",
		fs.WithDir("content",
			fs.WithDir("d_0000001",
				fs.WithFile("Prozess_Digitalisierung_PREMIS.xml", ""),
			),
		),
		fs.WithDir("header"),
	)

	digitizedSIPCase := fs.NewDir(t, "SIP_20201201_vecteur",
		fs.WithDir("content",
			fs.WithDir("d_0000001",
				fs.WithFile("Prozess_Digitalisierung_PREMIS.xml", ""),
			),
		),
		fs.WithDir("header"),
	)

	bornDigital := fs.NewDir(t, "",
		fs.WithDir("content"),
		fs.WithDir("header"),
	)

	tests := []struct {
		name    string
		path    string
		wantSIP sip.SIP
		wantErr string
	}{
		{
			name: "Creates a new digitized AIP",
			path: digitizedAIP.Path(),
			wantSIP: sip.SIP{
				Type:                enums.SIPTypeDigitizedAIP,
				Path:                digitizedAIP.Path(),
				ContentPath:         digitizedAIP.Join("content", "content"),
				ManifestPath:        digitizedAIP.Join("additional", "UpdatedAreldaMetadata.xml"),
				MetadataPath:        digitizedAIP.Join("content", "header", "old", "SIP", "metadata.xml"),
				UpdatedAreldaMDPath: digitizedAIP.Join("additional", "UpdatedAreldaMetadata.xml"),
				XSDPath:             digitizedAIP.Join("content", "header", "xsd", "arelda.xsd"),
				TopLevelPaths: []string{
					digitizedAIP.Join("content"),
					digitizedAIP.Join("additional"),
				},
			},
		},
		{
			name: "Creates a new digitized SIP",
			path: digitizedSIP.Path(),
			wantSIP: sip.SIP{
				Type:         enums.SIPTypeDigitizedSIP,
				Path:         digitizedSIP.Path(),
				ContentPath:  digitizedSIP.Join("content"),
				ManifestPath: digitizedSIP.Join("header", "metadata.xml"),
				MetadataPath: digitizedSIP.Join("header", "metadata.xml"),
				XSDPath:      digitizedSIP.Join("header", "xsd", "arelda.xsd"),
				TopLevelPaths: []string{
					digitizedSIP.Join("content"),
					digitizedSIP.Join("header"),
				},
			},
		},
		{
			name: "Creates a new digitized SIP (case-insensitive)",
			path: digitizedSIPCase.Path(),
			wantSIP: sip.SIP{
				Type:         enums.SIPTypeDigitizedSIP,
				Path:         digitizedSIPCase.Path(),
				ContentPath:  digitizedSIPCase.Join("content"),
				ManifestPath: digitizedSIPCase.Join("header", "metadata.xml"),
				MetadataPath: digitizedSIPCase.Join("header", "metadata.xml"),
				XSDPath:      digitizedSIPCase.Join("header", "xsd", "arelda.xsd"),
				TopLevelPaths: []string{
					digitizedSIPCase.Join("content"),
					digitizedSIPCase.Join("header"),
				},
			},
		},
		{
			name: "Creates a new born digital SIP",
			path: bornDigital.Path(),
			wantSIP: sip.SIP{
				Type:         enums.SIPTypeBornDigital,
				Path:         bornDigital.Path(),
				ContentPath:  bornDigital.Join("content"),
				ManifestPath: bornDigital.Join("header", "metadata.xml"),
				MetadataPath: bornDigital.Join("header", "metadata.xml"),
				XSDPath:      bornDigital.Join("header", "xsd", "arelda.xsd"),
				TopLevelPaths: []string{
					bornDigital.Join("content"),
					bornDigital.Join("header"),
				},
			},
		},
		{
			name:    "Fails with a non existing path",
			wantErr: "SIP: New: stat : no such file or directory",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, err := sip.New(tt.path)

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
