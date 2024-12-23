package workflow_test

import (
	"testing"

	"github.com/google/uuid"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/workflow"
)

func TestGenUUID(t *testing.T) {
	t.Parallel()

	testSuite := &temporalsdk_testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.ExecuteWorkflow(workflow.GenUUID)

	var id uuid.UUID
	err := env.GetWorkflowResult(&id)
	assert.NilError(t, err)

	assert.NilError(t, env.GetWorkflowError())
	assert.Assert(t, id != uuid.Nil)
}
