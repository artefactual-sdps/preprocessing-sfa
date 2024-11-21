package activities_test

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/fformat"
	fake_fformat "github.com/artefactual-sdps/preprocessing-sfa/internal/fformat/fake"
	fake_fvalidate "github.com/artefactual-sdps/preprocessing-sfa/internal/fvalidate/fake"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

func TestValidateFiles(t *testing.T) {
	t.Parallel()

	digitizedAIP, err := sip.New(fs.NewDir(t, "",
		fs.WithDir("additional",
			fs.WithFile("UpdatedAreldaMetadata.xml", ""),
		),
		fs.WithDir("content",
			fs.WithDir("content",
				fs.WithDir("d_0000001",
					fs.WithFile("test.pdf", ""),
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
	).Path())
	assert.NilError(t, err)

	tests := []struct {
		name      string
		params    activities.ValidateFilesParams
		expectId  func(*fake_fformat.MockIdentifierMockRecorder)
		expectVld func(*fake_fvalidate.MockValidatorMockRecorder)
		want      activities.ValidateFilesResult
		wantErr   string
	}{
		{
			name:   "Validates a PDF/A file",
			params: activities.ValidateFilesParams{SIP: digitizedAIP},
			expectId: func(m *fake_fformat.MockIdentifierMockRecorder) {
				m.Identify(
					filepath.Join(digitizedAIP.ContentPath, "d_0000001", "test.pdf"),
				).Return(
					&fformat.FileFormat{
						Namespace: "PRONOM",
						ID:        "fmt/354",
					},
					nil,
				)
				m.Identify(
					filepath.Join(digitizedAIP.ContentPath, "d_0000001", "Prozess_Digitalisierung_PREMIS.xml"),
				).Return(
					&fformat.FileFormat{
						Namespace: "PRONOM",
						ID:        "fmt/101",
					},
					nil,
				)
			},
			expectVld: func(m *fake_fvalidate.MockValidatorMockRecorder) {
				m.FormatIDs().Return([]string{"fmt/354"})
				m.Validate(digitizedAIP.ContentPath).Return("", nil)
			},
		},
		{
			name:   "Reports PDF validation errors",
			params: activities.ValidateFilesParams{SIP: digitizedAIP},
			expectId: func(m *fake_fformat.MockIdentifierMockRecorder) {
				m.Identify(
					filepath.Join(digitizedAIP.ContentPath, "d_0000001", "test.pdf"),
				).Return(
					&fformat.FileFormat{
						Namespace: "PRONOM",
						ID:        "fmt/354",
					},
					nil,
				)
				m.Identify(
					filepath.Join(digitizedAIP.ContentPath, "d_0000001", "Prozess_Digitalisierung_PREMIS.xml"),
				).Return(
					&fformat.FileFormat{
						Namespace: "PRONOM",
						ID:        "fmt/101",
					},
					nil,
				)
			},
			expectVld: func(m *fake_fvalidate.MockValidatorMockRecorder) {
				m.FormatIDs().Return([]string{"fmt/354"})
				m.Validate(digitizedAIP.ContentPath).Return(
					"One or more PDF/A files are invalid",
					nil,
				)
			},
			want: activities.ValidateFilesResult{
				Failures: []string{"One or more PDF/A files are invalid"},
			},
		},
		{
			name:   "Errors on application error",
			params: activities.ValidateFilesParams{SIP: digitizedAIP},
			expectId: func(m *fake_fformat.MockIdentifierMockRecorder) {
				m.Identify(
					filepath.Join(digitizedAIP.ContentPath, "d_0000001", "test.pdf"),
				).Return(
					&fformat.FileFormat{
						Namespace: "PRONOM",
						ID:        "fmt/354",
					},
					nil,
				)
				m.Identify(
					filepath.Join(digitizedAIP.ContentPath, "d_0000001", "Prozess_Digitalisierung_PREMIS.xml"),
				).Return(
					&fformat.FileFormat{
						Namespace: "PRONOM",
						ID:        "fmt/101",
					},
					nil,
				)
			},
			expectVld: func(m *fake_fvalidate.MockValidatorMockRecorder) {
				m.FormatIDs().Return([]string{"fmt/354"})
				m.Validate(digitizedAIP.ContentPath).Return(
					"",
					errors.New("can't open /fake/path: permission denied"),
				)
			},
			wantErr: "can't open /fake/path: permission denied",
		},
		{
			name:   "Skip validation when format identification fails",
			params: activities.ValidateFilesParams{SIP: digitizedAIP},
			expectId: func(m *fake_fformat.MockIdentifierMockRecorder) {
				m.Identify(
					filepath.Join(digitizedAIP.ContentPath, "d_0000001", "test.pdf"),
				).Return(
					nil,
					fmt.Errorf(
						"multiple file formats matched: %s",
						filepath.Join(digitizedAIP.ContentPath, "d_0000001", "test.pdf"),
					),
				)
				m.Identify(
					filepath.Join(digitizedAIP.ContentPath, "d_0000001", "Prozess_Digitalisierung_PREMIS.xml"),
				).Return(
					&fformat.FileFormat{
						Namespace: "PRONOM",
						ID:        "fmt/101",
					},
					nil,
				)
			},
			expectVld: func(m *fake_fvalidate.MockValidatorMockRecorder) {
				m.FormatIDs().Return([]string{"fmt/354"})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := &temporalsdk_testsuite.WorkflowTestSuite{}
			env := ts.NewTestActivityEnvironment()

			ctrl := gomock.NewController(t)

			mockIdr := fake_fformat.NewMockIdentifier(ctrl)
			if tt.expectId != nil {
				tt.expectId(mockIdr.EXPECT())
			}

			mockVdr := fake_fvalidate.NewMockValidator(ctrl)
			if tt.expectVld != nil {
				tt.expectVld(mockVdr.EXPECT())
			}

			env.RegisterActivityWithOptions(
				activities.NewValidateFiles(mockIdr, mockVdr).Execute,
				temporalsdk_activity.RegisterOptions{Name: activities.ValidateFilesName},
			)

			enc, err := env.ExecuteActivity(activities.ValidateFilesName, tt.params)
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("error is nil, expecting: %q", tt.wantErr)
				} else {
					assert.ErrorContains(t, err, tt.wantErr)
				}

				return
			}
			assert.NilError(t, err)

			var result activities.ValidateFilesResult
			_ = enc.Get(&result)

			assert.DeepEqual(t, result, tt.want)
		})
	}
}
