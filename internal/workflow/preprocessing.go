package workflow

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	"github.com/artefactual-sdps/temporal-activities/removefiles"
	"go.artefactual.dev/tools/temporal"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
)

var premisRe *regexp.Regexp = regexp.MustCompile("(?i)_PREMIS.xml$")

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

	// Validate SIP structure.
	var checkSIPStructureRes activities.CheckSIPStructureResult
	checkSIPStructureErr := temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.CheckSIPStructureName,
		&activities.CheckSIPStructureParams{SIP: identifySIP.SIP},
	).Get(ctx, &checkSIPStructureRes)

	// Validate file formats.
	var allowedFileFormats activities.AllowedFileFormatsResult
	allowedFileFormatsErr := temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.AllowedFileFormatsName,
		&activities.AllowedFileFormatsParams{ContentPath: identifySIP.SIP.ContentPath},
	).Get(ctx, &allowedFileFormats)

	// Validate metadata.
	var metadataValidation activities.MetadataValidationResult
	metadataValidationErr := temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.MetadataValidationName,
		&activities.MetadataValidationParams{MetadataPath: identifySIP.SIP.MetadataPath},
	).Get(ctx, &metadataValidation)

	// Combine and return validation errors.
	e = errors.Join(checkSIPStructureErr, allowedFileFormatsErr, metadataValidationErr)
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

	// Combine PREMIS files into one.
	var combinePREMIS activities.CombinePREMISResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.CombinePREMISName,
		&activities.CombinePREMISParams{Path: localPath},
	).Get(ctx, &combinePREMIS)
	if e != nil {
		return nil, e
	}

	// Remove PREMIS files.
	var removeFilesResult removefiles.ActivityResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		removefiles.ActivityName,
		&removefiles.ActivityParams{
			Path:           localPath,
			RemovePatterns: []*regexp.Regexp{premisRe},
		},
	).Get(ctx, &removeFilesResult)
	if e != nil {
		return nil, e
	}

	// Create Bag.
	var createBag activities.CreateBagResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.CreateBagName,
		&activities.CreateBagParams{Path: localPath},
	).Get(ctx, &createBag)
	if e != nil {
		return nil, e
	}

	// TODO: validate checksums located in the XML metadata file
	// against the checksums generated on Bag creation.

	logger.Debug("PreprocessingWorkflow workflow finished!", "result", r, "error", e)

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
