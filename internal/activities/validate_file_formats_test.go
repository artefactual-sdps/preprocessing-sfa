package activities_test

import (
	"fmt"
	"path/filepath"
	"testing"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

const pngContent = "\x89PNG\r\n\x1a\n\x00\x00\x00\x0DIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00\x90\x77\x53\xDE\x00\x00\x00\x00IEND\xAE\x42\x60\x82"

func TestValidateFileFormats(t *testing.T) {
	t.Parallel()

	invalidFormatsDir := fs.NewDir(t, "",
		fs.WithDir("test_transfer",
			fs.WithDir("content",
				fs.WithDir("content",
					fs.WithFile("file2.png", pngContent),
					fs.WithDir("dir",
						fs.WithFile("file1.png", pngContent),
					),
				),
			),
		),
	).Path()
	invalidFormatsTransferPath := filepath.Join(invalidFormatsDir, "test_transfer")
	invalidFormatsContentPath := filepath.Join(invalidFormatsTransferPath, "content", "content")

	validFormatsDir := fs.NewDir(t, "",
		fs.WithDir("data",
			fs.WithDir("dir",
				fs.WithDir("content",
					fs.WithDir("content",
						fs.WithFile("file1.txt", "content"),
						fs.WithDir("dir",
							fs.WithFile("file2.txt", "content"),
						),
					),
				),
			),
		),
	).Path()
	validFormatsTransferPath := filepath.Join(validFormatsDir, "data", "dir")
	validFormatsContentPath := filepath.Join(validFormatsTransferPath, "content", "content")

	tests := []struct {
		name    string
		params  activities.ValidateFileFormatsParams
		want    activities.ValidateFileFormatsResult
		wantErr string
	}{
		{
			name: "Successes with valid formats",
			params: activities.ValidateFileFormatsParams{
				SIP: sip.SIP{
					Type:        enums.SIPTypeDigitizedAIP,
					Path:        validFormatsTransferPath,
					ContentPath: validFormatsContentPath,
				},
			},
		},
		{
			name: "Fails with invalid formats",
			params: activities.ValidateFileFormatsParams{
				SIP: sip.SIP{
					Type:        enums.SIPTypeDigitizedAIP,
					Path:        invalidFormatsTransferPath,
					ContentPath: invalidFormatsContentPath,
				},
			},
			want: activities.ValidateFileFormatsResult{
				Failures: []string{
					fmt.Sprintf(
						`file format %q not allowed: "%s/dir/file1.png"`,
						"fmt/11",
						invalidFormatsContentPath,
					),
					fmt.Sprintf(
						`file format %q not allowed: "%s/file2.png"`,
						"fmt/11",
						invalidFormatsContentPath,
					),
				},
			},
		},
		{
			name: "Fails with an invalid content path",
			params: activities.ValidateFileFormatsParams{
				SIP: sip.SIP{
					Type:        enums.SIPTypeDigitizedAIP,
					ContentPath: "/path/to/missing/dir",
				},
			},
			wantErr: "activity error (type: validate-file-formats, scheduledEventID: 0, startedEventID: 0, identity: ): ValidateFileFormats: lstat /path/to/missing/dir: no such file or directory",
		},
		{
			name: "Fails with empty source",
			params: activities.ValidateFileFormatsParams{
				SIP: sip.SIP{
					Type: enums.SIPTypeDigitizedAIP,
					ContentPath: fs.NewDir(t, "",
						fs.WithFile("file.txt", ""),
					).Path(),
				},
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
