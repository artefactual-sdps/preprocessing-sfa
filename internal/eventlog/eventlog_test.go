package eventlog_test

import (
	"fmt"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/eventlog"
)

func TestEvent(t *testing.T) {
	t.Parallel()

	var (
		started   = time.Date(2024, 6, 6, 14, 48, 12, 0, time.UTC)
		completed = time.Date(2024, 6, 6, 14, 48, 13, 0, time.UTC)
	)

	t.Run("Event succeeds", func(t *testing.T) {
		t.Parallel()

		event := eventlog.NewEvent(started, "test event")
		event.Complete(
			completed,
			enums.EventOutcomeSuccess,
			"completed at %s",
			completed.Format(time.RFC3339),
		)
		assert.DeepEqual(t, event, &eventlog.Event{
			Name:        "test event",
			Message:     "completed at 2024-06-06T14:48:13Z",
			Outcome:     enums.EventOutcomeSuccess,
			StartedAt:   started,
			CompletedAt: completed,
		})
		assert.Equal(t, event.IsSuccess(), true)
	})

	t.Run("Event outcome is validation failure", func(t *testing.T) {
		t.Parallel()

		p := "/tmp/test-sip/additional/UpdatedAreldaMetadata.xml"

		event := eventlog.NewEvent(started, "test event")
		event.Complete(
			completed,
			enums.EventOutcomeValidationFailure,
			"Content error: metadata validation has failed: %s does not match expected metadata requirements",
			p,
		)
		assert.DeepEqual(t, event, &eventlog.Event{
			Name: "test event",
			Message: fmt.Sprintf(
				"Content error: metadata validation has failed: %s does not match expected metadata requirements",
				p,
			),
			Outcome:     enums.EventOutcomeValidationFailure,
			StartedAt:   started,
			CompletedAt: completed,
		})
		assert.Equal(t, event.IsSuccess(), false)
	})
}
