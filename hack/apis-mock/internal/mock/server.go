package mock

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/artefactual-sdps/preprocessing-sfa/hack/apis-mock/internal/gen"
)

// Security implements the APIS authentication scheme for the generated server.
type Security struct {
	token string
}

// NewSecurity returns the APIS mock security handler with the given static token.
func NewSecurity(token string) *Security {
	return &Security{token: token}
}

// HandleSmart keeps authentication deliberately small for local development.
// Every request must present one known bearer token.
func (s *Security) HandleSmart(ctx context.Context, _ gen.OperationName, t gen.Smart) (context.Context, error) {
	if t.Token != s.token {
		return nil, errors.New("invalid bearer token")
	}
	return ctx, nil
}

// Handler keeps just enough state to reproduce the APIS workflow:
// analysis, optional cancellation, and one import run.
type Handler struct {
	mu       sync.Mutex
	nextTask int
	nextRun  int
	tasks    map[string]*taskState
}

// taskState stores the minimal in-memory state for an APIS import task.
type taskState struct {
	id             string
	name           string
	createdBy      string
	createdAt      time.Time
	status         gen.ImportTaskStatus
	analysisPolls  int
	importPolls    int
	analysisDone   bool
	importDone     bool
	analysisResult gen.AnalysisResult
	importResult   gen.ImportResult
	runID          string
}

// NewHandler returns the in-memory APIS mock handler.
func NewHandler() *Handler {
	return &Handler{tasks: make(map[string]*taskState)}
}

// APIHealthzGet is left unimplemented because the preprocessing
// integration work only depends on the import-task endpoints.
func (h *Handler) APIHealthzGet(_ context.Context) (gen.APIHealthzGetRes, error) {
	return &gen.APIHealthzGetOK{Status: "OK"}, nil
}

// APIImporttasksPost creates one in-memory task with a predetermined analysis
// outcome. Later polling simply walks that task through the expected APIS
// lifecycle until the terminal result becomes visible.
func (h *Handler) APIImporttasksPost(
	_ context.Context,
	req gen.OptAPIImporttasksPostReq,
) (gen.APIImporttasksPostRes, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextTask++
	taskID := fmt.Sprintf("task-%06d", h.nextTask)
	task := &taskState{
		id:             taskID,
		createdAt:      time.Now().UTC(),
		status:         gen.ImportTaskStatusNeu,
		analysisResult: gen.AnalysisResultAlleNeu,
		importResult:   gen.ImportResultErfolgreich,
	}

	if r, ok := req.Get(); ok {
		task.name = r.File.Name
		task.createdBy = r.Username
		if result, ok := analysisResultFromName(r.File.Name); ok {
			task.analysisResult = result
		}
	}

	h.tasks[taskID] = task

	return &gen.CreateImportTaskResponse{
		ImportTaskId: taskID,
	}, nil
}

// APIImporttasksIDCancelPost only supports the cancellation transition used by the
// preprocessing conflict flow.
func (h *Handler) APIImporttasksIDCancelPost(
	_ context.Context,
	_ gen.APIImporttasksIDCancelPostReq,
	params gen.APIImporttasksIDCancelPostParams,
) (gen.APIImporttasksIDCancelPostRes, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	task, ok := h.tasks[params.ID]
	if !ok {
		notFound := gen.APIImporttasksIDCancelPostNotFound(
			problem(404, "Not Found", "import task does not exist"),
		)
		return &notFound, nil
	}

	if task.status == gen.ImportTaskStatusImportiert {
		conflict := gen.APIImporttasksIDCancelPostConflict(
			problem(409, "Conflict", "cannot cancel an already imported task"),
		)
		return &conflict, nil
	}

	task.status = gen.ImportTaskStatusAbgebrochen

	return &gen.APIImporttasksIDCancelPostNoContent{}, nil
}

// APIImporttasksIDImportrunsPost starts the import phase after a successful
// analysis result. The POST itself only allocates the run; subsequent polling
// on the task status endpoint exposes the import lifecycle.
func (h *Handler) APIImporttasksIDImportrunsPost(
	_ context.Context,
	req gen.OptAPIImporttasksIDImportrunsPostReq,
	params gen.APIImporttasksIDImportrunsPostParams,
) (gen.APIImporttasksIDImportrunsPostRes, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	task, ok := h.tasks[params.ID]
	if !ok {
		notFound := gen.APIImporttasksIDImportrunsPostNotFound(
			problem(404, "Not Found", "import task does not exist"),
		)
		return &notFound, nil
	}
	if task.status == gen.ImportTaskStatusAbgebrochen {
		return badRequest("cannot create import run for canceled task"), nil
	}
	if task.status != gen.ImportTaskStatusAnalysiert {
		return badRequest("cannot create import run before analysis has finished"), nil
	}
	if task.analysisResult != gen.AnalysisResultAlleNeu && task.analysisResult != gen.AnalysisResultAlleGleich {
		return badRequest("cannot create import run for this analysis result"), nil
	}
	if task.runID != "" {
		return badRequest("cannot create a second import run for the same task"), nil
	}

	h.nextRun++
	runID := fmt.Sprintf("run-%06d", h.nextRun)
	task.runID = runID
	task.status = gen.ImportTaskStatusWirdImportiert
	task.importPolls = 0
	task.importResult = gen.ImportResultErfolgreich
	if r, ok := req.Get(); ok {
		if result, ok := importResultFromName(r.File.Name); ok {
			task.importResult = result
		}
	}

	return &gen.CreateImportRunResponse{
		ImportRunId: runID,
	}, nil
}

