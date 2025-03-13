package ais_test

import (
	"errors"
	"testing"

	"go.artefactual.dev/tools/mockutil"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/ais"
	fake_amss "github.com/artefactual-sdps/preprocessing-sfa/internal/amss/fake"
)

func TestGetAIPPathActivity(t *testing.T) {
	t.Parallel()

	type test struct {
		name       string
		params     *ais.GetAIPPathActivityParams
		mockExpect func(m *fake_amss.MockClient)
		want       *ais.GetAIPPathActivityResult
		wantErr    string
	}
	for _, tt := range []test{
		{
			name: "success",
			params: &ais.GetAIPPathActivityParams{
				AIPUUID: "test-uuid",
			},
			mockExpect: func(m *fake_amss.MockClient) {
				m.EXPECT().GetAIPPath(mockutil.Context(), "test-uuid").Return("test/path/METS.xml", nil)
			},
			want: &ais.GetAIPPathActivityResult{Path: "test/path/METS.xml"},
		},
		{
			name: "error",
			params: &ais.GetAIPPathActivityParams{
				AIPUUID: "test-uuid",
			},
			mockExpect: func(m *fake_amss.MockClient) {
				m.EXPECT().GetAIPPath(mockutil.Context(), "test-uuid").Return("", errors.New("test error"))
			},
			wantErr: "test error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockClient := fake_amss.NewMockClient(ctrl)

			if tt.mockExpect != nil {
				tt.mockExpect(mockClient)
			}

			ts := &temporalsdk_testsuite.WorkflowTestSuite{}
			env := ts.NewTestActivityEnvironment()
			env.RegisterActivityWithOptions(
				ais.NewGetAIPPathActivity(mockClient).Execute,
				temporalsdk_activity.RegisterOptions{Name: ais.GetAIPPathActivityName},
			)

			future, err := env.ExecuteActivity(ais.GetAIPPathActivityName, tt.params)
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("error is nil, expecting: %q", tt.wantErr)
				} else {
					assert.ErrorContains(t, err, tt.wantErr)
				}

				return
			}
			assert.NilError(t, err)

			var got ais.GetAIPPathActivityResult
			future.Get(&got)
			assert.DeepEqual(t, &got, tt.want)
		})
	}
}
