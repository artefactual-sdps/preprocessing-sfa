package workflow

const (
	DecisionRequestSignal  = "decision-request-signal"
	DecisionResponseSignal = "decision-response-signal"

	DecisionOptionCancelIngest      = "Cancel ingest"
	DecisionOptionContinueOverwrite = "Continue and overwrite"
	DecisionOptionContinueAppend    = "Continue and append"
)

type DecisionRequest struct {
	Message string
	Options []string
}

type DecisionResponse struct {
	Option string
}
