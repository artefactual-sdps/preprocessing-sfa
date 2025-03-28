package workflow

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/artefactual-sdps/temporal-activities/archiveextract"
	"github.com/artefactual-sdps/temporal-activities/bagcreate"
	"github.com/artefactual-sdps/temporal-activities/bagvalidate"
	"github.com/artefactual-sdps/temporal-activities/ffvalidate"
	"github.com/artefactual-sdps/temporal-activities/xmlvalidate"
	"go.artefactual.dev/tools/temporal"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/eventlog"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/localact"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/persistence"
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
		sharedPath      string
		checkDuplicates bool
		veraPDFVersion  string
		psvc            persistence.Service
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

func NewPreprocessingWorkflow(
	sharedPath string,
	checkDuplicates bool,
	veraPDFVersion string,
	psvc persistence.Service,
) *PreprocessingWorkflow {
	return &PreprocessingWorkflow{
		sharedPath:      sharedPath,
		checkDuplicates: checkDuplicates,
		veraPDFVersion:  veraPDFVersion,
		psvc:            psvc,
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

	if w.checkDuplicates {
		// Calculate SIP checksum.
		ev := result.newEvent(ctx, "Calculate SIP checksum")
		var checksumSIP activities.ChecksumSIPResult
		e = temporalsdk_workflow.ExecuteActivity(
			withFilesysActOpts(ctx),
			activities.ChecksumSIPName,
			&activities.ChecksumSIPParams{Path: localPath},
		).Get(ctx, &checksumSIP)
		if e != nil {
			result.systemError(ctx, e, ev, "Calculating the SIP checksum has failed")
			return result, nil
		}
		ev.Succeed(ctx, "SIP checksum calculated")

		// Check for duplicate SIP.
		ev = result.newEvent(ctx, "Check for duplicate SIP")
		var checkDuplicate localact.CheckDuplicateResult
		e = temporalsdk_workflow.ExecuteLocalActivity(
			withLocalActOpts(ctx),
			localact.CheckDuplicate,
			w.psvc,
			&localact.CheckDuplicateParams{
				Name:     filepath.Base(localPath),
				Checksum: checksumSIP.Hash,
			},
		).Get(ctx, &checkDuplicate)
		if e != nil {
			result.systemError(ctx, e, ev, "Error attempting to check for duplicate SIP")
			return result, nil
		}
		if checkDuplicate.IsDuplicate {
			result.validationError(
				ctx,
				ev,
				"SIP is a duplicate",
				[]string{"A previously submitted SIP has the same checksum"},
			)
			return result, nil
		}
		ev.Succeed(ctx, "SIP is not a duplicate")
	}

	// Extract SIP.
	ev := result.newEvent(ctx, "Extract SIP")
	var archiveExtract archiveextract.Result
	e = temporalsdk_workflow.ExecuteActivity(
		withFilesysActOpts(ctx),
		archiveextract.Name,
		&archiveextract.Params{SourcePath: localPath},
	).Get(ctx, &archiveExtract)
	if e != nil {
		result.systemError(ctx, e, ev, "Extracting the SIP has failed")
		return result, nil
	}
	localPath = archiveExtract.ExtractPath
	result.RelativePath, e = filepath.Rel(w.sharedPath, localPath)
	if e != nil {
		result.systemError(ctx, e, ev, "Extracting the SIP has failed")
		return result, nil
	}
	ev.Succeed(ctx, "SIP extracted")

	// Check if the SIP is a BagIt bag.
	var isBag localact.IsBagResult
	e = temporalsdk_workflow.ExecuteLocalActivity(
		withLocalActOpts(ctx),
		localact.IsBag,
		&localact.IsBagParams{Path: localPath},
	).Get(ctx, &isBag)
	if e != nil {
		return nil, fmt.Errorf("bag check: %v", e)
	}

	// Unbag the SIP if it is a bag.
	if isBag.IsBag {
		ev := result.newEvent(ctx, "Validate Bag")
		var bagValidateResult bagvalidate.Result
		e = temporalsdk_workflow.ExecuteActivity(
			withFilesysActOpts(ctx),
			bagvalidate.Name,
			&bagvalidate.Params{Path: localPath},
		).Get(ctx, &bagValidateResult)
		if e != nil {
			result.systemError(ctx, e, ev, "Error attempting to validate the Bag")
			return result, nil
		}
		if bagValidateResult.Error != "" {
			result.validationError(ctx, ev, "Bag validation has failed", []string{bagValidateResult.Error})
		} else {
			ev.Succeed(ctx, "Bag validated")
		}

		ev = result.newEvent(ctx, "Unbag SIP")
		var unbagResult activities.UnbagResult
		e = temporalsdk_workflow.ExecuteActivity(
			withFilesysActOpts(ctx),
			activities.UnbagName,
			&activities.UnbagParams{Path: localPath},
		).Get(ctx, &unbagResult)
		if e != nil {
			result.systemError(ctx, e, ev, "Unbagging the SIP has failed")
			return result, nil
		}
		localPath = unbagResult.Path
		result.RelativePath, e = filepath.Rel(w.sharedPath, localPath)
		if e != nil {
			result.systemError(ctx, e, ev, "Unbagging the SIP has failed")
			return result, nil
		}
		ev.Succeed(ctx, "SIP unbagged")
	}

	// Identify SIP.
	ev = result.newEvent(ctx, "Identify SIP structure")
	var identifySIP activities.IdentifySIPResult
	e = temporalsdk_workflow.ExecuteActivity(
		withFilesysActOpts(ctx),
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
		withFilesysActOpts(ctx),
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

	// Validate SIP name.
	ev = result.newEvent(ctx, "Validate SIP name")
	var ValidateSIPName activities.ValidateSIPNameResult
	e = temporalsdk_workflow.ExecuteActivity(
		withFilesysActOpts(ctx),
		activities.ValidateSIPNameName,
		&activities.ValidateSIPNameParams{SIP: identifySIP.SIP},
	).Get(ctx, &ValidateSIPName)
	if e != nil {
		result.systemError(ctx, e, ev, "SIP name validation has failed")
		return result, nil
	}
	if ValidateSIPName.Failures != nil {
		result.validationError(ctx, ev, "SIP name validation has failed", ValidateSIPName.Failures)
	} else {
		ev.Succeed(ctx, "SIP name matches validation criteria")
	}

	// Verify that package contents match the manifest.
	manifestEv := result.newEvent(ctx, "Verify SIP manifest")
	checksumEv := result.newEvent(ctx, "Verify SIP checksums")
	var verifyManifest activities.VerifyManifestResult
	e = temporalsdk_workflow.ExecuteActivity(
		withFilesysActOpts(ctx),
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

	// Verify that SIP file formats are on allowlist (SIP types only).
	if identifySIP.SIP.IsSIP() {
		ev = result.newEvent(ctx, "Validate SIP file formats")
		var ffvalidateResult ffvalidate.Result
		e = temporalsdk_workflow.ExecuteActivity(
			withFilesysActOpts(ctx),
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
				"file format validation has failed. One or more file formats are not allowed",
				ffvalidateResult.Failures,
			)
		} else {
			ev.Succeed(ctx, "No disallowed file formats found")
		}
	}

	// Validate the SIP files.
	ev = result.newEvent(ctx, "Validate SIP files")
	var validateFilesResult activities.ValidateFilesResult
	e = temporalsdk_workflow.ExecuteActivity(
		temporalsdk_workflow.WithActivityOptions(
			ctx,
			temporalsdk_workflow.ActivityOptions{
				ScheduleToCloseTimeout: time.Hour,
				RetryPolicy: &temporalsdk_temporal.RetryPolicy{
					MaximumAttempts: 1,
				},
			},
		),
		activities.ValidateFilesName,
		&activities.ValidateFilesParams{SIP: identifySIP.SIP},
	).Get(ctx, &validateFilesResult)
	if e != nil {
		result.systemError(ctx, e, ev, "System error: file validation has failed")
		return result, nil
	}

	if validateFilesResult.Failures != nil {
		result.validationError(
			ctx,
			ev,
			"file validation has failed. One or more files are invalid", validateFilesResult.Failures,
		)
	} else {
		ev.Succeed(ctx, "No invalid files found")
	}

	// Validate metadata.
	ev = result.newEvent(ctx, "Validate SIP metadata")
	var validateMetadata xmlvalidate.Result
	e = temporalsdk_workflow.ExecuteActivity(
		withFilesysActOpts(ctx),
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

	// Validate logical metadata (AIP types only).
	if identifySIP.SIP.IsAIP() {
		ev = result.newEvent(ctx, "Validate logical metadata")
		var validateLMD activities.ValidatePREMISResult
		e = temporalsdk_workflow.ExecuteActivity(
			withFilesysActOpts(ctx),
			activities.ValidatePREMISName,
			activities.ValidatePREMISParams{Path: identifySIP.SIP.LogicalMDPath},
		).Get(ctx, &validateLMD)
		if e != nil {
			result.systemError(ctx, e, ev, "logical metadata validation has failed")
			return result, nil
		}
		if validateLMD.Failures != nil {
			result.validationError(
				ctx,
				ev,
				"logical metadata validation has failed",
				validateLMD.Failures,
			)
		} else {
			ev.Succeed(ctx, "Logical metadata validation successful")
		}
	}

	// Stop here if the SIP content isn't valid.
	if result.Outcome == OutcomeContentError {
		return result, nil
	}

	// Write PREMIS XML.
	ev = result.newEvent(ctx, "Create premis.xml")

	if e = writePREMISFile(ctx, identifySIP.SIP, w.veraPDFVersion); e != nil {
		result.systemError(ctx, e, ev, "premis.xml creation has failed")
	} else {
		ev.Succeed(ctx, "Created a premis.xml and stored in metadata directory")
	}

	// Re-structure SIP.
	ev = result.newEvent(ctx, "Restructure SIP")
	var transformSIP activities.TransformSIPResult
	e = temporalsdk_workflow.ExecuteActivity(
		withFilesysActOpts(ctx),
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
		withFilesysActOpts(ctx),
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
		withFilesysActOpts(ctx),
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
	return temporalsdk_workflow.WithLocalActivityOptions(
		ctx,
		temporalsdk_workflow.LocalActivityOptions{
			ScheduleToCloseTimeout: 5 * time.Second,
			RetryPolicy: &temporalsdk_temporal.RetryPolicy{
				InitialInterval:    time.Second,
				BackoffCoefficient: 2,
				MaximumInterval:    time.Minute,
				MaximumAttempts:    3,
			},
		},
	)
}

func withFilesysActOpts(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})
}

func writePREMISFile(ctx temporalsdk_workflow.Context, sip sip.SIP, veraPDFVersion string) error {
	var e error
	path := filepath.Join(sip.Path, "metadata", "premis.xml")

	// Add PREMIS objects.
	var addPREMISObjects activities.AddPREMISObjectsResult
	e = temporalsdk_workflow.ExecuteActivity(
		withFilesysActOpts(ctx),
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
		withFilesysActOpts(ctx),
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

	// Add PREMIS event noting validate SIP name result.
	validateSIPNameOutcomeDetail := fmt.Sprintf(
		"SIP name %q matches validation criteria.",
		sip.Name(),
	)

	e = temporalsdk_workflow.ExecuteActivity(
		withFilesysActOpts(ctx),
		activities.AddPREMISEventName,
		&activities.AddPREMISEventParams{
			PREMISFilePath: path,
			Agent:          premis.AgentDefault(),
			Type:           "validation",
			Detail:         "name=\"Validate SIP name\"",
			OutcomeDetail:  validateSIPNameOutcomeDetail,
			Failures:       nil,
		},
	).Get(ctx, &addPREMISEvent)
	if e != nil {
		return e
	}

	if sip.IsSIP() {
		// Add PREMIS events for validate file format activity (SIP types only).
		e = temporalsdk_workflow.ExecuteActivity(
			withFilesysActOpts(ctx),
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
	}

	// Add PREMIS events for validate file activity.
	veraPDFAgent := premis.Agent{
		Type:    "software",
		Name:    veraPDFVersion,
		IdType:  "url",
		IdValue: "https://verapdf.org",
	}

	e = temporalsdk_workflow.ExecuteActivity(
		withFilesysActOpts(ctx),
		activities.AddPREMISValidationEventName,
		&activities.AddPREMISValidationEventParams{
			SIP:            sip,
			PREMISFilePath: path,
			Agent:          veraPDFAgent,
			Summary: premis.EventSummary{
				Type:          "validation",
				Detail:        "name=\"Validate SIP file formats\"",
				Outcome:       "valid",
				OutcomeDetail: "File format complies with specification",
			},
		},
	).Get(ctx, &addPREMISEvent)
	if e != nil {
		return e
	}

	// Add PREMIS event noting validate metadata result.
	e = temporalsdk_workflow.ExecuteActivity(
		withFilesysActOpts(ctx),
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

	// Add Enduro and veraPDF PREMIS agents.
	var addPREMISEnduroAgent activities.AddPREMISAgentResult
	e = temporalsdk_workflow.ExecuteActivity(
		withFilesysActOpts(ctx),
		activities.AddPREMISAgentName,
		&activities.AddPREMISAgentParams{
			PREMISFilePath: path,
			Agent:          premis.AgentDefault(),
		},
	).Get(ctx, &addPREMISEnduroAgent)
	if e != nil {
		return e
	}

	var addPREMISVeraPDFAgent activities.AddPREMISAgentResult
	e = temporalsdk_workflow.ExecuteActivity(
		withFilesysActOpts(ctx),
		activities.AddPREMISAgentName,
		&activities.AddPREMISAgentParams{
			PREMISFilePath: path,
			Agent:          veraPDFAgent,
		},
	).Get(ctx, &addPREMISVeraPDFAgent)
	if e != nil {
		return e
	}

	return nil
}
