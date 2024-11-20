package workflow

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/artefactual-sdps/temporal-activities/bagcreate"
	"github.com/artefactual-sdps/temporal-activities/ffvalidate"
	"github.com/artefactual-sdps/temporal-activities/xmlvalidate"
	"go.artefactual.dev/tools/temporal"
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

type (
	PreprocessingWorkflow struct {
		sharedPath string
	}

	PreprocessingWorkflowParams struct {
		RelativePath string
	}

	PreprocessingWorkflowResult struct {
		Outcome           Outcome
		RelativePath      string
		PreservationTasks []*eventlog.Event
	}
)

func NewPreprocessingWorkflow(sharedPath string) *PreprocessingWorkflow {
	return &PreprocessingWorkflow{
		sharedPath: sharedPath,
	}
}

func (r *PreprocessingWorkflowResult) newEvent(ctx temporalsdk_workflow.Context, name string) *eventWrapper {
	ev := newWrappedEvent(ctx, name)
	r.PreservationTasks = append(r.PreservationTasks, ev.Event)

	return ev
}

func (r *PreprocessingWorkflowResult) validationError(
	ctx temporalsdk_workflow.Context,
	ev *eventWrapper,
	msg string,
	failures []string,
) {
	r.Outcome = OutcomeContentError
	ev.Complete(
		ctx,
		enums.EventOutcomeValidationFailure,
		"Content error: %s:\n%s",
		msg,
		strings.Join(failures, "\n"),
	)
}

func (r *PreprocessingWorkflowResult) systemError(
	ctx temporalsdk_workflow.Context,
	err error,
	ev *eventWrapper,
	msg string,
) {
	logger := temporalsdk_workflow.GetLogger(ctx)
	logger.Error("System error", "message", err.Error())

	// Complete last preservation task event.
	ev.Complete(ctx, enums.EventOutcomeSystemFailure, "System error: %s", msg)
	r.Outcome = OutcomeSystemError
}

