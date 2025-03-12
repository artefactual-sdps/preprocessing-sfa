package ais_test

import (
	"errors"
	"testing"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/ais"
	fake_amss "github.com/artefactual-sdps/preprocessing-sfa/internal/amss/fake"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"
)

func TestGetAIPPathActivity(t *testing.T) {
	t.Parallel()

	type test struct {
		name    string
		params  *ais.GetAIPPathActivityParams
		want    *ais.GetAIPPathActivityResult
		wantErr error
	}
	for _, tt := range []test{
		{
			name: "success",
			params: &ais.GetAIPPathActivityParams{
				AIPUUID: "test-uuid",
			},
			want: &ais.GetAIPPathActivityResult{Path: "test/path/METS.xml"},
		},
		{
			name: "error",
			params: &ais.GetAIPPathActivityParams{
				AIPUUID: "test-uuid",
			},
			want: &ais.GetAIPPathActivityResult{
				Path: "",
			},
			wantErr: errors.New("test error"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			amssClient := fake_amss.NewMockService(ctrl)

			// TODO: Define the expected behavior of the mock here.

			ts := &temporalsdk_testsuite.WorkflowTestSuite{}
			env := ts.NewTestActivityEnvironment()
			env.RegisterActivityWithOptions(
				ais.NewGetAIPPathActivity(amssClient).Execute,
				temporalsdk_activity.RegisterOptions{Name: ais.GetAIPPathActivityName},
			)

			future, err := env.ExecuteActivity(ais.GetAIPPathActivityName, tt.params)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("error is nil, expecting: %q", tt.wantErr)
				} else {
					assert.ErrorContains(t, err, tt.wantErr.Error())
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
