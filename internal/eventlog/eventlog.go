package eventlog

import (
	"time"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
)

type Event struct {
	Name        string
	Message     string
	Outcome     enums.EventOutcome
	StartedAt   time.Time
	CompletedAt time.Time
}

func NewEvent(t time.Time, name string) *Event {
	return &Event{
		Name:      name,
		Outcome:   enums.EventOutcomeUnspecified,
		StartedAt: t,
	}
}

func (e *Event) IsSuccess() bool {
	return e.Outcome == enums.EventOutcomeSuccess
}

func (e *Event) Complete(t time.Time, outcome enums.EventOutcome, msg string) *Event {
	e.CompletedAt = t
	e.Outcome = outcome
	e.Message = msg

	return e
}
