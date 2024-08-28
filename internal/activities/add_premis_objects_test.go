package activities_test

import (
	"testing"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/premis"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

func TestAddPREMISObjects(t *testing.T) {
	t.Parallel()

	// Normally populated files (for execution expected to work).
	ContentFilesNormal := fs.NewDir(t, "",
		fs.WithDir("content",
			fs.WithDir("content",
				fs.WithDir("d_0000001",
					fs.WithFile("00000001.jp2", ""),
					fs.WithFile("00000001_PREMIS.xml", ""),
				),
			),
		),
	)

	PREMISFilePathNormal := ContentFilesNormal.Join("metadata", "premis.xml")

	// No files (for execution expected to work).
	ContentNoFiles := fs.NewDir(t, "",
		fs.WithDir("content",
			fs.WithDir("content",
				fs.WithDir("d_0000001"),
			),
		),
	)

	PREMISFilePathNoFiles := ContentNoFiles.Join("metadata", "premis.xml")

	// Non-existent paths (for execution expected to fail).
	ContentNonExistent := fs.NewDir(t, "",
		fs.WithDir("content",
			fs.WithDir("content",
				fs.WithDir("d_0000001"),
			),
		),
	)

	PREMISFilePathNonExistent := ContentNonExistent.Join("metadata", "premis.xml")

	ContentNonExistent.Remove()

	tests := []struct {
		name    string
		params  activities.AddPREMISObjectsParams
		result  activities.AddPREMISObjectsResult
		wantErr string
	}{
		{
			name: "Add PREMIS objects for normal content",
			params: activities.AddPREMISObjectsParams{
				SIP: sip.SIP{
					Type:        enums.SIPTypeDigitizedAIP,
					ContentPath: ContentFilesNormal.Path(),
				},
				PREMISFilePath: PREMISFilePathNormal,
			},
			result: activities.AddPREMISObjectsResult{},
		},
		{
			name: "Add PREMIS objects for no content",
			params: activities.AddPREMISObjectsParams{
				SIP: sip.SIP{
					Type:        enums.SIPTypeDigitizedAIP,
					ContentPath: ContentNoFiles.Path(),
				},
				PREMISFilePath: PREMISFilePathNoFiles,
			},
			result: activities.AddPREMISObjectsResult{},
		},
		{
			name: "Add PREMIS objects for bad path",
			params: activities.AddPREMISObjectsParams{
				SIP: sip.SIP{
					Type:        enums.SIPTypeDigitizedAIP,
					ContentPath: ContentNonExistent.Path(),
				},
				PREMISFilePath: PREMISFilePathNonExistent,
			},
			result:  activities.AddPREMISObjectsResult{},
			wantErr: "no such file or directory",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := &temporalsdk_testsuite.WorkflowTestSuite{}
			env := ts.NewTestActivityEnvironment()
			env.RegisterActivityWithOptions(
				activities.NewAddPREMISObjects().Execute,
				temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISObjectsName},
			)

			var res activities.AddPREMISObjectsResult
			future, err := env.ExecuteActivity(activities.AddPREMISObjectsName, tt.params)

			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("error is nil, expecting: %q", tt.wantErr)
				} else {
					assert.ErrorContains(t, err, tt.wantErr)
				}

				return
			}

			assert.NilError(t, err)

			future.Get(&res)
			assert.NilError(t, err)
			assert.DeepEqual(t, res, tt.result)

			// If the content directory has files, make sure that PREMIS file can be parsed.
			contentFiles, err := premis.FilesWithinDirectory(tt.params.SIP.ContentPath)
			assert.NilError(t, err)

			if len(contentFiles) > 0 {
				_, err = premis.ParseFile(tt.params.PREMISFilePath)
				assert.NilError(t, err)
			}
		})
	}
}
