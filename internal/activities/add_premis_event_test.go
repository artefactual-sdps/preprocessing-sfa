package activities_test

import (
	"testing"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/premis"
)

func TestAddPREMISEvent(t *testing.T) {
	t.Parallel()

	// Normally populated files (for execution expected to work).
	ContentFilesNormal := fs.NewDir(t, "",
		fs.WithDir("metadata"),
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
		fs.WithDir("metadata"),
		fs.WithDir("content",
			fs.WithDir("content",
				fs.WithDir("d_0000001"),
			),
		),
	)

	PREMISFilePathNoFiles := ContentNoFiles.Join("metadata", "premis.xml")

	// Non-existent paths (for execution expected to fail).
	ContentNonExistent := fs.NewDir(t, "",
		fs.WithDir("metadata"),
		fs.WithDir("content",
			fs.WithDir("content",
				fs.WithDir("d_0000001"),
			),
		),
	)

	PREMISFilePathNonExistent := ContentNonExistent.Join("metadata", "premis.xml")

	ContentNonExistent.Remove()

	// No failures.
	var noFailures []string

	// Failure.
	var failures []string
	failures = append(failures, "some failure")

	tests := []struct {
		name    string
		params  activities.AddPREMISEventParams
		result  activities.AddPREMISEventResult
		wantErr string
	}{
		{
			name: "Add PREMIS event for normal content with no failures",
			params: activities.AddPREMISEventParams{
				PREMISFilePath: PREMISFilePathNormal,
				Agent:          premis.AgentDefault(),
				Type:           "someActivity",
				Failures:       noFailures,
			},
			result: activities.AddPREMISEventResult{},
		},
		{
			name: "Add PREMIS event for normal content with failures",
			params: activities.AddPREMISEventParams{
				PREMISFilePath: PREMISFilePathNormal,
				Agent:          premis.AgentDefault(),
				Type:           "someActivity",
				Failures:       failures,
			},
			result: activities.AddPREMISEventResult{},
		},
		{
			name: "Add PREMIS event for no content",
			params: activities.AddPREMISEventParams{
				PREMISFilePath: PREMISFilePathNoFiles,
				Agent:          premis.AgentDefault(),
				Type:           "someActivity",
				Failures:       noFailures,
			},
			result: activities.AddPREMISEventResult{},
		},
		{
			name: "Add PREMIS event for bad path",
			params: activities.AddPREMISEventParams{
				PREMISFilePath: PREMISFilePathNonExistent,
				Agent:          premis.AgentDefault(),
				Type:           "someActivity",
				Failures:       noFailures,
			},
			result:  activities.AddPREMISEventResult{},
			wantErr: "no such file or directory",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := &temporalsdk_testsuite.WorkflowTestSuite{}
			env := ts.NewTestActivityEnvironment()
			env.RegisterActivityWithOptions(
				activities.NewAddPREMISEvent().Execute,
				temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISEventName},
			)

			var res activities.AddPREMISEventResult
			future, err := env.ExecuteActivity(activities.AddPREMISEventName, tt.params)

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

			_, err = premis.ParseFile(tt.params.PREMISFilePath)
			assert.NilError(t, err)
		})
	}
}