// APIImporttasksIDImportrunsRunIdStatusGet mirrors the current import phase in
// the lighter-weight run-specific vocabulary from the spec. It never advances
// state on its own; the task status endpoint remains the single lifecycle
// driver for the mock.
func (h *Handler) APIImporttasksIDImportrunsRunIdStatusGet(
	_ context.Context,
	params gen.APIImporttasksIDImportrunsRunIdStatusGetParams,
) (gen.APIImporttasksIDImportrunsRunIdStatusGetRes, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	task, ok := h.tasks[params.ID]
	if !ok {
		notFound := gen.APIImporttasksIDImportrunsRunIdStatusGetNotFound(
			problem(404, "Not Found", "import task does not exist"),
		)
		return &notFound, nil
	}
	if task.runID == "" || task.runID != params.RunId {
		notFound := gen.APIImporttasksIDImportrunsRunIdStatusGetNotFound(
			problem(404, "Not Found", "import run does not exist"),
		)
		return &notFound, nil
	}

	return importRunStatusResponse(task), nil
}

// APIImporttasksIDStatusGet is the canonical lifecycle endpoint for the mock.
// Each poll advances analysis or import by one deterministic step because the
// real integration also treats this endpoint as the main source of truth.
//
// The mock still returns a proper 404 response for unknown task IDs instead of
// fabricating a happy-path task because that catches worker bugs earlier.
func (h *Handler) APIImporttasksIDStatusGet(
	_ context.Context,
	params gen.APIImporttasksIDStatusGetParams,
) (gen.APIImporttasksIDStatusGetRes, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	task, ok := h.tasks[params.ID]
	if !ok {
		notFound := gen.APIImporttasksIDStatusGetNotFound(
			problem(404, "Not Found", "import task does not exist"),
		)
		return &notFound, nil
	}

	advanceTask(task)

	return taskStatusResponse(task), nil
}

// advanceTask encodes the smallest state machine that still matches the flows
// described in issues 173 and 174.
func advanceTask(task *taskState) {
	switch task.status {
	case gen.ImportTaskStatusNeu:
		task.analysisPolls++
		if task.analysisPolls > 1 {
			task.status = gen.ImportTaskStatusInAnalyse
		}
	case gen.ImportTaskStatusInAnalyse:
		task.analysisDone = true
		task.status = gen.ImportTaskStatusAnalysiert
	case gen.ImportTaskStatusWirdImportiert:
		task.importPolls++
		if task.importPolls > 1 {
			task.importDone = true
			task.status = gen.ImportTaskStatusImportiert
		}
	}
}

// taskStatusResponse only returns the fields preprocessing-sfa currently uses:
// the task status plus terminal analysis/import results.
func taskStatusResponse(task *taskState) *gen.ImportTaskStatusResponse {
	res := &gen.ImportTaskStatusResponse{Status: task.status}

	if task.analysisDone {
		res.AnalysisResult = gen.NewOptNilAnalysisResult(task.analysisResult)
	}
	if task.importDone {
		res.ImportResult = gen.NewOptNilImportResult(task.importResult)
	}

	return res
}

// importRunStatusResponse projects task import state onto the run status
// endpoint for callers that still want a run-specific view.
func importRunStatusResponse(task *taskState) *gen.ImportRunStatusResponse {
	res := &gen.ImportRunStatusResponse{Status: gen.ImportStatusCreated}

	switch task.status {
	case gen.ImportTaskStatusAbgebrochen:
		res.Status = gen.ImportStatusCanceled
	case gen.ImportTaskStatusWirdImportiert:
		res.Status = gen.ImportStatusStarted
	case gen.ImportTaskStatusImportiert:
		if task.importResult == gen.ImportResultFehler {
			res.Status = gen.ImportStatusFailed
		} else {
			res.Status = gen.ImportStatusCompleted
		}
		res.ImportResult = gen.NewOptImportResult(task.importResult)
	}

	return res
}

// problem builds the common RFC 7807-style response payloads used by the
// generated APIS error responses.
func problem(status int32, title, detail string) gen.ProblemDetails {
	return gen.ProblemDetails{
		Status: gen.NewOptNilInt32(status),
		Title:  gen.NewOptNilString(title),
		Detail: gen.NewOptNilString(detail),
	}
}

func badRequest(detail string) *gen.APIImporttasksIDImportrunsPostBadRequest {
	return &gen.APIImporttasksIDImportrunsPostBadRequest{
		Status: gen.NewOptNilInt32(400),
		Title:  gen.NewOptNilString("Bad Request"),
		Detail: gen.NewOptNilString(detail),
	}
}

// analysisResultFromName maps filename markers to analysis outcomes for manual and
// automated development scenarios.
func analysisResultFromName(value string) (gen.AnalysisResult, bool) {
	switch {
	case strings.Contains(value, "mock-fehler"):
		return gen.AnalysisResultFehler, true
	case strings.Contains(value, "mock-konflikte"):
		return gen.AnalysisResultKonflikte, true
	case strings.Contains(value, "mock-allegleich"):
		return gen.AnalysisResultAlleGleich, true
	case strings.Contains(value, "mock-alleneu"):
		return gen.AnalysisResultAlleNeu, true
	default:
		return "", false
	}
}

// importResultFromName maps filename markers to import outcomes for manual and
// automated development scenarios.
func importResultFromName(value string) (gen.ImportResult, bool) {
	switch {
	case strings.Contains(value, "mock-import-fehler"):
		return gen.ImportResultFehler, true
	case strings.Contains(value, "mock-import-erfolgreich"):
		return gen.ImportResultErfolgreich, true
	default:
		return "", false
	}
}
