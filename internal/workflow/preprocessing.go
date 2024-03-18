package workflow

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"go.artefactual.dev/tools/temporal"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"
)

type PreprocessingWorkflowParams struct {
	RelativePath string
}

type PreprocessingWorkflowResult struct {
	RelativePath string
}

type PreprocessingWorkflow struct {
	sharedPath string
}

func NewPreprocessingWorkflow(sharedPath string) *PreprocessingWorkflow {
	return &PreprocessingWorkflow{
		sharedPath: sharedPath,
	}
}

func (w *PreprocessingWorkflow) Execute(ctx temporalsdk_workflow.Context, params *PreprocessingWorkflowParams) (r *PreprocessingWorkflowResult, e error) {
	logger := temporalsdk_workflow.GetLogger(ctx)
	logger.Debug("PreprocessingWorkflow workflow running!", "params", params)

	if params == nil || params.RelativePath == "" {
		e = temporal.NewNonRetryableError(fmt.Errorf("error calling workflow with unexpected inputs"))
		return nil, e
	}

	var removePaths []string

	defer func() {
		var result activities.RemovePathsResult
		err := temporalsdk_workflow.ExecuteActivity(withLocalActOpts(ctx), activities.RemovePathsName, &activities.RemovePathsParams{
			Paths: removePaths,
		}).Get(ctx, &result)
		e = errors.Join(e, err)

		logger.Debug("PreprocessingWorkflow workflow finished!", "result", r, "error", e)
	}()

	localPath := filepath.Join(w.sharedPath, filepath.Clean(params.RelativePath))

	// Extract package.
	var extractPackageRes activities.ExtractPackageResult
	e = temporalsdk_workflow.ExecuteActivity(withLocalActOpts(ctx), activities.ExtractPackageName, &activities.ExtractPackageParams{
		Path: localPath,
	}).Get(ctx, &extractPackageRes)
	if e != nil {
		return nil, e
	}

	// Always remove extracted path.
	removePaths = append(removePaths, extractPackageRes.Path)

	// Validate SIP structure.
	var checkStructureRes activities.CheckSipStructureResult
	e = temporalsdk_workflow.ExecuteActivity(withLocalActOpts(ctx), activities.CheckSipStructureName, &activities.CheckSipStructureParams{
		SipPath: extractPackageRes.Path,
	}).Get(ctx, &checkStructureRes)
	if e != nil {
		return nil, e
	}

	// Check allowed file formats.
	var allowedFileFormats activities.AllowedFileFormatsResult
	e = temporalsdk_workflow.ExecuteActivity(withLocalActOpts(ctx), activities.AllowedFileFormatsName, &activities.AllowedFileFormatsParams{
		SipPath: extractPackageRes.Path,
	}).Get(ctx, &allowedFileFormats)
	if e != nil {
		return nil, e
	}

	// Return both errors.
	if !checkStructureRes.Ok {
		e = activities.ErrInvaliSipStructure
	}
	if !allowedFileFormats.Ok {
		e = errors.Join(e, activities.ErrIlegalFileFormat)
	}
	if e != nil {
		return nil, e
	}

	// Validate metadata.xsd.
	var metadataValidation activities.MetadataValidationResult
	e = temporalsdk_workflow.ExecuteActivity(withLocalActOpts(ctx), activities.MetadataValidationName, &activities.MetadataValidationParams{
		SipPath: extractPackageRes.Path,
	}).Get(ctx, &metadataValidation)
	if e != nil {
		return nil, e
	}

	// Repackage SFA SIP into a Bag.
	var sipCreation activities.SipCreationResult
	e = temporalsdk_workflow.ExecuteActivity(withLocalActOpts(ctx), activities.SipCreationName, &activities.SipCreationParams{
		SipPath: extractPackageRes.Path,
	}).Get(ctx, &sipCreation)
	if e != nil {
		return nil, e
	}

	// Get final relative path.
	realPath, e := filepath.Rel(w.sharedPath, sipCreation.NewSipPath)
	if e != nil {
		return nil, e
	}

	return &PreprocessingWorkflowResult{RelativePath: realPath}, e
}

func withLocalActOpts(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})
}
