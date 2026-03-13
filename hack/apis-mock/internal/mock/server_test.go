package mock_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
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
		Status:       gen.NewOptString("WirdImportiert"),
		ImportTaskId: gen.NewOptInt32(1),
		ImportRunId:  gen.NewOptString("run-000001"),
	})

	status = getTaskStatus(t, ctx, h, taskID)
	assert.DeepEqual(t, status, &gen.ImportTaskStatusResponse{
		Status:         gen.ImportTaskStatusWirdImportiert,
		AnalysisResult: gen.NewOptAnalysisResult(gen.AnalysisResultAlleNeu),
	})

	runStatus = getImportRunStatus(t, ctx, h, taskID, runID)
	assert.DeepEqual(t, runStatus, &gen.ImportRunStatusResponse{
		Status:       gen.NewOptString("WirdImportiert"),
		ImportTaskId: gen.NewOptInt32(1),
		ImportRunId:  gen.NewOptString("run-000001"),
	})

	status = getTaskStatus(t, ctx, h, taskID)
	assert.DeepEqual(t, status, &gen.ImportTaskStatusResponse{
		Status:         gen.ImportTaskStatusImportiert,
		AnalysisResult: gen.NewOptAnalysisResult(gen.AnalysisResultAlleNeu),
		ImportResult:   gen.NewOptImportResult(gen.ImportResultErfolgreich),
	})

	runStatus = getImportRunStatus(t, ctx, h, taskID, runID)
	assert.DeepEqual(t, runStatus, &gen.ImportRunStatusResponse{
		Status:       gen.NewOptString("Abgeschlossen"),
		ImportTaskId: gen.NewOptInt32(1),
		ImportRunId:  gen.NewOptString("run-000001"),
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

	patchRes, err := h.APIImportTasksIDPatch(
		ctx,
		&gen.CancelImportTaskRequest{Status: gen.ImportTaskStatusAbgebrochen},
		gen.APIImportTasksIDPatchParams{ID: taskID},
	)
	assert.NilError(t, err)
	assert.DeepEqual(t, patchRes, &gen.ImportTaskDto{
		Status:             gen.NewOptString(string(gen.ImportTaskStatusAbgebrochen)),
		AnalysisResult:     gen.NewOptNilString(string(gen.AnalysisResultKonflikte)),
		ImportTaskId:       gen.NewOptInt32(1),
		RowVersion:         []uint8{},
		TaskType:           gen.NewOptString("MockImportTask"),
		Name:               gen.NewOptNilString("metadata-mock-konflikte.xml"),
		NoAutomaticImport:  gen.NewOptBool(false),
		RequiresContainers: gen.NewOptBool(false),
		CreatedBy:          gen.NewOptString("dev@example.com"),
		CreatedOn:          gen.NewOptDateTime(time.Now()),
		AnalysisRecords:    []gen.AnalysisRecordDto{},
		DefaultValues:      []gen.DefaultValueDto{},
		Documents:          []gen.DocumentDto{},
		Imports:            []gen.ImportDto{},
	}, cmpopts.EquateApproxTime(time.Second))

	status = getTaskStatus(t, ctx, h, taskID)
	assert.DeepEqual(t, status, &gen.ImportTaskStatusResponse{
		Status:         gen.ImportTaskStatusAbgebrochen,
		AnalysisResult: gen.NewOptAnalysisResult(gen.AnalysisResultKonflikte),
	})

	runRes, err := h.APIImportTasksIDImportRunsPost(
		ctx,
		gen.NewOptAPIImportTasksIDImportRunsPostReq(gen.APIImportTasksIDImportRunsPostReq{
			File: ogenhttp.MultipartFile{Name: "METS.xml"},
		}),
		gen.APIImportTasksIDImportRunsPostParams{ID: taskID},
	)
	assert.NilError(t, err)
	assert.DeepEqual(t, runRes, &gen.ValidationProblemDetails{
		Title:  gen.NewOptNilString("Bad Request"),
		Status: gen.NewOptNilInt32(400),
		Detail: gen.NewOptNilString("cannot create import run for canceled task"),
		Errors: gen.NewOptValidationProblemDetailsErrors(
			gen.ValidationProblemDetailsErrors{"status": {"task is canceled"}},
		),
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
		Status:       gen.NewOptString("Fehler"),
		ImportTaskId: gen.NewOptInt32(1),
		ImportRunId:  gen.NewOptString("run-000001"),
		Error:        gen.NewOptNilString("mock import failed"),
	})
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
	assert.DeepEqual(t, res, &gen.APIImportTasksIDPatchNotFound{
		Title:  gen.NewOptNilString("Not Found"),
		Status: gen.NewOptNilInt32(404),
		Detail: gen.NewOptNilString("import task does not exist"),
	})
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
	assert.DeepEqual(t, res, &gen.APIImportTasksPostCreated{
		ID:      gen.NewOptNilString("task-000001"),
		Success: gen.NewOptBool(true),
	})

	return res.(*gen.APIImportTasksPostCreated).ID.Value
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
	assert.DeepEqual(t, res, &gen.CreateImportRunResponse{
		ImportTaskId: gen.NewOptString("task-000001"),
		ImportRunId:  gen.NewOptNilString("run-000001"),
		Success:      gen.NewOptBool(true),
	})

	return res.(*gen.CreateImportRunResponse).ImportRunId.Value
}

func getTaskStatus(
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

func getImportRunStatus(
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
