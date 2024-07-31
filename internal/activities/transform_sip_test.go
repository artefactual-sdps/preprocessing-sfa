package activities_test

import (
	"fmt"
	"testing"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

func TestTransformSIP(t *testing.T) {
	t.Parallel()

	vecteurAIPPath := fs.NewDir(t, "",
		fs.WithDir("additional",
			fs.WithFile("UpdatedAreldaMetadata.xml", ""),
		),
		fs.WithDir("content",
			fs.WithDir("content",
				fs.WithDir("d_0000001",
					fs.WithFile("00000001.jp2", ""),
					fs.WithFile("00000001_PREMIS.xml", ""),
					fs.WithFile("Prozess_Digitalisierung_PREMIS.xml", ""),
				),
			),
			fs.WithDir("header",
				fs.WithDir("old",
					fs.WithDir("SIP",
						fs.WithFile("metadata.xml", ""),
					),
				),
				fs.WithDir("xsd",
					fs.WithFile("arelda.xsd", ""),
				),
			),
		),
	).Path()
	vecteurSIPPath := fs.NewDir(t, "",
		fs.WithDir("content",
			fs.WithDir("d_0000001",
				fs.WithFile("00000001.jp2", ""),
				fs.WithFile("00000001_PREMIS.xml", ""),
				fs.WithFile("Prozess_Digitalisierung_PREMIS.xml", ""),
			),
		),
		fs.WithDir("header",
			fs.WithFile("metadata.xml", ""),
			fs.WithDir("xsd",
				fs.WithFile("arelda.xsd", ""),
			),
		),
	).Path()

	vecteurAIP, err := sip.NewSIP(vecteurAIPPath)
	assert.NilError(t, err)
	vecteurSIP, err := sip.NewSIP(vecteurSIPPath)
	assert.NilError(t, err)

	expectedVecteurAIP := fs.Expected(t,
		fs.WithDir("objects", fs.WithMode(0o700),
			fs.WithDir("d_0000001",
				fs.WithFile("00000001.jp2", ""),
				fs.WithFile("00000001_PREMIS.xml", ""),
			),
		),
		fs.WithDir("metadata", fs.WithMode(0o700),
			fs.WithFile("Prozess_Digitalisierung_PREMIS_d_0000001.xml", ""),
			fs.WithFile("UpdatedAreldaMetadata.xml", ""),
		),
	)
	expectedVecteurSIP := fs.Expected(t,
		fs.WithDir("objects", fs.WithMode(0o700),
			fs.WithDir("d_0000001",
				fs.WithFile("00000001.jp2", ""),
				fs.WithFile("00000001_PREMIS.xml", ""),
			),
		),
		fs.WithDir("metadata", fs.WithMode(0o700),
			fs.WithFile("metadata.xml", ""),
			fs.WithFile("Prozess_Digitalisierung_PREMIS_d_0000001.xml", ""),
		),
	)

	missingMetadataSIP, err := sip.NewSIP(fs.NewDir(t, "").Path())
	assert.NilError(t, err)
	missingContentSIP, err := sip.NewSIP(fs.NewDir(t, "",
		fs.WithDir("header",
			fs.WithFile("metadata.xml", ""),
		),
	).Path())
	assert.NilError(t, err)

	tests := []struct {
		name    string
		params  activities.TransformSIPParams
		wantSIP fs.Manifest
		wantErr string
	}{
		{
			name:    "Transforms a Vecteur AIP",
			params:  activities.TransformSIPParams{SIP: *vecteurAIP},
			wantSIP: expectedVecteurAIP,
		},
		{
			name:    "Transforms a Vecteur SIP",
			params:  activities.TransformSIPParams{SIP: *vecteurSIP},
			wantSIP: expectedVecteurSIP,
		},
		{
			name:   "Fails with a SIP missing the metadata file",
			params: activities.TransformSIPParams{SIP: *missingMetadataSIP},
			wantErr: fmt.Sprintf(
				"rename %s/header/metadata.xml %s/metadata/metadata.xml: no such file or directory",
				missingMetadataSIP.Path,
				missingMetadataSIP.Path,
			),
		},
		{
			name:   "Fails with a SIP missing the content directory",
			params: activities.TransformSIPParams{SIP: *missingContentSIP},
			wantErr: fmt.Sprintf(
				"lstat %s/content: no such file or directory",
				missingContentSIP.Path,
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := &temporalsdk_testsuite.WorkflowTestSuite{}
			env := ts.NewTestActivityEnvironment()
			env.RegisterActivityWithOptions(
				activities.NewTransformSIP().Execute,
				temporalsdk_activity.RegisterOptions{Name: activities.TransformSIPName},
			)

			_, err := env.ExecuteActivity(activities.TransformSIPName, tt.params)

			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("error is nil, expecting: %q", tt.wantErr)
				} else {
					assert.ErrorContains(t, err, tt.wantErr)
				}

				return
			}

			assert.NilError(t, err)
			assert.Assert(t, fs.Equal(tt.params.SIP.Path, tt.wantSIP))
		})
	}
}
