package workflow

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/artefactual-sdps/temporal-activities/bagit"
	"go.artefactual.dev/tools/temporal"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
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

func (w *PreprocessingWorkflow) Execute(
	ctx temporalsdk_workflow.Context,
	params *PreprocessingWorkflowParams,
) (r *PreprocessingWorkflowResult, e error) {
	logger := temporalsdk_workflow.GetLogger(ctx)
	logger.Debug("PreprocessingWorkflow workflow running!", "params", params)

	defer func() {
		logger.Debug("PreprocessingWorkflow workflow finished!", "result", r, "error", e)
	}()

	if params == nil || params.RelativePath == "" {
		e = temporal.NewNonRetryableError(fmt.Errorf("error calling workflow with unexpected inputs"))
		return nil, e
	}

	localPath := filepath.Join(w.sharedPath, filepath.Clean(params.RelativePath))

	// Identify SIP.
	var identifySIP activities.IdentifySIPResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.IdentifySIPName,
		&activities.IdentifySIPParams{Path: localPath},
	).Get(ctx, &identifySIP)
	if e != nil {
		return nil, e
	}

	// Validate structure.
	var validateStructure activities.ValidateStructureResult
	validateStructureErr := temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.ValidateStructureName,
		&activities.ValidateStructureParams{SIP: identifySIP.SIP},
	).Get(ctx, &validateStructure)

	// Validate file formats.
	var validateFileFormats activities.ValidateFileFormatsResult
	validateFileFormatsErr := temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.ValidateFileFormatsName,
		&activities.ValidateFileFormatsParams{ContentPath: identifySIP.SIP.ContentPath},
	).Get(ctx, &validateFileFormats)

	// Validate metadata.
	var validateMetadata activities.ValidateMetadataResult
	validateMetadataErr := temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.ValidateMetadataName,
		&activities.ValidateMetadataParams{MetadataPath: identifySIP.SIP.MetadataPath},
	).Get(ctx, &validateMetadata)

	// Combine and return validation errors.
	e = errors.Join(validateStructureErr, validateFileFormatsErr, validateMetadataErr)
	if e != nil {
		return nil, e
	}

	// Re-structure SIP.
	var transformSIP activities.TransformSIPResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.TransformSIPName,
		&activities.TransformSIPParams{SIP: identifySIP.SIP},
	).Get(ctx, &transformSIP)
	if e != nil {
		return nil, e
	}

	// Bag the SIP for Enduro processing.
	var createBag bagit.CreateBagActivityResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		bagit.CreateBagActivityName,
		&bagit.CreateBagActivityParams{SourcePath: localPath},
	).Get(ctx, &createBag)
	if e != nil {
		return nil, e
	}

	// TODO: validate checksums located in the XML metadata file
	// against the checksums generated on Bag creation.

	return &PreprocessingWorkflowResult{RelativePath: params.RelativePath}, e
}

func withLocalActOpts(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})
}
