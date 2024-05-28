package activities_test

import (
	"testing"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
)

func TestIdentifyTransfer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		params activities.IdentifyTransferParams
		result activities.IdentifyTransferResult
	}{
		{
			name: "Identifies a VecteurAIP",
			params: activities.IdentifyTransferParams{
				Path: fs.NewDir(t, "", fs.WithDir("content"), fs.WithDir("additional")).Path(),
			},
			result: activities.IdentifyTransferResult{Type: enums.TransferTypeVecteurAIP},
		},
		{
			name: "Identifies a VecteurSIP",
			params: activities.IdentifyTransferParams{
				Path: fs.NewDir(t, "", fs.WithDir("content"), fs.WithDir("header")).Path(),
			},
			result: activities.IdentifyTransferResult{Type: enums.TransferTypeVecteurSIP},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := &temporalsdk_testsuite.WorkflowTestSuite{}
			env := ts.NewTestActivityEnvironment()
			env.RegisterActivityWithOptions(
				activities.NewIdentifyTransfer().Execute,
				temporalsdk_activity.RegisterOptions{Name: activities.IdentifyTransferName},
			)

			var res activities.IdentifyTransferResult
			future, err := env.ExecuteActivity(activities.IdentifyTransferName, tt.params)
			future.Get(&res)
			assert.NilError(t, err)
			assert.DeepEqual(t, res, tt.result)
		})
	}
}
