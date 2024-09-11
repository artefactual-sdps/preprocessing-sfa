package workflow

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/artefactual-sdps/temporal-activities/bagcreate"
	"go.artefactual.dev/tools/temporal"
	temporalsdk_log "go.temporal.io/sdk/log"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/eventlog"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/premis"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
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

	// Verify that package contents match the manifest.
	verifyManifestEvent := newEvent(ctx, "Verify SIP manifest")
	verifyChecksumsEvent := newEvent(ctx, "Verify SIP checksums")
	var verifyManifest activities.VerifyManifestResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.VerifyManifestName,
		&activities.VerifyManifestParams{SIP: identifySIP.SIP},
	).Get(ctx, &verifyManifest)
	if e != nil {
		result.addEvent(verifyManifestEvent.Complete(
			ctx,
			enums.EventOutcomeSystemFailure,
			"System error: manifest verification has failed",
		))
		return systemError(logger, "Verify manifest", &result, e), nil
	}

	if len(verifyManifest.MissingFiles) > 0 || len(verifyManifest.UnexpectedFiles) > 0 {
		failures := slices.Concat(verifyManifest.MissingFiles, verifyManifest.UnexpectedFiles)
		verifyManifestEvent.Complete(
			ctx,
			enums.EventOutcomeValidationFailure,
			"Content error: SIP contents do not match %q:\n%s",
			filepath.Base(identifySIP.SIP.ManifestPath),
			strings.Join(failures, "\n"),
		)
	} else {
		verifyManifestEvent.Complete(
			ctx,
			enums.EventOutcomeSuccess,
			"SIP contents match manifest",
		)
	}
	result.addEvent(verifyManifestEvent)

	if len(verifyManifest.ChecksumFailures) > 0 {
		verifyChecksumsEvent.Complete(
			ctx,
			enums.EventOutcomeValidationFailure,
			"Content error: SIP checksums do not match file contents:\n%s",
			strings.Join(verifyManifest.ChecksumFailures, "\n"),
		)
	} else {
		verifyChecksumsEvent.Complete(
			ctx,
			enums.EventOutcomeSuccess,
			"SIP checksums match file contents",
		)
	}
	result.addEvent(verifyChecksumsEvent)

	// Validate file formats.
	validateFileFormatsEvent := newEvent(ctx, "Validate SIP file formats")
	var validateFileFormats activities.ValidateFileFormatsResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.ValidateFileFormatsName,
		&activities.ValidateFileFormatsParams{SIP: identifySIP.SIP},
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
		&activities.ValidateMetadataParams{SIP: identifySIP.SIP},
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
	for _, t := range result.PreservationTasks {
		if !t.IsSuccess() {
			return contentError(&result), nil
		}
	}

	if e = writePREMISFile(ctx, identifySIP.SIP); e != nil {
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

	// Write the identifiers.json file.
	writeIDFileEvent := newEvent(ctx, "Create identifier.json")
	var writeIDFile activities.WriteIdentifierFileResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.WriteIdentifierFileName,
		&activities.WriteIdentifierFileParams{PIP: transformSIP.PIP},
	).Get(ctx, &writeIDFile)
	if e != nil {
		result.addEvent(writeIDFileEvent.Complete(
			ctx,
			enums.EventOutcomeSystemFailure,
			"System error: creating identifier.json has failed",
		))
		return systemError(logger, "Write identifier file", &result, e), nil
	}
	result.addEvent(writeIDFileEvent.Complete(
		ctx,
		enums.EventOutcomeSuccess,
		"Created an identifier.json file",
	))

	// Bag the SIP for Enduro processing.
	createBagEvent := newEvent(ctx, "Bag SIP")
	var createBag bagcreate.Result
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		bagcreate.Name,
		&bagcreate.Params{SourcePath: localPath},
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

func writePREMISFile(ctx temporalsdk_workflow.Context, sip sip.SIP) error {
	var e error
	path := filepath.Join(sip.Path, "metadata", "premis.xml")

	// Add PREMIS objects.
	var addPREMISObjects activities.AddPREMISObjectsResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.AddPREMISObjectsName,
		&activities.AddPREMISObjectsParams{
			SIP:            sip,
			PREMISFilePath: path,
		},
	).Get(ctx, &addPREMISObjects)
	if e != nil {
		return e
	}

	// Add PREMIS event noting validate structure result.
	validateStructureOutcomeDetail := fmt.Sprintf(
		"SIP structure identified: %s. SIP structure matches validation criteria.",
		sip.Type.String(),
	)

	var addPREMISEvent activities.AddPREMISEventResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.AddPREMISEventName,
		&activities.AddPREMISEventParams{
			PREMISFilePath: path,
			Agent:          premis.AgentDefault(),
			Type:           "validation",
			Detail:         "name=\"Validate SIP structure\"",
			OutcomeDetail:  validateStructureOutcomeDetail,
			Failures:       nil,
		},
	).Get(ctx, &addPREMISEvent)
	if e != nil {
		return e
	}

	// Add PREMIS events for validate file format activity.
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.AddPREMISEventName,
		&activities.AddPREMISEventParams{
			PREMISFilePath: path,
			Agent:          premis.AgentDefault(),
			Type:           "validation",
			Detail:         "name=\"Validate file format\"",
			OutcomeDetail:  "Format allowed",
			Failures:       nil,
		},
	).Get(ctx, &addPREMISEvent)
	if e != nil {
		return e
	}

	// Add PREMIS event noting validate metadata result.
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.AddPREMISEventName,
		&activities.AddPREMISEventParams{
			PREMISFilePath: path,
			Agent:          premis.AgentDefault(),
			Type:           "validation",
			Detail:         "name=\"Validate SIP metadata\"",
			OutcomeDetail:  "Metadata validation successful",
			Failures:       nil,
		},
	).Get(ctx, &addPREMISEvent)
	if e != nil {
		return e
	}

	// Add Enduro PREMIS agent.
	var addPREMISAgent activities.AddPREMISAgentResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.AddPREMISAgentName,
		&activities.AddPREMISAgentParams{
			PREMISFilePath: path,
			Agent:          premis.AgentDefault(),
		},
	).Get(ctx, &addPREMISAgent)
	if e != nil {
		return e
	}

	return nil
}
