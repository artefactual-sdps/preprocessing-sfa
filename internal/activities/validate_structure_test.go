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

func TestValidateStructure(t *testing.T) {
	t.Parallel()

	digitizedAIP, err := sip.New(fs.NewDir(t, "",
		fs.WithDir("additional",
			fs.WithFile("UpdatedAreldaMetadata.xml", ""),
		),
		fs.WithDir("content",
			fs.WithDir("content",
				fs.WithDir("d_0000001"),
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
	).Path())
	assert.NilError(t, err)

	digitizedSIP, err := sip.New(fs.NewDir(t, "",
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
	).Path())
	assert.NilError(t, err)

	unexpectedPiecesSIP, err := sip.New(fs.NewDir(t, "",
		fs.WithDir("content",
			fs.WithDir("d_0000001"),
			fs.WithFile("unexpected.txt", ""),
		),
		fs.WithDir("header",
			fs.WithFile("metadata.xml", ""),
			fs.WithDir("xsd",
				fs.WithFile("arelda.xsd", ""),
			),
		),
		fs.WithDir("unexpected"),
	).Path())
	assert.NilError(t, err)

	missingPiecesSIP, err := sip.New(fs.NewDir(t, "").Path())
	assert.NilError(t, err)

	missingPiecesAIP, err := sip.New(fs.NewDir(t, "",
		fs.WithDir("additional"),
	).Path())
	assert.NilError(t, err)

	digitizedSIPExtraDossiers, err := sip.New(fs.NewDir(t, "",
		fs.WithDir("content",
			fs.WithDir("d_0000001",
				fs.WithFile("00000001.jp2", ""),
				fs.WithFile("00000001_PREMIS.xml", ""),
				fs.WithFile("Prozess_Digitalisierung_PREMIS.xml", ""),
			),
			fs.WithDir("d_0000002",
				fs.WithFile("00000002.jp2", ""),
				fs.WithFile("00000002_PREMIS.xml", ""),
				fs.WithFile("Prozess_Digitalisierung_PREMIS.xml", ""),
			),
			fs.WithDir("d_0000003",
				fs.WithFile("00000003.jp2", ""),
				fs.WithFile("00000003_PREMIS.xml", ""),
				fs.WithFile("Prozess_Digitalisierung_PREMIS.xml", ""),
			),
		),
		fs.WithDir("header",
			fs.WithFile("metadata.xml", ""),
			fs.WithDir("xsd",
				fs.WithFile("arelda.xsd", ""),
			),
		),
	).Path())
	assert.NilError(t, err)

	tests := []struct {
		name    string
		params  activities.ValidateStructureParams
		want    activities.ValidateStructureResult
		wantErr string
	}{
		{
			name:   "Validates a digitized AIP",
			params: activities.ValidateStructureParams{SIP: digitizedAIP},
		},
		{
			name:   "Validates a digitized SIP",
			params: activities.ValidateStructureParams{SIP: digitizedSIP},
		},
		{
			name:   "Returns failures when the SIP has unexpected components",
			params: activities.ValidateStructureParams{SIP: unexpectedPiecesSIP},
			want: activities.ValidateStructureResult{
				Failures: []string{
					fmt.Sprintf("Unexpected directory: %q", unexpectedPiecesSIP.Path+"/unexpected"),
					fmt.Sprintf("Unexpected file: %q", unexpectedPiecesSIP.Path+"/content/unexpected.txt"),
				},
			},
		},
		{
			name:   "Returns failures when the SIP is missing components",
			params: activities.ValidateStructureParams{SIP: missingPiecesSIP},
			want: activities.ValidateStructureResult{
				Failures: []string{
					"Content folder is missing",
					"XSD folder is missing",
					"metadata.xml is missing",
				},
			},
		},
		{
			name:   "Returns failures when a digitized AIP is missing components",
			params: activities.ValidateStructureParams{SIP: missingPiecesAIP},
			want: activities.ValidateStructureResult{
				Failures: []string{
					"Content folder is missing",
					"XSD folder is missing",
					"metadata.xml is missing",
					"UpdatedAreldaMetadata.xml is missing",
				},
			},
		},
		{
			name:   "Returns a failure when a digitized SIP has more than one dossier",
			params: activities.ValidateStructureParams{SIP: digitizedSIPExtraDossiers},
			want: activities.ValidateStructureResult{
				Failures: []string{"More than one dossier in the content directory"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := &temporalsdk_testsuite.WorkflowTestSuite{}
			env := ts.NewTestActivityEnvironment()
			env.RegisterActivityWithOptions(
				activities.NewValidateStructure().Execute,
				temporalsdk_activity.RegisterOptions{Name: activities.ValidateStructureName},
			)

			enc, err := env.ExecuteActivity(activities.ValidateStructureName, tt.params)
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("error is nil, expecting: %q", tt.wantErr)
				} else {
					assert.ErrorContains(t, err, tt.wantErr)
				}

				return
			}

			var result activities.ValidateStructureResult
			_ = enc.Get(&result)

			assert.NilError(t, err)
			assert.DeepEqual(t, result, tt.want)
		})
	}
}
