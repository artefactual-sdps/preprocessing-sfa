package mock_test

import (
	"context"
	"testing"

	ogenhttp "github.com/ogen-go/ogen/http"
	"gotest.tools/v3/assert"

	"github.com/artefactual-sdps/preprocessing-sfa/hack/apis-mock/internal/gen"
	"github.com/artefactual-sdps/preprocessing-sfa/hack/apis-mock/internal/mock"
)

func TestTaskStatusDrivesAnalysisAndImportLifecycle(t *testing.T) {
	ctx := t.Context()
	h := mock.NewHandler()

	taskID := createTask(t, ctx, h, "metadata.xml", "dev@example.com")
	status := taskStatus(t, ctx, h, taskID)
	assert.Equal(t, status.Status, gen.ImportTaskStatusNeu)

	status = taskStatus(t, ctx, h, taskID)
	assert.Equal(t, status.Status, gen.ImportTaskStatusInAnalyse)

	_, ok := status.AnalysisProgressInPercent.Get()
	assert.Assert(t, !ok, "expected analysis progress to be unset")

	status = taskStatus(t, ctx, h, taskID)
	assert.Equal(t, status.Status, gen.ImportTaskStatusAnalysiert)

	result, ok := status.AnalysisResult.Get()
	assert.Assert(t, ok, "expected analysis result")
	assert.Equal(t, result, gen.AnalysisResultAlleNeu)

	runID := createRun(t, ctx, h, taskID, "METS.xml", "")
	runStatus := importRunStatus(t, ctx, h, taskID, runID)
	_, ok = runStatus.ProgressPercent.Get()
	assert.Assert(t, !ok, "expected import run progress to be unset")
	assert.Equal(t, statusValue(t, runStatus.Status), "Running")

	status = taskStatus(t, ctx, h, taskID)
	assert.Equal(t, status.Status, gen.ImportTaskStatusWirdImportiert)

	runStatus = importRunStatus(t, ctx, h, taskID, runID)
	_, ok = runStatus.ProgressPercent.Get()
	assert.Assert(t, !ok, "expected import run progress to be unset")
	assert.Equal(t, statusValue(t, runStatus.Status), "Running")

	status = taskStatus(t, ctx, h, taskID)
	assert.Equal(t, status.Status, gen.ImportTaskStatusImportiert)

	importResult, ok := status.ImportResult.Get()
	assert.Assert(t, ok, "expected import result")
	assert.Equal(t, importResult, gen.ImportResultErfolgreich)

	runStatus = importRunStatus(t, ctx, h, taskID, runID)
	_, ok = runStatus.ProgressPercent.Get()
	assert.Assert(t, !ok, "expected import run progress to be unset")
	assert.Equal(t, statusValue(t, runStatus.Status), "Completed")
}

func TestConflictTaskCanBeCancelled(t *testing.T) {
	ctx := t.Context()
	h := mock.NewHandler()

	taskID := createTask(t, ctx, h, "metadata-mock-konflikte.xml", "dev@example.com")
	_ = taskStatus(t, ctx, h, taskID)
	_ = taskStatus(t, ctx, h, taskID)
	status := taskStatus(t, ctx, h, taskID)
	assert.Equal(t, status.Status, gen.ImportTaskStatusAnalysiert)

	result, ok := status.AnalysisResult.Get()
	assert.Assert(t, ok, "expected analysis result")
	assert.Equal(t, result, gen.AnalysisResultKonflikte)

	patchRes, err := h.APIImportTasksIDPatch(
		ctx,
		&gen.CancelImportTaskRequest{Status: gen.ImportTaskStatusAbgebrochen},
		gen.APIImportTasksIDPatchParams{ID: taskID},
	)
	assert.NilError(t, err)

	updated, ok := patchRes.(*gen.ImportTaskDto)
	assert.Assert(t, ok, "expected ImportTaskDto, got %T", patchRes)
	assert.Equal(t, statusValue(t, updated.Status), string(gen.ImportTaskStatusAbgebrochen))
	assert.Equal(t, statusValue(t, updated.AnalysisResult), string(gen.AnalysisResultKonflikte))

	status = taskStatus(t, ctx, h, taskID)
	assert.Equal(t, status.Status, gen.ImportTaskStatusAbgebrochen)

	result, ok = status.AnalysisResult.Get()
	assert.Assert(t, ok, "expected canceled task analysis result")
	assert.Equal(t, result, gen.AnalysisResultKonflikte)

	runRes, err := h.APIImportTasksIDImportRunsPost(
		ctx,
		gen.NewOptAPIImportTasksIDImportRunsPostReq(gen.APIImportTasksIDImportRunsPostReq{
			File: ogenhttp.MultipartFile{Name: "METS.xml"},
		}),
		gen.APIImportTasksIDImportRunsPostParams{ID: taskID},
	)
	assert.NilError(t, err)

	_, ok = runRes.(*gen.ValidationProblemDetails)
	assert.Assert(t, ok, "expected validation problem after cancel, got %T", runRes)
}

