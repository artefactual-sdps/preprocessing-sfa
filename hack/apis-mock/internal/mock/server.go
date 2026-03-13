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
	id              string
	seqID           int32
	name            string
	createdBy       string
	createdAt       time.Time
	status          gen.ImportTaskStatus
	analysisPolls   int
	importPolls     int
	analysisDone    bool
	importDone      bool
	analysisOutcome gen.AnalysisResult
	importOutcome   gen.ImportResult
	runID           string
}

// NewHandler returns the in-memory APIS mock handler.
func NewHandler() *Handler {
	return &Handler{tasks: make(map[string]*taskState)}
}

// APIHealthzGet is left unimplemented because the preprocessing
// integration work only depends on the import-task endpoints.
func (h *Handler) APIHealthzGet(_ context.Context) (gen.APIHealthzGetRes, error) {
	return nil, errors.New("not implemented")
}

// APIImportTasksPost creates one in-memory task with a predetermined analysis
// outcome. Later polling simply walks that task through the expected APIS
// lifecycle until the terminal result becomes visible.
func (h *Handler) APIImportTasksPost(
	_ context.Context,
	req gen.OptAPIImportTasksPostReq,
) (gen.APIImportTasksPostRes, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextTask++
	taskID := fmt.Sprintf("task-%06d", h.nextTask)
	task := &taskState{
		id:              taskID,
		seqID:           int32(h.nextTask),
		createdAt:       time.Now().UTC(),
		status:          gen.ImportTaskStatusNeu,
		analysisOutcome: gen.AnalysisResultAlleNeu,
		importOutcome:   gen.ImportResultErfolgreich,
	}

	if req, ok := req.Get(); ok {
		task.name = req.File.Name
		task.createdBy = req.Username
		if result, ok := analysisResultHint(req.File.Name); ok {
			task.analysisOutcome = result
		}
	}

	h.tasks[taskID] = task

	return &gen.APIImportTasksPostCreated{
		Success: gen.NewOptBool(true),
		ID:      gen.NewOptNilString(taskID),
	}, nil
}

// APIImportTasksIDPatch only supports the cancellation transition used by the
// preprocessing conflict flow.
func (h *Handler) APIImportTasksIDPatch(
	_ context.Context,
	req gen.APIImportTasksIDPatchReq,
	params gen.APIImportTasksIDPatchParams,
) (gen.APIImportTasksIDPatchRes, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	task, ok := h.tasks[params.ID]
	if !ok {
		notFound := gen.APIImportTasksIDPatchNotFound(
			problem(404, "Not Found", "import task does not exist"),
		)
		return &notFound, nil
	}

	status, ok := patchStatus(req)
	if !ok || status != gen.ImportTaskStatusAbgebrochen {
		conflict := gen.APIImportTasksIDPatchConflict(
			problem(409, "Conflict", "only status=Abgebrochen can be applied"),
		)
		return &conflict, nil
	}
	if task.status == gen.ImportTaskStatusImportiert {
		conflict := gen.APIImportTasksIDPatchConflict(
			problem(409, "Conflict", "cannot cancel an already imported task"),
		)
		return &conflict, nil
	}

	task.status = gen.ImportTaskStatusAbgebrochen

	return taskDTO(task), nil
}

// APIImportTasksIDImportRunsPost starts the import phase after a successful
// analysis result. The POST itself only allocates the run; subsequent polling
// on the task status endpoint exposes the import lifecycle.
func (h *Handler) APIImportTasksIDImportRunsPost(
	_ context.Context,
	req gen.OptAPIImportTasksIDImportRunsPostReq,
	params gen.APIImportTasksIDImportRunsPostParams,
) (gen.APIImportTasksIDImportRunsPostRes, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	task, ok := h.tasks[params.ID]
	if !ok {
		notFound := gen.APIImportTasksIDImportRunsPostNotFound(
			problem(404, "Not Found", "import task does not exist"),
		)
		return &notFound, nil
	}
	if task.status == gen.ImportTaskStatusAbgebrochen {
		return validationProblem(
			"cannot create import run for canceled task",
			"status",
			"task is canceled",
		), nil
	}
	if task.status != gen.ImportTaskStatusAnalysiert {
		return validationProblem(
			"cannot create import run before analysis has finished",
			"status",
			"task analysis is still running",
		), nil
	}
	if task.analysisOutcome != gen.AnalysisResultAlleNeu && task.analysisOutcome != gen.AnalysisResultAlleGleich {
		return validationProblem(
			"cannot create import run for this analysis result",
			"analysisResult",
			string(task.analysisOutcome),
		), nil
	}
	if task.runID != "" {
		return validationProblem(
			"cannot create a second import run for the same task",
			"status",
			"task already has an import run",
		), nil
	}

	h.nextRun++
	runID := fmt.Sprintf("run-%06d", h.nextRun)
	task.runID = runID
	task.status = gen.ImportTaskStatusWirdImportiert
	task.importPolls = 0
	task.importOutcome = gen.ImportResultErfolgreich
	if req, ok := req.Get(); ok {
		if result, ok := importResultHint(req.File.Name); ok {
			task.importOutcome = result
		}
	}

	return &gen.CreateImportRunResponse{
		Success:      gen.NewOptBool(true),
		ImportTaskId: gen.NewOptString(task.id),
		ImportRunId:  gen.NewOptNilString(runID),
	}, nil
}

