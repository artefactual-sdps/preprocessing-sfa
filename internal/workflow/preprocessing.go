package workflow

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/artefactual-sdps/temporal-activities/removefiles"
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

	if params == nil || params.RelativePath == "" {
		e = temporal.NewNonRetryableError(fmt.Errorf("error calling workflow with unexpected inputs"))
		return nil, e
	}

	var removePaths []string

	defer func() {
		var result activities.RemovePathsResult
		err := temporalsdk_workflow.ExecuteActivity(withLocalActOpts(ctx), activities.RemovePathsName, &activities.RemovePathsParams{
			Paths: removePaths,
		}).
			Get(ctx, &result)
		e = errors.Join(e, err)

		logger.Debug("PreprocessingWorkflow workflow finished!", "result", r, "error", e)
	}()

	localPath := filepath.Join(w.sharedPath, filepath.Clean(params.RelativePath))

	// Validate SIP structure.
	var checkStructureRes activities.CheckSipStructureResult
	e = temporalsdk_workflow.ExecuteActivity(withLocalActOpts(ctx), activities.CheckSipStructureName, &activities.CheckSipStructureParams{
		SipPath: localPath,
	}).
		Get(ctx, &checkStructureRes)
	if e != nil {
		return nil, e
	}

	// Check allowed file formats.
	var allowedFileFormats activities.AllowedFileFormatsResult
	e = temporalsdk_workflow.ExecuteActivity(withLocalActOpts(ctx), activities.AllowedFileFormatsName, &activities.AllowedFileFormatsParams{
		SipPath: localPath,
	}).
		Get(ctx, &allowedFileFormats)
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
	path := filepath.Join(localPath, "header", "metadata.xml")
	if checkStructureRes.SIP == nil {
		path = filepath.Join(localPath, "additional", "UpdatedAreldaMetadata.xml")
	}
	e = temporalsdk_workflow.ExecuteActivity(withLocalActOpts(ctx), activities.MetadataValidationName, &activities.MetadataValidationParams{
		MetadataPath: path,
	}).
		Get(ctx, &metadataValidation)
	if e != nil {
		return nil, e
	}

	var bagPath string
	if checkStructureRes.SIP != nil {
		// Repackage SFA SIP into a Bag.
		var sipCreation activities.SipCreationResult
		e = temporalsdk_workflow.ExecuteActivity(withLocalActOpts(ctx), activities.SipCreationName, &activities.SipCreationParams{
			SipPath: localPath,
		}).
			Get(ctx, &sipCreation)
		if e != nil {
			return nil, e
		}

		bagPath = sipCreation.NewSipPath

		// Remove initial SIP.
		removePaths = append(removePaths, localPath)
	} else {
		// Re-structure Vecteur AIP into SIP.
		var transformVecteurAIP activities.TransformVecteurAIPResult
		e = temporalsdk_workflow.ExecuteActivity(
			withLocalActOpts(ctx),
			activities.TransformVecteurAIPName,
			&activities.TransformVecteurAIPParams{Path: localPath},
		).Get(ctx, &transformVecteurAIP)
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

		// Remove PREMIS XML files.
		var removeFilesResult removefiles.ActivityResult
		e = temporalsdk_workflow.ExecuteActivity(
			withLocalActOpts(ctx),
			removefiles.ActivityName,
			&removefiles.ActivityParams{Path: localPath},
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

		bagPath = localPath
	}

	// Get final relative path.
	realPath, e := filepath.Rel(w.sharedPath, bagPath)
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