func TestImportFailureIsSurfacedThroughTaskStatus(t *testing.T) {
	ctx := t.Context()
	h := mock.NewHandler()

	taskID := createTask(t, ctx, h, "metadata.xml", "dev@example.com")
	_ = taskStatus(t, ctx, h, taskID)
	_ = taskStatus(t, ctx, h, taskID)
	_ = taskStatus(t, ctx, h, taskID)
	runID := createRun(t, ctx, h, taskID, "METS-mock-import-fehler.xml", "")
	_ = taskStatus(t, ctx, h, taskID)
	_ = taskStatus(t, ctx, h, taskID)
	status := taskStatus(t, ctx, h, taskID)
	assert.Equal(t, status.Status, gen.ImportTaskStatusImportiert)

	result, ok := status.ImportResult.Get()
	assert.Assert(t, ok, "expected import result")
	assert.Equal(t, result, gen.ImportResultFehler)

	runStatus := importRunStatus(t, ctx, h, taskID, runID)
	assert.Equal(t, statusValue(t, runStatus.Status), "Failed")

	errMsg, ok := runStatus.Error.Get()
	assert.Assert(t, ok && errMsg != "", "expected import run failure message")
}

func TestPatchUnknownTaskReturnsNotFound(t *testing.T) {
	ctx := t.Context()
	h := mock.NewHandler()

	res, err := h.APIImportTasksIDPatch(
		ctx,
		&gen.CancelImportTaskRequest{Status: gen.ImportTaskStatusAbgebrochen},
		gen.APIImportTasksIDPatchParams{ID: "missing"},
	)
	assert.NilError(t, err)

	_, ok := res.(*gen.APIImportTasksIDPatchNotFound)
	assert.Assert(t, ok, "expected not found response, got %T", res)
}

func TestStatusUnknownTaskReturnsError(t *testing.T) {
	ctx := t.Context()
	h := mock.NewHandler()

	_, err := h.APIImportTasksIDStatusGet(ctx, gen.APIImportTasksIDStatusGetParams{ID: "missing"})
	assert.Error(t, err, "import task does not exist: missing")
}

func TestSecurityHandlerTokenValidation(t *testing.T) {
	ctx := t.Context()
	sec := mock.NewSecurity("expected")

	_, err := sec.HandleSmart(ctx, gen.APIHealthzGetOperation, gen.Smart{Token: "expected"})
	assert.NilError(t, err)

	_, err = sec.HandleSmart(ctx, gen.APIHealthzGetOperation, gen.Smart{Token: "wrong"})
	assert.Error(t, err, "invalid bearer token")
}

func createTask(t *testing.T, ctx context.Context, h *mock.Handler, filename, username string) string {
	t.Helper()

	res, err := h.APIImportTasksPost(ctx, gen.NewOptAPIImportTasksPostReq(gen.APIImportTasksPostReq{
		File:     ogenhttp.MultipartFile{Name: filename},
		SipType:  gen.SipTypeBornDigitalSIP,
		Username: username,
	}))
	assert.NilError(t, err)

	created, ok := res.(*gen.APIImportTasksPostCreated)
	assert.Assert(t, ok, "expected created response, got %T", res)

	taskID, ok := created.ID.Get()
	assert.Assert(t, ok && taskID != "", "expected task ID in create response")

	return taskID
}

func createRun(
	t *testing.T,
	ctx context.Context,
	h *mock.Handler,
	taskID string,
	filename string,
	behaviour gen.ImportBehaviourType,
) string {
	t.Helper()

	req := gen.APIImportTasksIDImportRunsPostReq{
		File: ogenhttp.MultipartFile{Name: filename},
	}
	if behaviour != "" {
		req.ImportBehaviour = gen.NewOptImportBehaviourType(behaviour)
	}

	res, err := h.APIImportTasksIDImportRunsPost(
		ctx,
		gen.NewOptAPIImportTasksIDImportRunsPostReq(req),
		gen.APIImportTasksIDImportRunsPostParams{ID: taskID},
	)
	assert.NilError(t, err)

	created, ok := res.(*gen.CreateImportRunResponse)
	assert.Assert(t, ok, "expected create import run response, got %T", res)

	runID, ok := created.ImportRunId.Get()
	assert.Assert(t, ok && runID != "", "expected import run ID")

	return runID
}

func taskStatus(
	t *testing.T,
	ctx context.Context,
	h *mock.Handler,
	taskID string,
) *gen.ImportTaskStatusResponse {
	t.Helper()

	res, err := h.APIImportTasksIDStatusGet(ctx, gen.APIImportTasksIDStatusGetParams{ID: taskID})
	assert.NilError(t, err)

	return res
}

func importRunStatus(
	t *testing.T,
	ctx context.Context,
	h *mock.Handler,
	taskID, runID string,
) *gen.ImportRunStatusResponse {
	t.Helper()

	res, err := h.APIImportTasksIDImportRunsRunIdStatusGet(ctx, gen.APIImportTasksIDImportRunsRunIdStatusGetParams{
		ID:    taskID,
		RunId: runID,
	})
	assert.NilError(t, err)

	status, ok := res.(*gen.ImportRunStatusResponse)
	assert.Assert(t, ok, "expected import run status response, got %T", res)

	return status
}

func statusValue[T interface {
	Get() (string, bool)
}](t *testing.T, value T) string {
	t.Helper()

	result, ok := value.Get()
	if !ok {
		t.Fatal("expected optional string value to be set")
	}

	return result
}
