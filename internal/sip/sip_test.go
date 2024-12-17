package sip_test

import (
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

func TestNew(t *testing.T) {
	t.Parallel()

	digitizedAIP := fs.NewDir(t, "",
		fs.WithDir("Digitized-AIP",
			fs.WithDir("content",
				fs.WithDir("d_0000001",
					fs.WithFile("Prozess_Digitalisierung_PREMIS.xml", ""),
				),
			),
			fs.WithDir("additional"),
		),
	)
	digiAIPPath := digitizedAIP.Join("Digitized-AIP")

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

	bornDigitalAIP := fs.NewDir(t, "",
		fs.WithDir("Born-Digital-AIP",
			fs.WithDir("additional",
				fs.WithFile("UpdatedAreldaMetadata.xml", ""),
				fs.WithFile("Born-Digital-AIP-premis.xml", ""),
			),
			fs.WithDir("content",
				fs.WithDir("content"),
				fs.WithDir("header",
					fs.WithDir("old",
						fs.WithDir("SIP",
							fs.WithFile("metadata.xml", ""),
						),
					),
					fs.WithDir("xsd"),
				),
			),
		),
	)
	bdAIPPath := bornDigitalAIP.Join("Born-Digital-AIP")

	bornDigitalSIP := fs.NewDir(t, "",
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
			path: digiAIPPath,
			wantSIP: sip.SIP{
				Type:                enums.SIPTypeDigitizedAIP,
				Path:                digiAIPPath,
				ContentPath:         filepath.Join(digiAIPPath, "content", "content"),
				LogicalMDPath:       filepath.Join(digiAIPPath, "additional", "Digitized-AIP-premis.xml"),
				ManifestPath:        filepath.Join(digiAIPPath, "additional", "UpdatedAreldaMetadata.xml"),
				MetadataPath:        filepath.Join(digiAIPPath, "content", "header", "old", "SIP", "metadata.xml"),
				UpdatedAreldaMDPath: filepath.Join(digiAIPPath, "additional", "UpdatedAreldaMetadata.xml"),
				XSDPath:             filepath.Join(digiAIPPath, "content", "header", "xsd", "arelda.xsd"),
				TopLevelPaths: []string{
					filepath.Join(digiAIPPath, "content"),
					filepath.Join(digiAIPPath, "additional"),
				},
			},
		},
		{
			name: "Creates a new born digital AIP",
			path: bdAIPPath,
			wantSIP: sip.SIP{
				Type:                enums.SIPTypeBornDigitalAIP,
				Path:                bdAIPPath,
				ContentPath:         filepath.Join(bdAIPPath, "content", "content"),
				LogicalMDPath:       filepath.Join(bdAIPPath, "additional", "Born-Digital-AIP-premis.xml"),
				ManifestPath:        filepath.Join(bdAIPPath, "additional", "UpdatedAreldaMetadata.xml"),
				MetadataPath:        filepath.Join(bdAIPPath, "content", "header", "old", "SIP", "metadata.xml"),
				UpdatedAreldaMDPath: filepath.Join(bdAIPPath, "additional", "UpdatedAreldaMetadata.xml"),
				XSDPath:             filepath.Join(bdAIPPath, "content", "header", "xsd", "arelda.xsd"),
				TopLevelPaths: []string{
					filepath.Join(bdAIPPath, "content"),
					filepath.Join(bdAIPPath, "additional"),
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
			path: bornDigitalSIP.Path(),
			wantSIP: sip.SIP{
				Type:         enums.SIPTypeBornDigitalSIP,
				Path:         bornDigitalSIP.Path(),
				ContentPath:  bornDigitalSIP.Join("content"),
				ManifestPath: bornDigitalSIP.Join("header", "metadata.xml"),
				MetadataPath: bornDigitalSIP.Join("header", "metadata.xml"),
				XSDPath:      bornDigitalSIP.Join("header", "xsd", "arelda.xsd"),
				TopLevelPaths: []string{
					bornDigitalSIP.Join("content"),
					bornDigitalSIP.Join("header"),
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

func TestName(t *testing.T) {
	t.Parallel()

	s := sip.SIP{
		Type: enums.SIPTypeDigitizedSIP,
		Path: "/path/to/SIP_20201201_Vecteur",
	}
	assert.Equal(t, s.Name(), "SIP_20201201_Vecteur")
}

func TestIsAIP(t *testing.T) {
	t.Parallel()

	s := sip.SIP{
		Type: enums.SIPTypeBornDigitalAIP,
		Path: "/path/to/AIP_20201201",
	}
	assert.Assert(t, s.IsAIP())

	s = sip.SIP{
		Type: enums.SIPTypeBornDigitalSIP,
		Path: "/path/to/SIP_20201201_Vecteur",
	}
	assert.Assert(t, !s.IsAIP())
}

func TestIsSIP(t *testing.T) {
	t.Parallel()

	s := sip.SIP{
		Type: enums.SIPTypeBornDigitalSIP,
		Path: "/path/to/SIP_20201201_Vecteur",
	}
	assert.Assert(t, s.IsSIP())

	s = sip.SIP{
		Type: enums.SIPTypeBornDigitalAIP,
		Path: "/path/to/AIP_20201201",
	}
	assert.Assert(t, !s.IsSIP())
}