// APIImportTasksIDImportRunsRunIdStatusGet mirrors the current import phase in
// the lighter-weight run-specific vocabulary from the spec. It never advances
// state on its own; the task status endpoint remains the single lifecycle
// driver for the mock.
func (h *Handler) APIImportTasksIDImportRunsRunIdStatusGet(
	_ context.Context,
	params gen.APIImportTasksIDImportRunsRunIdStatusGetParams,
) (gen.APIImportTasksIDImportRunsRunIdStatusGetRes, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	task, ok := h.tasks[params.ID]
	if !ok {
		notFound := gen.APIImportTasksIDImportRunsRunIdStatusGetNotFound(
			problem(404, "Not Found", "import task does not exist"),
		)
		return &notFound, nil
	}
	if task.runID == "" || task.runID != params.RunId {
		notFound := gen.APIImportTasksIDImportRunsRunIdStatusGetNotFound(
			problem(404, "Not Found", "import run does not exist"),
		)
		return &notFound, nil
	}

	return importRunStatusResponse(task), nil
}

// APIImportTasksIDStatusGet is the canonical lifecycle endpoint for the mock.
// Each poll advances analysis or import by one deterministic step because the
// real integration also treats this endpoint as the main source of truth.
//
// The shared spec only models a 200 response here. The mock still fails fast
// for unknown task IDs instead of fabricating a happy-path task because that
// catches worker bugs earlier during development.
func (h *Handler) APIImportTasksIDStatusGet(
	_ context.Context,
	params gen.APIImportTasksIDStatusGetParams,
) (*gen.ImportTaskStatusResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	task, ok := h.tasks[params.ID]
	if !ok {
		return nil, fmt.Errorf("import task does not exist: %s", params.ID)
	}

	advanceTask(task)

	return taskStatusResponse(task), nil
}

// patchStatus extracts the requested task status from either generated PATCH
// request wrapper used by ogen.
func patchStatus(req gen.APIImportTasksIDPatchReq) (gen.ImportTaskStatus, bool) {
	switch req := req.(type) {
	case *gen.CancelImportTaskRequest:
		return req.Status, true
	case *gen.CancelImportTaskRequestWithContentType:
		return req.Content.Status, true
	default:
		return "", false
	}
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
		res.AnalysisResult = gen.NewOptAnalysisResult(task.analysisOutcome)
	}
	if task.importDone {
		res.ImportResult = gen.NewOptImportResult(task.importOutcome)
	}

	return res
}

// importRunStatusResponse projects task import state onto the run status
// endpoint for callers that still want a run-specific view.
func importRunStatusResponse(task *taskState) *gen.ImportRunStatusResponse {
	status := "Queued"

	switch task.status {
	case gen.ImportTaskStatusAbgebrochen:
		status = "Canceled"
	case gen.ImportTaskStatusWirdImportiert:
		status = "Running"
	case gen.ImportTaskStatusImportiert:
		if task.importOutcome == gen.ImportResultFehler {
			status = "Failed"
		} else {
			status = "Completed"
		}
	}

	res := &gen.ImportRunStatusResponse{
		ImportTaskId: gen.NewOptInt32(task.seqID),
		ImportRunId:  gen.NewOptString(task.runID),
		Status:       gen.NewOptString(status),
	}
	if status == "Failed" {
		res.Error = gen.NewOptNilString("mock import failed")
	}

	return res
}

// taskDTO builds the narrow PATCH response payload needed by clients after a
// cancellation request.
func taskDTO(task *taskState) *gen.ImportTaskDto {
	dto := &gen.ImportTaskDto{
		ImportTaskId:       gen.NewOptInt32(task.seqID),
		RowVersion:         []byte{},
		TaskType:           gen.NewOptString("MockImportTask"),
		Name:               gen.NewOptNilString(task.name),
		NoAutomaticImport:  gen.NewOptBool(false),
		RequiresContainers: gen.NewOptBool(false),
		Status:             gen.NewOptString(string(task.status)),
		CreatedBy:          gen.NewOptString(task.createdBy),
		CreatedOn:          gen.NewOptDateTime(task.createdAt),
		AnalysisRecords:    []gen.AnalysisRecordDto{},
		DefaultValues:      []gen.DefaultValueDto{},
		Documents:          []gen.DocumentDto{},
		Imports:            []gen.ImportDto{},
	}
	if task.analysisDone {
		dto.AnalysisResult = gen.NewOptNilString(string(task.analysisOutcome))
	}
	if task.importDone {
		dto.ImportResult = gen.NewOptNilString(string(task.importOutcome))
	}

	return dto
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

// validationProblem returns the field-oriented 400 response shape generated
// from the APIS spec for rejected import-run requests.
func validationProblem(detail, field, message string) *gen.ValidationProblemDetails {
	return &gen.ValidationProblemDetails{
		Status: gen.NewOptNilInt32(400),
		Title:  gen.NewOptNilString("Bad Request"),
		Detail: gen.NewOptNilString(detail),
		Errors: gen.NewOptValidationProblemDetailsErrors(
			gen.ValidationProblemDetailsErrors{field: {message}},
		),
	}
}

// analysisResultHint maps filename markers to analysis outcomes for manual and
// automated development scenarios.
func analysisResultHint(value string) (gen.AnalysisResult, bool) {
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

// importResultHint maps filename markers to import outcomes for manual and
// automated development scenarios.
func importResultHint(value string) (gen.ImportResult, bool) {
	switch {
	case strings.Contains(value, "mock-import-fehler"):
		return gen.ImportResultFehler, true
	case strings.Contains(value, "mock-import-erfolgreich"):
		return gen.ImportResultErfolgreich, true
	default:
		return "", false
	}
}