func (w *PreprocessingWorkflow) Execute(
	ctx temporalsdk_workflow.Context,
	params *PreprocessingWorkflowParams,
) (*PreprocessingWorkflowResult, error) {
	var e error
	result := &PreprocessingWorkflowResult{}

	logger := temporalsdk_workflow.GetLogger(ctx)
	logger.Debug("Preprocessing workflow running!", "params", params)

	defer func() {
		logger.Debug("Preprocessing workflow finished!", "result", result, "error", e)
	}()

	if params == nil || params.RelativePath == "" {
		e = temporal.NewNonRetryableError(fmt.Errorf("error calling workflow with unexpected inputs"))
		return nil, e
	}
	result.RelativePath = params.RelativePath

	localPath := filepath.Join(w.sharedPath, filepath.Clean(params.RelativePath))

	// Identify SIP.
	ev := result.newEvent(ctx, "Identify SIP structure")
	var identifySIP activities.IdentifySIPResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.IdentifySIPName,
		&activities.IdentifySIPParams{Path: localPath},
	).Get(ctx, &identifySIP)
	if e != nil {
		result.systemError(ctx, e, ev, "SIP structure identification has failed")
		return result, nil
	}
	ev.Succeed(ctx, "SIP structure identified: %s", identifySIP.SIP.Type.String())

	// Validate structure.
	ev = result.newEvent(ctx, "Validate SIP structure")
	var validateStructure activities.ValidateStructureResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.ValidateStructureName,
		&activities.ValidateStructureParams{SIP: identifySIP.SIP},
	).Get(ctx, &validateStructure)
	if e != nil {
		result.systemError(ctx, e, ev, "SIP structure validation has failed")
		return result, nil
	}
	if validateStructure.Failures != nil {
		result.validationError(ctx, ev, "SIP structure validation has failed", validateStructure.Failures)
	} else {
		ev.Succeed(ctx, "SIP structure matches validation criteria")
	}

	// Verify that package contents match the manifest.
	manifestEv := result.newEvent(ctx, "Verify SIP manifest")
	checksumEv := result.newEvent(ctx, "Verify SIP checksums")
	var verifyManifest activities.VerifyManifestResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.VerifyManifestName,
		&activities.VerifyManifestParams{SIP: identifySIP.SIP},
	).Get(ctx, &verifyManifest)
	if e != nil {
		checksumEv.Complete(ctx, enums.EventOutcomeSystemFailure, "checksum verification has failed")
		result.systemError(ctx, e, manifestEv, "manifest verification has failed")
		return result, nil
	}

	if len(verifyManifest.MissingFiles) > 0 || len(verifyManifest.UnexpectedFiles) > 0 {
		failures := slices.Concat(verifyManifest.MissingFiles, verifyManifest.UnexpectedFiles)
		result.validationError(
			ctx,
			manifestEv,
			fmt.Sprintf("SIP contents do not match %q", filepath.Base(identifySIP.SIP.ManifestPath)),
			failures,
		)
	} else {
		manifestEv.Succeed(ctx, "SIP contents match manifest")
	}

	if len(verifyManifest.ChecksumFailures) > 0 {
		result.validationError(
			ctx,
			checksumEv,
			"SIP checksums do not match file contents",
			verifyManifest.ChecksumFailures,
		)
	} else {
		checksumEv.Succeed(ctx, "SIP checksums match file contents")
	}

	// Validate file formats.
	ev = result.newEvent(ctx, "Validate SIP file formats")
	var ffvalidateResult ffvalidate.Result
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		ffvalidate.Name,
		&ffvalidate.Params{Path: identifySIP.SIP.ContentPath},
	).Get(ctx, &ffvalidateResult)
	if e != nil {
		result.systemError(ctx, e, ev, "System error: file format validation has failed")
		return result, nil
	}

	if ffvalidateResult.Failures != nil {
		result.validationError(
			ctx,
			ev,
			"file format validation has failed. One or more file formats are not allowed", ffvalidateResult.Failures,
		)
	} else {
		ev.Succeed(ctx, "No disallowed file formats found")
	}

	// Validate metadata.
	ev = result.newEvent(ctx, "Validate SIP metadata")
	var validateMetadata xmlvalidate.Result
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		xmlvalidate.Name,
		&xmlvalidate.Params{
			XMLPath: identifySIP.SIP.ManifestPath,
			XSDPath: identifySIP.SIP.XSDPath,
		},
	).Get(ctx, &validateMetadata)
	if e != nil {
		result.systemError(ctx, e, ev, "metadata validation has failed")
		return result, nil
	}

	if validateMetadata.Failures != nil {
		for idx, f := range validateMetadata.Failures {
			validateMetadata.Failures[idx] = strings.ReplaceAll(f, identifySIP.SIP.Path+"/", "")
		}
		result.validationError(ctx, ev, "metadata validation has failed", validateMetadata.Failures)
	} else {
		ev.Succeed(ctx, "Metadata validation successful")
	}

	// Stop here if the SIP content isn't valid.
	if result.Outcome == OutcomeContentError {
		return result, nil
	}

	// Write PREMIS XML.
	ev = result.newEvent(ctx, "Create premis.xml")
	if e = writePREMISFile(ctx, identifySIP.SIP); e != nil {
		result.systemError(ctx, e, ev, "premis.xml creation has failed")
	} else {
		ev.Succeed(ctx, "Created a premis.xml and stored in metadata directory")
	}

	// Re-structure SIP.
	ev = result.newEvent(ctx, "Restructure SIP")
	var transformSIP activities.TransformSIPResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.TransformSIPName,
		&activities.TransformSIPParams{SIP: identifySIP.SIP},
	).Get(ctx, &transformSIP)
	if e != nil {
		result.systemError(ctx, e, ev, "restructuring has failed")
		return result, nil
	}
	ev.Succeed(ctx, "SIP has been restructured")

	// Write the identifiers.json file.
	ev = result.newEvent(ctx, "Create identifier.json")
	var writeIDFile activities.WriteIdentifierFileResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.WriteIdentifierFileName,
		&activities.WriteIdentifierFileParams{PIP: transformSIP.PIP},
	).Get(ctx, &writeIDFile)
	if e != nil {
		result.systemError(ctx, e, ev, "creating identifier.json has failed")
		return result, nil
	}
	ev.Succeed(ctx, "Created an identifier.json and stored in metadata directory")

	// Bag the SIP for Enduro processing.
	ev = result.newEvent(ctx, "Bag SIP")
	var createBag bagcreate.Result
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		bagcreate.Name,
		&bagcreate.Params{SourcePath: localPath},
	).Get(ctx, &createBag)
	if e != nil {
		result.systemError(ctx, e, ev, "bagging has failed")
		return result, nil
	}
	ev.Succeed(ctx, "SIP has been bagged")

	return result, nil
}

func withLocalActOpts(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})
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
