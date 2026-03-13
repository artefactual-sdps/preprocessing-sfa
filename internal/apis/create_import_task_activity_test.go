package apis_test

import (
	"context"
	"errors"
	"io"
	"testing"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/apis"
	fake_apis "github.com/artefactual-sdps/preprocessing-sfa/internal/apis/fake"
	apisgen "github.com/artefactual-sdps/preprocessing-sfa/internal/apis/gen"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

func TestCreateImportTaskActivity(t *testing.T) {
	t.Parallel()

	sipValue, err := sip.New(fs.NewDir(t, "",
		fs.WithDir("content",
			fs.WithDir("d_0000001",
				fs.WithFile("content.txt", "payload"),
			),
		),
		fs.WithDir("header",
			fs.WithFile("metadata.xml", "metadata-body"),
			fs.WithDir("xsd",
				fs.WithFile("arelda.xsd", ""),
			),
		),
	).Path())
	assert.NilError(t, err)

	tests := []struct {
		name    string
		params  apis.CreateImportTaskParams
		expect  func(*fake_apis.MockClientMockRecorder)
		want    apis.CreateImportTaskResult
		wantErr string
	}{
		{
			name: "creates import task",
			params: apis.CreateImportTaskParams{
				SIP:      sipValue,
				Username: "archivist@example.com",
			},
			expect: func(m *fake_apis.MockClientMockRecorder) {
				m.APIImportTasksPost(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, req apisgen.OptAPIImportTasksPostReq) (apisgen.APIImportTasksPostRes, error) {
						payload, ok := req.Get()
						assert.Assert(t, ok)
						assert.Equal(t, payload.Username, "archivist@example.com")
						assert.Equal(t, payload.SipType, apisgen.SipTypeBornDigitalSIP)
						assert.Equal(t, payload.File.Name, "metadata.xml")

						body, err := io.ReadAll(payload.File.File)
						assert.NilError(t, err)
						assert.Equal(t, string(body), "metadata-body")

						res := apisgen.APIImportTasksPostCreated{
							Success: apisgen.NewOptBool(true),
							ID:      apisgen.NewOptNilString("task-000001"),
						}
						return &res, nil
					},
				)
			},
			want: apis.CreateImportTaskResult{TaskID: "task-000001"},
		},
		{
			name: "passes empty username through",
			params: apis.CreateImportTaskParams{
				SIP: sipValue,
			},
			expect: func(m *fake_apis.MockClientMockRecorder) {
				m.APIImportTasksPost(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, req apisgen.OptAPIImportTasksPostReq) (apisgen.APIImportTasksPostRes, error) {
						payload, ok := req.Get()
						assert.Assert(t, ok)
						assert.Equal(t, payload.Username, "")

						res := apisgen.APIImportTasksPostCreated{
							Success: apisgen.NewOptBool(true),
							ID:      apisgen.NewOptNilString("task-000002"),
						}
						return &res, nil
					},
				)
			},
			want: apis.CreateImportTaskResult{TaskID: "task-000002"},
		},
		{
			name: "returns response error",
			params: apis.CreateImportTaskParams{
				SIP: sipValue,
			},
			expect: func(m *fake_apis.MockClientMockRecorder) {
				m.APIImportTasksPost(gomock.Any(), gomock.Any()).Return(
					&apisgen.APIImportTasksPostBadRequest{
						Success: apisgen.NewOptBool(false),
						Error:   apisgen.NewOptNilString("invalid payload"),
					},
					nil,
				)
			},
			wantErr: "invalid payload",
		},
		{
			name: "returns client error",
			params: apis.CreateImportTaskParams{
				SIP: sipValue,
			},
			expect: func(m *fake_apis.MockClientMockRecorder) {
				m.APIImportTasksPost(gomock.Any(), gomock.Any()).Return(nil, errors.New("boom"))
			},
			wantErr: "boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			client := fake_apis.NewMockClient(ctrl)
			if tt.expect != nil {
				tt.expect(client.EXPECT())
			}

			suite := temporalsdk_testsuite.WorkflowTestSuite{}
			env := suite.NewTestActivityEnvironment()
			env.RegisterActivityWithOptions(
				apis.NewCreateImportTaskActivity(client).Execute,
				temporalsdk_activity.RegisterOptions{Name: apis.CreateImportTaskActivityName},
			)

			future, err := env.ExecuteActivity(apis.CreateImportTaskActivityName, &tt.params)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q", tt.wantErr)
				}
				assert.ErrorContains(t, err, tt.wantErr)

				return
			}
			assert.NilError(t, err)

			var result apis.CreateImportTaskResult
			assert.NilError(t, future.Get(&result))
			assert.DeepEqual(t, result, tt.want)
		})
	}
}
