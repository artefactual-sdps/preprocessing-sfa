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
	status := getTaskStatus(t, ctx, h, taskID)
	assert.DeepEqual(t, status, &gen.ImportTaskStatusResponse{
		Status: gen.ImportTaskStatusNeu,
	})

	status = getTaskStatus(t, ctx, h, taskID)
	assert.DeepEqual(t, status, &gen.ImportTaskStatusResponse{
		Status: gen.ImportTaskStatusInAnalyse,
	})

	status = getTaskStatus(t, ctx, h, taskID)
	assert.DeepEqual(t, status, &gen.ImportTaskStatusResponse{
		Status:         gen.ImportTaskStatusAnalysiert,
		AnalysisResult: gen.NewOptAnalysisResult(gen.AnalysisResultAlleNeu),
	})

	runID := createRun(t, ctx, h, taskID, "METS.xml", "")
	runStatus := getImportRunStatus(t, ctx, h, taskID, runID)
	assert.DeepEqual(t, runStatus, &gen.ImportRunStatusResponse{
		Status: gen.ImportStatusStarted,
	})

	status = getTaskStatus(t, ctx, h, taskID)
	assert.DeepEqual(t, status, &gen.ImportTaskStatusResponse{
		Status:         gen.ImportTaskStatusWirdImportiert,
		AnalysisResult: gen.NewOptAnalysisResult(gen.AnalysisResultAlleNeu),
	})

	runStatus = getImportRunStatus(t, ctx, h, taskID, runID)
	assert.DeepEqual(t, runStatus, &gen.ImportRunStatusResponse{
		Status: gen.ImportStatusStarted,
	})

	status = getTaskStatus(t, ctx, h, taskID)
	assert.DeepEqual(t, status, &gen.ImportTaskStatusResponse{
		Status:         gen.ImportTaskStatusImportiert,
		AnalysisResult: gen.NewOptAnalysisResult(gen.AnalysisResultAlleNeu),
		ImportResult:   gen.NewOptImportResult(gen.ImportResultErfolgreich),
	})

	runStatus = getImportRunStatus(t, ctx, h, taskID, runID)
	assert.DeepEqual(t, runStatus, &gen.ImportRunStatusResponse{
		Status:       gen.ImportStatusCompleted,
		ImportResult: gen.NewOptImportResult(gen.ImportResultErfolgreich),
	})
}

func TestConflictTaskCanBeCancelled(t *testing.T) {
	ctx := t.Context()
	h := mock.NewHandler()

	taskID := createTask(t, ctx, h, "metadata-mock-konflikte.xml", "dev@example.com")
	_ = getTaskStatus(t, ctx, h, taskID)
	_ = getTaskStatus(t, ctx, h, taskID)
	status := getTaskStatus(t, ctx, h, taskID)
	assert.DeepEqual(t, status, &gen.ImportTaskStatusResponse{
		Status:         gen.ImportTaskStatusAnalysiert,
		AnalysisResult: gen.NewOptAnalysisResult(gen.AnalysisResultKonflikte),
	})

	cancelRes, err := h.APIImporttasksIDCancelPost(
		ctx,
		&gen.CancelImportTaskRequest{},
		gen.APIImporttasksIDCancelPostParams{ID: taskID},
	)
	assert.NilError(t, err)
	assert.DeepEqual(t, cancelRes, &gen.APIImporttasksIDCancelPostNoContent{})

	status = getTaskStatus(t, ctx, h, taskID)
	assert.DeepEqual(t, status, &gen.ImportTaskStatusResponse{
		Status:         gen.ImportTaskStatusAbgebrochen,
		AnalysisResult: gen.NewOptAnalysisResult(gen.AnalysisResultKonflikte),
	})

	runRes, err := h.APIImporttasksIDImportrunsPost(
		ctx,
		gen.NewOptAPIImporttasksIDImportrunsPostReq(gen.APIImporttasksIDImportrunsPostReq{
			File: ogenhttp.MultipartFile{Name: "METS.xml"},
		}),
		gen.APIImporttasksIDImportrunsPostParams{ID: taskID},
	)
	assert.NilError(t, err)
	assert.DeepEqual(t, runRes, &gen.APIImporttasksIDImportrunsPostBadRequest{
		Title:  gen.NewOptNilString("Bad Request"),
		Status: gen.NewOptNilInt32(400),
		Detail: gen.NewOptNilString("cannot create import run for canceled task"),
	})
}

