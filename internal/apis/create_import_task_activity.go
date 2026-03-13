package apis

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	ogenhttp "github.com/ogen-go/ogen/http"
	"go.artefactual.dev/tools/temporal"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/apis/gen"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

const CreateImportTaskActivityName = "create-apis-import-task"

type (
	CreateImportTaskActivity struct {
		client Client
	}

	CreateImportTaskParams struct {
		SIP      sip.SIP
		Username string
	}

	CreateImportTaskResult struct {
		TaskID string
	}
)

func NewCreateImportTaskActivity(client Client) *CreateImportTaskActivity {
	return &CreateImportTaskActivity{client: client}
}

// Execute uploads the SIP metadata.xml file to APIS and returns the created
// import-task identifier for later polling and post-storage work.
func (a *CreateImportTaskActivity) Execute(
	ctx context.Context,
	params *CreateImportTaskParams,
) (*CreateImportTaskResult, error) {
	metadata, err := os.Open(params.SIP.MetadataPath)
	if err != nil {
		return nil, temporal.NewNonRetryableError(fmt.Errorf("open metadata.xml: %v", err))
	}
	defer metadata.Close()

	// SIP types should match between our internal representation and APIS, but
	// validating here keeps bad inputs from being retried pointlessly.
	sipType := gen.SipType(params.SIP.Type.String())
	if err := sipType.Validate(); err != nil {
		return nil, temporal.NewNonRetryableError(fmt.Errorf("invalid SIP type %q: %v", params.SIP.Type, err))
	}

	req := gen.APIImportTasksPostReq{
		File: ogenhttp.MultipartFile{
			Name: filepath.Base(params.SIP.MetadataPath),
			File: metadata,
		},
		SipType:  sipType,
		Username: params.Username,
	}
	res, err := a.client.APIImportTasksPost(ctx, gen.NewOptAPIImportTasksPostReq(req))
	if err != nil {
		return nil, fmt.Errorf("create APIS import task: %v", err)
	}

	switch t := res.(type) {
	case *gen.APIImportTasksPostCreated:
		taskID, ok := t.ID.Get()
		if !ok || taskID == "" {
			return nil, temporal.NewNonRetryableError(fmt.Errorf(
				"create APIS import task: missing task ID in created response",
			))
		}
		return &CreateImportTaskResult{TaskID: taskID}, nil
	case *gen.APIImportTasksPostBadRequest:
		return nil, temporal.NewNonRetryableError(fmt.Errorf(
			"create APIS import task: bad request: %s",
			createImportTaskResponseError((*gen.CreateImportTaskResponse)(t)),
		))
	case *gen.APIImportTasksPostUnsupportedMediaType:
		return nil, temporal.NewNonRetryableError(fmt.Errorf(
			"create APIS import task: unsupported media type: %s",
			createImportTaskResponseError((*gen.CreateImportTaskResponse)(t)),
		))
	case *gen.APIImportTasksPostInternalServerError:
		return nil, fmt.Errorf(
			"create APIS import task: server error: %s",
			createImportTaskResponseError((*gen.CreateImportTaskResponse)(t)),
		)
	case *gen.ProblemDetails:
		return nil, temporal.NewNonRetryableError(fmt.Errorf("create APIS import task: unauthorized"))
	default:
		return nil, temporal.NewNonRetryableError(fmt.Errorf("create APIS import task: unexpected response"))
	}
}

func createImportTaskResponseError(res *gen.CreateImportTaskResponse) string {
	if errMsg, ok := res.Error.Get(); ok && errMsg != "" {
		return errMsg
	}
	return "no additional details"
}
