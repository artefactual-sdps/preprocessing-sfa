package workflow

import (
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/eventlog"
)

type eventWrapper struct {
	*eventlog.Event
}

func newWrappedEvent(ctx temporalsdk_workflow.Context, name string) *eventWrapper {
	return &eventWrapper{eventlog.NewEvent(temporalsdk_workflow.Now(ctx), name)}
}

func (w *eventWrapper) Complete(
	ctx temporalsdk_workflow.Context,
	outcome enums.EventOutcome,
	msg string,
	notes ...string,
) *eventWrapper {
	w.Event.Complete(temporalsdk_workflow.Now(ctx), outcome, msg, notes)
	return w
}

func (w *eventWrapper) Succeed(
	ctx temporalsdk_workflow.Context,
	msg string,
	notes ...string,
) *eventWrapper {
	return w.Complete(ctx, enums.EventOutcomeSuccess, msg, notes...)
}
