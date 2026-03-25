package activities_test

import (
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

	digitizedAIP, err := sip.New(
		fs.NewDir(t, "extract-",
			fs.WithDir("7537ab2c-4e6b-4820-95bf-bd2c577351c3",
				fs.WithDir("additional",
					fs.WithFile("7537ab2c-4e6b-4820-95bf-bd2c577351c3-premis.xml", ""),
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
			),
		).Join("7537ab2c-4e6b-4820-95bf-bd2c577351c3"),
	)
	assert.NilError(t, err)

	digitizedSIP, err := sip.New(fs.NewDir(t, "extract-",
		fs.WithDir("SIP_202200915_dept",
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
		),
	).Join("SIP_202200915_dept"))
	assert.NilError(t, err)

	emptySIP, err := sip.New(
		fs.NewDir(t, "extract-",
			fs.WithDir("AIP-1234"),
		).Join("AIP-1234"),
	)
	assert.NilError(t, err)

	unexpectedPiecesSIP, err := sip.New(
		fs.NewDir(t, "extract-",
			fs.WithDir("SIP_202200915_dept",
				fs.WithDir("content",
					fs.WithDir("d_0000001",
						fs.WithFile("content.txt", ""),
					),
					fs.WithFile("unexpected.txt", ""),
				),
				fs.WithDir("header",
					fs.WithFile("metadata.xml", ""),
					fs.WithDir("xsd",
						fs.WithFile("arelda.xsd", ""),
					),
				),
				fs.WithDir("unexpected",
					fs.WithFile("data.txt", ""),
				),
			),
		).Join("SIP_202200915_dept"),
	)
	assert.NilError(t, err)

	missingPiecesSIP, err := sip.New(
		fs.NewDir(t, "extract-",
			fs.WithDir("SIP_202200915_dept",
				fs.WithDir("header",
					fs.WithFile("metadata.xml", ""),
				),
			),
		).Join("SIP_202200915_dept"),
	)
	assert.NilError(t, err)

	missingPiecesAIP, err := sip.New(
		fs.NewDir(t, "extract-",
			fs.WithDir("AIP-1234",
				fs.WithDir("additional",
					fs.WithFile("content.txt", ""),
				),
				fs.WithFile("Prozess_Digitalisierung_PREMIS.xml", ""),
			),
		).Join("AIP-1234"),
	)
	assert.NilError(t, err)

	digitizedSIPExtraDossier, err := sip.New(
		fs.NewDir(t, "extract-",
			fs.WithDir("SIP_20251015_Vecteur_987654",
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
				),
				fs.WithDir("header",
					fs.WithFile("metadata.xml", ""),
					fs.WithDir("xsd",
						fs.WithFile("arelda.xsd", ""),
					),
				),
			),
		).Join("SIP_20251015_Vecteur_987654"),
	)
	assert.NilError(t, err)

	badNameSIP, err := sip.New(
		fs.NewDir(t, "extract-",
			fs.WithDir("SIP_20251015_Vecteur_987654",
				fs.WithDir("content",
					fs.WithDir("d_0000001",
						fs.WithFile("00000001$.jp2", ""),
						fs.WithFile("00000001$_PREMIS.xml", ""),
					),
				),
				fs.WithDir("header",
					fs.WithFile("content!.txt", ""),
					fs.WithFile("metadata.xml", ""),
					fs.WithDir("xsd",
						fs.WithFile("arelda.xsd", ""),
					),
				),
			),
		).Join("SIP_20251015_Vecteur_987654"),
	)
	assert.NilError(t, err)

	digitizedAIPEmptyDir, err := sip.New(
		fs.NewDir(t, "extract-",
			fs.WithDir("AIP-1234",
				fs.WithDir("additional",
					fs.WithFile("UpdatedAreldaMetadata.xml", ""),
					fs.WithFile("AIP-1234-premis.xml", ""),
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
			),
		).Join("AIP-1234"),
	)
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
			name:   "Returns failures when the SIP is empty",
			params: activities.ValidateStructureParams{SIP: emptySIP},
			want: activities.ValidateStructureResult{
				Failures: []string{
					"The SIP is empty",
				},
			},
		},
		{
			name:   "Returns failures when the SIP has unexpected components",
			params: activities.ValidateStructureParams{SIP: unexpectedPiecesSIP},
			want: activities.ValidateStructureResult{
				Failures: []string{
					`Unexpected directory: "unexpected"`,
					`Unexpected file: "content/unexpected.txt"`,
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
					"AIP-1234-premis.xml is missing",
				},
			},
		},
		{
			name:   "Returns a failure when a digitized SIP has more than one dossier",
			params: activities.ValidateStructureParams{SIP: digitizedSIPExtraDossier},
			want: activities.ValidateStructureResult{
				Failures: []string{"More than one dossier in the content directory"},
			},
		},
		{
			name:   "Returns a failure when the name of files and/or directories in a SIP have invalid characters",
			params: activities.ValidateStructureParams{SIP: badNameSIP},
			want: activities.ValidateStructureResult{
				Failures: []string{
					"Name \"content/d_0000001/00000001$.jp2\" contains invalid character(s)",
					"Name \"content/d_0000001/00000001$_PREMIS.xml\" contains invalid character(s)",
					"Name \"header/content!.txt\" contains invalid character(s)",
				},
			},
		},
		{
			name:   "Returns a failure when a digitized AIP has an empty directory",
			params: activities.ValidateStructureParams{SIP: digitizedAIPEmptyDir},
			want: activities.ValidateStructureResult{
				Failures: []string{
					"An empty directory has been found - content/content/d_0000001",
					"Please remove the empty directories and update the metadata manifest accordingly",
				},
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
			assert.NilError(t, err)

			var result activities.ValidateStructureResult
			_ = enc.Get(&result)
			assert.DeepEqual(t, result, tt.want)
		})
	}
}
