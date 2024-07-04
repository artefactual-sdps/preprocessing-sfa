package workflow

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/artefactual-sdps/temporal-activities/bagit"
	"go.artefactual.dev/tools/temporal"
	temporalsdk_log "go.temporal.io/sdk/log"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/eventlog"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/premis"
)

type Outcome int

const (
	OutcomeSuccess Outcome = iota
	OutcomeSystemError
	OutcomeContentError
)

type PreprocessingWorkflowParams struct {
	RelativePath string
}

type PreprocessingWorkflowResult struct {
	Outcome           Outcome
	RelativePath      string
	PreservationTasks []eventlog.Event
}

func (r *PreprocessingWorkflowResult) addEvent(e *eventWrapper) {
	if e != nil && e.Event != nil {
		r.PreservationTasks = append(r.PreservationTasks, *e.Event)
	}
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
	var result PreprocessingWorkflowResult

	logger := temporalsdk_workflow.GetLogger(ctx)
	logger.Debug("Preprocessing workflow running!", "params", params)

	defer func() {
		logger.Debug("Preprocessing workflow finished!", "result", r, "error", e)
	}()

	if params == nil || params.RelativePath == "" {
		e = temporal.NewNonRetryableError(fmt.Errorf("error calling workflow with unexpected inputs"))
		return nil, e
	}
	result.RelativePath = params.RelativePath

	localPath := filepath.Join(w.sharedPath, filepath.Clean(params.RelativePath))

	// Identify SIP.
	identifySIPEvent := newEvent(ctx, "Identify SIP structure")
	var identifySIP activities.IdentifySIPResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.IdentifySIPName,
		&activities.IdentifySIPParams{Path: localPath},
	).Get(ctx, &identifySIP)
	if e != nil {
		result.addEvent(identifySIPEvent.Complete(
			ctx,
			enums.EventOutcomeSystemFailure,
			"System error: SIP structure identification has failed",
		))
		return systemError(logger, "Identify SIP", &result, e), nil
	}

	result.addEvent(identifySIPEvent.Complete(
		ctx,
		enums.EventOutcomeSuccess,
		"SIP structure identified: %s",
		identifySIP.SIP.Type.String(),
	))

	// Validate structure.
	validateStructureEvent := newEvent(ctx, "Validate SIP structure")
	var validateStructure activities.ValidateStructureResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.ValidateStructureName,
		&activities.ValidateStructureParams{SIP: identifySIP.SIP},
	).Get(ctx, &validateStructure)
	if e != nil {
		result.addEvent(validateStructureEvent.Complete(
			ctx,
			enums.EventOutcomeSystemFailure,
			"System error: SIP structure validation has failed",
		))
		return systemError(logger, "Validate structure", &result, e), nil
	}

	if validateStructure.Failures != nil {
		validateStructureEvent.Complete(
			ctx,
			enums.EventOutcomeValidationFailure,
			"Content error: SIP structure validation has failed:\n%s",
			strings.Join(validateStructure.Failures, "\n"),
		)
	} else {
		validateStructureEvent.Complete(
			ctx,
			enums.EventOutcomeSuccess,
			"SIP structure matches validation criteria",
		)
	}
	result.addEvent(validateStructureEvent)

	// Add PREMIS objects.
	premisFilePath := filepath.Join(localPath, "metadata", "premis.xml")

	var addPREMISObjects activities.AddPREMISObjectsResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.AddPREMISObjectsName,
		&activities.AddPREMISObjectsParams{
			PREMISFilePath: premisFilePath,
			ContentPath:    identifySIP.SIP.ContentPath,
		},
	).Get(ctx, &addPREMISObjects)
	if e != nil {
		return nil, e
	}

	// Add PREMIS event noting validate structure result.
	validateStructureOutcomeDetail := fmt.Sprintf(
		"SIP structure identified: %s. SIP structure matches validation criteria.",
		identifySIP.SIP.Type.String(),
	)

	var addPREMISEvent activities.AddPREMISEventResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.AddPREMISEventName,
		&activities.AddPREMISEventParams{
			PREMISFilePath: premisFilePath,
			Agent:          premis.AgentDefault(),
			Type:           "validation",
			Detail:         "name=\"Validate SIP structure\"",
			OutcomeDetail:  validateStructureOutcomeDetail,
			Failures:       validateStructure.Failures,
		},
	).Get(ctx, &addPREMISEvent)
	if e != nil {
		return nil, e
	}

	// Validate file formats.
	validateFileFormatsEvent := newEvent(ctx, "Validate SIP file formats")
	var validateFileFormats activities.ValidateFileFormatsResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.ValidateFileFormatsName,
		&activities.ValidateFileFormatsParams{
			ContentPath:    identifySIP.SIP.ContentPath,
			PREMISFilePath: premisFilePath,
			Agent:          premis.AgentDefault()},
	).Get(ctx, &validateFileFormats)
	if e != nil {
		result.addEvent(validateFileFormatsEvent.Complete(
			ctx,
			enums.EventOutcomeSystemFailure,
			"System error: file format validation has failed",
		))
		return systemError(logger, "Validate file formats", &result, e), nil
	}

	if validateFileFormats.Failures != nil {
		validateFileFormatsEvent.Complete(
			ctx,
			enums.EventOutcomeValidationFailure,
			"Content error: file format validation has failed. One or more file formats are not allowed:\n%s",
			strings.Join(validateFileFormats.Failures, "\n"),
		)
	} else {
		validateFileFormatsEvent.Complete(
			ctx,
			enums.EventOutcomeSuccess,
			"No disallowed file formats found",
		)
	}
	result.addEvent(validateFileFormatsEvent)

	// Validate metadata.
	validateMetadataEvent := newEvent(ctx, "Validate SIP metadata")
	var validateMetadata activities.ValidateMetadataResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.ValidateMetadataName,
		&activities.ValidateMetadataParams{MetadataPath: identifySIP.SIP.MetadataPath},
	).Get(ctx, &validateMetadata)
	if e != nil {
		validateMetadataEvent.Complete(
			ctx,
			enums.EventOutcomeSystemFailure,
			"System error: metadata validation has failed",
		)
		result.addEvent(validateMetadataEvent)
		return systemError(logger, "Validate metadata", &result, e), nil
	}

	if validateMetadata.Failures != nil {
		validateMetadataEvent.Complete(
			ctx,
			enums.EventOutcomeValidationFailure,
			"Content error: metadata validation has failed: %s",
			strings.Join(validateMetadata.Failures, "\n"),
		)
	} else {
		validateMetadataEvent.Complete(ctx, enums.EventOutcomeSuccess, "Metadata validation successful")
	}
	result.addEvent(validateMetadataEvent)

	// Stop here if the SIP content isn't valid.
	if !validateStructureEvent.IsSuccess() ||
		!validateFileFormatsEvent.IsSuccess() ||
		!validateMetadataEvent.IsSuccess() {
		return contentError(&result), nil
	}

	// Add PREMIS event noting validate metadata result.
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.AddPREMISEventName,
		&activities.AddPREMISEventParams{
			PREMISFilePath: premisFilePath,
			Agent:          premis.AgentDefault(),
			Type:           "validation",
			Detail:         "name=\"Validate SIP metadata\"",
			OutcomeDetail:  "Metadata validation successful",
			Failures:       validateMetadata.Failures,
		},
	).Get(ctx, &addPREMISEvent)
	if e != nil {
		return nil, e
	}

	// Add PREMIS agent.
	var addPREMISAgent activities.AddPREMISAgentResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.AddPREMISAgentName,
		&activities.AddPREMISAgentParams{
			PREMISFilePath: premisFilePath,
			Agent:          premis.AgentDefault(),
		},
	).Get(ctx, &addPREMISAgent)
	if e != nil {
		return nil, e
	}

	// Re-structure SIP.
	restructureSIPEvent := newEvent(ctx, "Restructure SIP")
	var transformSIP activities.TransformSIPResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.TransformSIPName,
		&activities.TransformSIPParams{SIP: identifySIP.SIP},
	).Get(ctx, &transformSIP)
	if e != nil {
		result.addEvent(restructureSIPEvent.Complete(
			ctx,
			enums.EventOutcomeSystemFailure,
			"System error: restructuring has failed",
		))
		return systemError(logger, "Restructure SIP", &result, e), nil
	}

	result.addEvent(restructureSIPEvent.Complete(
		ctx,
		enums.EventOutcomeSuccess,
		"SIP has been restructured",
	))

	// Bag the SIP for Enduro processing.
	createBagEvent := newEvent(ctx, "Bag SIP")
	var createBag bagit.CreateBagActivityResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		bagit.CreateBagActivityName,
		&bagit.CreateBagActivityParams{SourcePath: localPath},
	).Get(ctx, &createBag)
	if e != nil {
		result.addEvent(createBagEvent.Complete(
			ctx,
			enums.EventOutcomeSystemFailure,
			"System error: bagging has failed",
		))
		return systemError(logger, "Create bag", &result, e), nil
	}

	result.addEvent(createBagEvent.Complete(
		ctx,
		enums.EventOutcomeSuccess,
		"SIP has been bagged",
	))

	// TODO: validate checksums located in the XML metadata file
	// against the checksums generated on Bag creation.

	return &result, nil
}

func withLocalActOpts(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})
}

func contentError(r *PreprocessingWorkflowResult) *PreprocessingWorkflowResult {
	r.Outcome = OutcomeContentError
	return r
}

func systemError(
	logger temporalsdk_log.Logger,
	activityName string,
	r *PreprocessingWorkflowResult,
	err error,
) *PreprocessingWorkflowResult {
	logger.Error(activityName, "system error", err.Error())
	r.Outcome = OutcomeSystemError
	return r
}