func TestImportFailureIsSurfacedThroughTaskStatus(t *testing.T) {
	ctx := t.Context()
	h := mock.NewHandler()

	taskID := createTask(t, ctx, h, "metadata.xml", "dev@example.com")
	_ = getTaskStatus(t, ctx, h, taskID)
	_ = getTaskStatus(t, ctx, h, taskID)
	_ = getTaskStatus(t, ctx, h, taskID)
	runID := createRun(t, ctx, h, taskID, "METS-mock-import-fehler.xml", "")
	_ = getTaskStatus(t, ctx, h, taskID)
	_ = getTaskStatus(t, ctx, h, taskID)
	status := getTaskStatus(t, ctx, h, taskID)
	assert.DeepEqual(t, status, &gen.ImportTaskStatusResponse{
		Status:         gen.ImportTaskStatusImportiert,
		AnalysisResult: gen.NewOptAnalysisResult(gen.AnalysisResultAlleNeu),
		ImportResult:   gen.NewOptImportResult(gen.ImportResultFehler),
	})

	runStatus := getImportRunStatus(t, ctx, h, taskID, runID)
	assert.DeepEqual(t, runStatus, &gen.ImportRunStatusResponse{
		Status:       gen.ImportStatusFailed,
		ImportResult: gen.NewOptImportResult(gen.ImportResultFehler),
	})
}

func TestPatchUnknownTaskReturnsNotFound(t *testing.T) {
	ctx := t.Context()
	h := mock.NewHandler()

	res, err := h.APIImporttasksIDCancelPost(
		ctx,
		&gen.CancelImportTaskRequest{},
		gen.APIImporttasksIDCancelPostParams{ID: "missing"},
	)
	assert.NilError(t, err)
	assert.DeepEqual(t, res, &gen.APIImporttasksIDCancelPostNotFound{
		Title:  gen.NewOptNilString("Not Found"),
		Status: gen.NewOptNilInt32(404),
		Detail: gen.NewOptNilString("import task does not exist"),
	})
}

func TestStatusUnknownTaskReturnsNotFound(t *testing.T) {
	ctx := t.Context()
	h := mock.NewHandler()

	res, err := h.APIImporttasksIDStatusGet(ctx, gen.APIImporttasksIDStatusGetParams{ID: "missing"})
	assert.NilError(t, err)
	assert.DeepEqual(t, res, &gen.APIImporttasksIDStatusGetNotFound{
		Title:  gen.NewOptNilString("Not Found"),
		Status: gen.NewOptNilInt32(404),
		Detail: gen.NewOptNilString("import task does not exist"),
	})
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

	res, err := h.APIImporttasksPost(ctx, gen.NewOptAPIImporttasksPostReq(gen.APIImporttasksPostReq{
		File:     ogenhttp.MultipartFile{Name: filename},
		SipType:  gen.SipTypeBornDigitalSIP,
		Username: username,
	}))
	assert.NilError(t, err)
	assert.DeepEqual(t, res, &gen.CreateImportTaskResponse{
		ImportTaskId: "task-000001",
	})

	return res.(*gen.CreateImportTaskResponse).ImportTaskId
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

	req := gen.APIImporttasksIDImportrunsPostReq{
		File: ogenhttp.MultipartFile{Name: filename},
	}
	if behaviour != "" {
		req.ImportBehaviour = gen.NewOptImportBehaviourType(behaviour)
	}

	res, err := h.APIImporttasksIDImportrunsPost(
		ctx,
		gen.NewOptAPIImporttasksIDImportrunsPostReq(req),
		gen.APIImporttasksIDImportrunsPostParams{ID: taskID},
	)
	assert.NilError(t, err)
	assert.DeepEqual(t, res, &gen.CreateImportRunResponse{
		ImportRunId: "run-000001",
	})

	return res.(*gen.CreateImportRunResponse).ImportRunId
}

func getTaskStatus(
	t *testing.T,
	ctx context.Context,
	h *mock.Handler,
	taskID string,
) *gen.ImportTaskStatusResponse {
	t.Helper()

	res, err := h.APIImporttasksIDStatusGet(ctx, gen.APIImporttasksIDStatusGetParams{ID: taskID})
	assert.NilError(t, err)

	status, ok := res.(*gen.ImportTaskStatusResponse)
	assert.Assert(t, ok, "expected import task status response, got %T", res)

	return status
}

func getImportRunStatus(
	t *testing.T,
	ctx context.Context,
	h *mock.Handler,
	taskID, runID string,
) *gen.ImportRunStatusResponse {
	t.Helper()

	res, err := h.APIImporttasksIDImportrunsRunIdStatusGet(ctx, gen.APIImporttasksIDImportrunsRunIdStatusGetParams{
		ID:    taskID,
		RunId: runID,
	})
	assert.NilError(t, err)

	status, ok := res.(*gen.ImportRunStatusResponse)
	assert.Assert(t, ok, "expected import run status response, got %T", res)

	return status
}
