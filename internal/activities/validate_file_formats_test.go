package activities_test

import (
	"fmt"
	"testing"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
)

const pngContent = "\x89PNG\r\n\x1a\n\x00\x00\x00\x0DIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00\x90\x77\x53\xDE\x00\x00\x00\x00IEND\xAE\x42\x60\x82"

func TestValidateFileFormats(t *testing.T) {
	t.Parallel()

	invalidFormatsPath := fs.NewDir(t, "",
		fs.WithDir("dir",
			fs.WithFile("file1.png", pngContent),
		),
		fs.WithFile("file2.png", pngContent),
	).Path()

	tests := []struct {
		name    string
		params  activities.ValidateFileFormatsParams
		want    activities.ValidateFileFormatsResult
		wantErr string
	}{
		{
			name: "Successes with valid formats",
			params: activities.ValidateFileFormatsParams{
				ContentPath: fs.NewDir(t, "",
					fs.WithDir("dir",
						fs.WithFile("file1.txt", "content"),
					),
					fs.WithFile("file2.txt", "content"),
				).Path(),
			},
		},
		{
			name:   "Fails with invalid formats",
			params: activities.ValidateFileFormatsParams{ContentPath: invalidFormatsPath},
			want: activities.ValidateFileFormatsResult{
				Failures: []string{
					fmt.Sprintf(
						`file format %q not allowed: "%s/dir/file1.png"`,
						"fmt/11",
						invalidFormatsPath,
					),
					fmt.Sprintf(
						`file format %q not allowed: "%s/file2.png"`,
						"fmt/11",
						invalidFormatsPath,
					),
				},
			},
		},
		{
			name:    "Fails with an invalid content path",
			params:  activities.ValidateFileFormatsParams{ContentPath: "/path/to/missing/dir"},
			wantErr: "activity error (type: validate-file-formats, scheduledEventID: 0, startedEventID: 0, identity: ): ValidateFileFormats: lstat /path/to/missing/dir: no such file or directory",
		},
		{
			name: "Fails with empty source",
			params: activities.ValidateFileFormatsParams{
				ContentPath: fs.NewDir(t, "",
					fs.WithFile("file.txt", ""),
				).Path(),
			},
			wantErr: "activity error (type: validate-file-formats, scheduledEventID: 0, startedEventID: 0, identity: ): ValidateFileFormats: identify format: empty source",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := &temporalsdk_testsuite.WorkflowTestSuite{}
			env := ts.NewTestActivityEnvironment()
			env.RegisterActivityWithOptions(
				activities.NewValidateFileFormats().Execute,
				temporalsdk_activity.RegisterOptions{Name: activities.ValidateFileFormatsName},
			)

			enc, err := env.ExecuteActivity(activities.ValidateFileFormatsName, tt.params)
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("error is nil, expecting: %q", tt.wantErr)
				} else {
					assert.Error(t, err, tt.wantErr)
				}

				return
			}
			assert.NilError(t, err)

			var result activities.ValidateFileFormatsResult
			_ = enc.Get(&result)
			assert.DeepEqual(t, result, tt.want)
		})
	}
}
