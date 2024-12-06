package workflow

import (
	"github.com/google/uuid"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"
)

func GenUUID(ctx temporalsdk_workflow.Context) (uuid.UUID, error) {
	var id uuid.UUID
	gen := func(ctx temporalsdk_workflow.Context) interface{} { return uuid.New() }
	enc := temporalsdk_workflow.SideEffect(ctx, gen)
	err := enc.Get(&id)
	return id, err
}
