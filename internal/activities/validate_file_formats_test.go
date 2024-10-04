package activities_test

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/tonglil/buflogr"
	"go.artefactual.dev/tools/temporal"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_interceptor "go.temporal.io/sdk/interceptor"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_worker "go.temporal.io/sdk/worker"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/fformat"
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

	emptyAllowlistDir := fs.NewDir(t, "", fs.WithFile("allowlist.csv", ""))
	invalidAllowListDir := fs.NewDir(t, "", fs.WithFile("allowlist.csv", `PRONOM_ID
fmt/95,fmt/96
`))

	tests := []struct {
		name    string
		cfg     fformat.Config
		params  activities.ValidateFileFormatsParams
		want    activities.ValidateFileFormatsResult
		wantErr string
		wantLog string
	}{
		{
			name: "Successes with valid formats",
			cfg: fformat.Config{
				AllowlistPath: "../testdata/allowed_file_formats.csv",
			},
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
			cfg: fformat.Config{
				AllowlistPath: "../testdata/allowed_file_formats.csv",
			},
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
			cfg: fformat.Config{
				AllowlistPath: "../testdata/allowed_file_formats.csv",
			},
			params: activities.ValidateFileFormatsParams{
				SIP: sip.SIP{
					Type:        enums.SIPTypeDigitizedAIP,
					ContentPath: "/path/to/missing/dir",
				},
			},
			wantErr: "ValidateFileFormats: lstat /path/to/missing/dir: no such file or directory",
		},
		{
			name: "Fails with empty source",
			cfg: fformat.Config{
				AllowlistPath: "../testdata/allowed_file_formats.csv",
			},
			params: activities.ValidateFileFormatsParams{
				SIP: sip.SIP{
					Type: enums.SIPTypeDigitizedAIP,
					ContentPath: fs.NewDir(t, "",
						fs.WithFile("file.txt", ""),
					).Path(),
				},
			},
			wantErr: "ValidateFileFormats: identify format: empty source",
		},
		{
			name: "Does nothing when no allowlist path configured",
			params: activities.ValidateFileFormatsParams{
				SIP: sip.SIP{
					Type:        enums.SIPTypeDigitizedAIP,
					Path:        validFormatsTransferPath,
					ContentPath: validFormatsContentPath,
				},
			},
			wantLog: "INFO ValidateFileFormats: No file format allowlist path set, skipping file format validation ActivityID 0 ActivityType validate-file-formats\n",
		},
		{
			name: "Errors when allowlist path doesn't exist",
			cfg:  fformat.Config{AllowlistPath: filepath.Join("/dev/null/allowlist.csv")},
			params: activities.ValidateFileFormatsParams{
				SIP: sip.SIP{
					Type:        enums.SIPTypeDigitizedAIP,
					Path:        validFormatsTransferPath,
					ContentPath: validFormatsContentPath,
				},
			},
			wantErr: "ValidateFileFormats: open /dev/null/allowlist.csv: not a directory",
		},
		{
			name: "Errors when allowlist is empty",
			cfg:  fformat.Config{AllowlistPath: emptyAllowlistDir.Join("allowlist.csv")},
			params: activities.ValidateFileFormatsParams{
				SIP: sip.SIP{
					Type:        enums.SIPTypeDigitizedAIP,
					Path:        validFormatsTransferPath,
					ContentPath: validFormatsContentPath,
				},
			},
			wantErr: "ValidateFileFormats: load allowed formats: no allowed file formats",
		},
		{
			name: "Errors when allowlist is not a valid CSV format",
			cfg:  fformat.Config{AllowlistPath: invalidAllowListDir.Join("allowlist.csv")},
			params: activities.ValidateFileFormatsParams{
				SIP: sip.SIP{
					Type:        enums.SIPTypeDigitizedAIP,
					Path:        validFormatsTransferPath,
					ContentPath: validFormatsContentPath,
				},
			},
			wantErr: "ValidateFileFormats: load allowed formats: invalid CSV: record on line 2: wrong number of fields",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logbuf bytes.Buffer
			logger := buflogr.NewWithBuffer(&logbuf)

			ts := &temporalsdk_testsuite.WorkflowTestSuite{}
			env := ts.NewTestActivityEnvironment()
			env.SetWorkerOptions(temporalsdk_worker.Options{
				Interceptors: []temporalsdk_interceptor.WorkerInterceptor{
					temporal.NewLoggerInterceptor(logger),
				},
			})
			env.RegisterActivityWithOptions(
				activities.NewValidateFileFormats(tt.cfg).Execute,
				temporalsdk_activity.RegisterOptions{Name: activities.ValidateFileFormatsName},
			)

			enc, err := env.ExecuteActivity(activities.ValidateFileFormatsName, tt.params)
			if tt.wantErr != "" {
				prefix := "activity error (type: validate-file-formats, scheduledEventID: 0, startedEventID: 0, identity: ): "
				if err == nil {
					t.Errorf("error is nil, expecting: %q", tt.wantErr)
				} else {
					assert.Error(t, err, prefix+tt.wantErr)
				}

				return
			}
			assert.NilError(t, err)

			if tt.wantLog != "" {
				assert.Equal(t, logbuf.String(), tt.wantLog)
			}

			var result activities.ValidateFileFormatsResult
			_ = enc.Get(&result)
			assert.DeepEqual(t, result, tt.want)
		})
	}
}
