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
	psvc persistence.Service,
) *PreprocessingWorkflow {
	return &PreprocessingWorkflow{
		sharedPath:      sharedPath,
		checkDuplicates: checkDuplicates,
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
	msg ...string,
) {
	r.Outcome = OutcomeContentError
	ev.Complete(
		ctx,
		enums.EventOutcomeValidationFailure,
		fmt.Sprintf("Content error: %s", strings.Join(msg, "\n\n")),
	)
}

func (r *PreprocessingWorkflowResult) systemError(
	ctx temporalsdk_workflow.Context,
	err error,
	ev *eventWrapper,
	msg ...string,
) {
	logger := temporalsdk_workflow.GetLogger(ctx)
	logger.Error("System error", "message", err.Error())

	// Complete last preservation task event.
	r.Outcome = OutcomeSystemError
	ev.Complete(
		ctx,
		enums.EventOutcomeSystemFailure,
		fmt.Sprintf("System error: %s", strings.Join(msg, "\n\n")),
	)
}

func (r *PreprocessingWorkflowResult) SetRelativePath(base, path string) error {
	rp, err := filepath.Rel(base, path)
	if err != nil {
		return err
	}

	r.RelativePath = rp

	return nil
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
		e = temporal.NewNonRetryableError(
			fmt.Errorf("error calling workflow with unexpected inputs"),
		)
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
			result.systemError(
				ctx,
				e,
				ev,
				"checksum calculation has failed.",
				"Enduro could not generate a checksum for the submitted SIP. Please try again, or ask a system administrator to investigate.",
			)
			return result, nil
		}
		ev.Succeed(
			ctx,
			fmt.Sprintf("SIP checksum calculated using %s", checksumSIP.Algo),
		)

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
			result.systemError(
				ctx,
				e,
				ev,
				"checking for a duplicate SIP has failed.",
				"An error occurred when checking whether SIP is a duplicate. Please try again, or ask a system administrator to investigate.",
			)
			return result, nil
		}
		if checkDuplicate.IsDuplicate {
			result.validationError(
				ctx,
				ev,
				"SIP is a duplicate.",
				"A previously submitted SIP has the same checksum. Please ensure that your package has not already been ingested.",
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
		result.systemError(
			ctx,
			e,
			ev,
			"SIP extraction has failed.",
			fmt.Sprintf(`%q could not be successfully extracted.`, filepath.Base(localPath)),
		)
		return result, nil
	}

	localPath = archiveExtract.ExtractPath
	if e := result.SetRelativePath(w.sharedPath, localPath); e != nil {
		result.systemError(
			ctx,
			e,
			ev,
			"SIP extraction has failed.",
			fmt.Sprintf(`%s could not be successfully extracted.`, filepath.Base(localPath)),
		)
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
			result.systemError(
				ctx,
				e,
				ev,
				"Bag validation has failed.",
				"An error occurred during the bag validation process. Please try again, or ask a system administrator to investigate.",
			)
			return result, nil
		}
		if bagValidateResult.Error != "" {
			result.validationError(
				ctx,
				ev,
				"Bag validation has failed.",
				// TODO: Add BagIt tool and version info.
				// "An attempt to validate the bag using [tool] - version [version] has failed:",
				bagValidateResult.Error,
				"Please ensure the bag is well-formed before reattempting ingest.",
				"Your SIP has been moved to the failed-sips directory.",
			)
		} else {
			// TODO: Add BagIt tool and version info.
			ev.Succeed(ctx, "Bag successfully validated")
		}

		ev = result.newEvent(ctx, "Unbag SIP")
		var unbagResult activities.UnbagResult
		e = temporalsdk_workflow.ExecuteActivity(
			withFilesysActOpts(ctx),
			activities.UnbagName,
			&activities.UnbagParams{Path: localPath},
		).Get(ctx, &unbagResult)
		if e != nil {
			result.systemError(
				ctx,
				e,
				ev,
				"SIP unbagging has failed.",
				"An error occurred during the SIP unbagging process. Please try again, or ask a system administrator to investigate.",
			)
			return result, nil
		}

		localPath = unbagResult.Path
		if e := result.SetRelativePath(w.sharedPath, localPath); e != nil {
			result.systemError(
				ctx,
				e,
				ev,
				"SIP unbagging has failed.",
				"An error occurred during the SIP unbagging process. Please try again, or ask a system administrator to investigate.",
			)
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
		result.validationError(
			ctx,
			ev,
			"SIP identification has failed.",
			"Enduro could not identify the package type. Please ensure that your SIP matches one of the supported package structures.",
		)
		return result, nil
	}

	sip := identifySIP.SIP
	ev.Succeed(ctx, fmt.Sprintf("SIP structure identified: %s", sip.Type))

	// Validate structure.
	ev = result.newEvent(ctx, "Validate SIP structure")
	var validateStructure activities.ValidateStructureResult
	e = temporalsdk_workflow.ExecuteActivity(
		withFilesysActOpts(ctx),
		activities.ValidateStructureName,
		&activities.ValidateStructureParams{SIP: sip},
	).Get(ctx, &validateStructure)
	if e != nil {
		result.systemError(
			ctx,
			e,
			ev,
			"SIP structure validation has failed.",
			"An error occurred during the structure validation process. Please try again, or ask a system administrator to investigate.",
		)
		return result, nil
	}
	if validateStructure.Failures != nil {
		result.validationError(
			ctx,
			ev,
			"SIP structure validation has failed.",
			ul(validateStructure.Failures),
			fmt.Sprintf("Please review the SIP and ensure that its structure matches the %s specifications.", sip.Type),
		)
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
		result.systemError(
			ctx,
			e,
			ev,
			"SIP name validation has failed.",
			"An error occurred during the SIP name validation process. Please try again, or ask a system administrator to investigate.",
		)
		return result, nil
	}
	if ValidateSIPName.Failures != nil {
		result.validationError(
			ctx,
			ev,
			"SIP name validation has failed.",
			fmt.Sprintf(
				"The name used for the package does not match the expected convention for the %q type.",
				sip.Type,
			),
			ul(ValidateSIPName.Failures),
			"Please review the naming conventions specified for this type of SIP.",
		)
	} else {
		ev.Succeed(ctx, "SIP name matches expected naming convention for the identified structure type")
	}

	// Verify that package contents match the manifest.
	manifestEv := result.newEvent(ctx, "Verify SIP manifest")
	checksumEv := result.newEvent(ctx, "Verify SIP checksums")
	var verifyManifest activities.VerifyManifestResult
	e = temporalsdk_workflow.ExecuteActivity(
		withFilesysActOpts(ctx),
		activities.VerifyManifestName,
		&activities.VerifyManifestParams{SIP: sip},
	).Get(ctx, &verifyManifest)
	if e != nil {
		result.systemError(
			ctx,
			e,
			manifestEv,
			"SIP manifest verification has failed.",
			"An error occurred during the manifest verification process. Please try again, or ask a system administrator to investigate.",
		)
		result.systemError(
			ctx,
			e,
			checksumEv,
			"SIP checksum verification has failed.",
			"An error occurred during the checksum verification process. Please try again, or ask a system administrator to investigate.",
		)
		return result, nil
	}

	if len(verifyManifest.MissingFiles) > 0 || len(verifyManifest.UnexpectedFiles) > 0 {
		failures := slices.Concat(verifyManifest.MissingFiles, verifyManifest.UnexpectedFiles)
		result.validationError(
			ctx,
			manifestEv,
			fmt.Sprintf("SIP contents do not match the %q manifest.", filepath.Base(sip.ManifestPath)),
			ul(failures),
			"Please review the SIP and ensure that its contents match those listed in the metadata manifest.",
		)
	} else {
		manifestEv.Succeed(ctx, "SIP contents match manifest")
	}

	if len(verifyManifest.ChecksumFailures) > 0 {
		result.validationError(
			ctx,
			checksumEv,
			"SIP checksums do not match file contents.",
			ul(verifyManifest.ChecksumFailures),
			"Please review the SIP and ensure that the metadata checksums match those of the files.",
		)
	} else {
		checksumEv.Succeed(ctx, "SIP checksums match file contents")
	}

	// Check for disallowed file formats (SIP types only).
	if sip.IsSIP() {
		ev = result.newEvent(ctx, "Check for disallowed file formats")
		var ffvalidateResult ffvalidate.Result
		e = temporalsdk_workflow.ExecuteActivity(
			withFilesysActOpts(ctx),
			ffvalidate.Name,
			&ffvalidate.Params{Path: sip.ContentPath},
		).Get(ctx, &ffvalidateResult)
		if e != nil {
			result.systemError(
				ctx,
				e,
				ev,
				"file format check has failed.",
				"An error occurred when checking for disallowed file formats. Please try again, or ask a system administrator to investigate.",
			)
			return result, nil
		}

		if ffvalidateResult.Failures != nil {
			result.validationError(
				ctx,
				ev,
				"file format check has failed.",
				"One or more file formats are not allowed:",
				ul(ffvalidateResult.Failures),
				"Please review the SIP and remove or replace all disallowed file formats.",
			)
		} else {
			ev.Succeed(ctx, "No disallowed file formats found")
		}
	}

	// Validate SIP file formats against the format specifications.
	ev = result.newEvent(ctx, "Validate SIP file formats")
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
		&activities.ValidateFilesParams{SIP: sip},
	).Get(ctx, &validateFilesResult)
	if e != nil {
		result.systemError(
			ctx,
			e,
			ev,
			"file format validation has failed.",
			"An error occurred during the file format validation process. Please try again, or ask a system administrator to investigate.",
		)
		return result, nil
	}

	if validateFilesResult.Failures != nil {
		result.validationError(
			ctx,
			ev,
			"file format validation has failed.",
			// TODO: Add tool name and version info.
			ul(validateFilesResult.Failures),
			"Please ensure all files are well-formed.",
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
			XMLPath: sip.ManifestPath,
			XSDPath: sip.XSDPath,
		},
	).Get(ctx, &validateMetadata)
	if e != nil {
		result.systemError(
			ctx,
			e,
			ev,
			"metadata validation has failed.",
			fmt.Sprintf(
				"An error has occurred while attempting to validate the %q file. Please try again, or ask a system administrator to investigate.",
				filepath.Base(sip.ManifestPath),
			),
		)
		return result, nil
	}

	if validateMetadata.Failures != nil {
		for idx, f := range validateMetadata.Failures {
			validateMetadata.Failures[idx] = strings.ReplaceAll(f, sip.Path+"/", "")
		}
		result.validationError(
			ctx,
			ev,
			"metadata validation has failed.",
			ul(validateMetadata.Failures),
			"Please ensure all metadata files are present and well-formed.",
		)
	} else {
		ev.Succeed(ctx,
			"Metadata validation successful on the following file(s):",
			ul([]string{filepath.Base(sip.ManifestPath)}),
		)
	}

	// Validate logical metadata (AIP types only).
	if sip.IsAIP() {
		ev = result.newEvent(ctx, "Validate logical metadata")
		var validateLMD activities.ValidatePREMISResult
		e = temporalsdk_workflow.ExecuteActivity(
			withFilesysActOpts(ctx),
			activities.ValidatePREMISName,
			activities.ValidatePREMISParams{Path: sip.LogicalMDPath},
		).Get(ctx, &validateLMD)
		if e != nil {
			result.systemError(
				ctx,
				e,
				ev,
				"logical metadata validation has failed.",
				fmt.Sprintf(
					"An error has occurred while attempting to validate the %q file. Please try again, or ask a system administrator to investigate.",
					filepath.Base(sip.LogicalMDPath),
				),
			)
			return result, nil
		}
		if validateLMD.Failures != nil {
			result.validationError(
				ctx,
				ev,
				"logical metadata validation has failed.",
				ul(validateLMD.Failures),
				"Please ensure all metadata files are present and well-formed.",
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
	if e = writePREMISFile(ctx, identifySIP.SIP); e != nil {
		result.systemError(
			ctx,
			e,
			ev,
			"premis.xml creation has failed",
			"An error has occurred while attempting to create the premis.xml file and store it in the metadata directory. Please try again, or ask a system administrator to investigate.",
		)
	} else {
		ev.Succeed(ctx, "Created a premis.xml file and stored it in the metadata directory")
	}

	// Re-structure SIP.
	ev = result.newEvent(ctx, "Restructure SIP")
	var transformSIP activities.TransformSIPResult
	e = temporalsdk_workflow.ExecuteActivity(
		withFilesysActOpts(ctx),
		activities.TransformSIPName,
		&activities.TransformSIPParams{SIP: sip},
	).Get(ctx, &transformSIP)
	if e != nil {
		result.systemError(
			ctx,
			e,
			ev,
			"restructuring has failed",
			"An error has occurred while attempting to restructure the SIP for preservation processing. Please try again, or ask a system administrator to investigate.",
		)
		return result, nil
	}
	ev.Succeed(ctx, "SIP has been restructured for preservation processing")

	// Write the identifiers.json file.
	ev = result.newEvent(ctx, "Create identifier.json")
	var writeIDFile activities.WriteIdentifierFileResult
	e = temporalsdk_workflow.ExecuteActivity(
		withFilesysActOpts(ctx),
		activities.WriteIdentifierFileName,
		&activities.WriteIdentifierFileParams{PIP: transformSIP.PIP},
	).Get(ctx, &writeIDFile)
	if e != nil {
		result.systemError(
			ctx,
			e,
			ev,
			"identifier.json creation has failed.",
			"An error has occurred while attempting to create the identifier.json file and store it in the metadata directory. Please try again, or ask a system administrator to investigate.",
		)
		return result, nil
	}
	ev.Succeed(ctx, "Created an identifier.json file and stored it in the metadata directory")

	// Bag the SIP for Enduro processing.
	ev = result.newEvent(ctx, "Bag SIP")
	var createBag bagcreate.Result
	e = temporalsdk_workflow.ExecuteActivity(
		withFilesysActOpts(ctx),
		bagcreate.Name,
		&bagcreate.Params{SourcePath: localPath},
	).Get(ctx, &createBag)
	if e != nil {
		result.systemError(
			ctx,
			e,
			ev,
			"SIP bagging has failed.",
			"An error has occurred while attempting to bag the SIP. Please try again, or ask a system administrator to investigate.",
		)
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

func writePREMISFile(ctx temporalsdk_workflow.Context, sip sip.SIP) error {
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
		// Add PREMIS events for the disallowed file format check (SIP types
		// only).
		e = temporalsdk_workflow.ExecuteActivity(
			withFilesysActOpts(ctx),
			activities.AddPREMISEventName,
			&activities.AddPREMISEventParams{
				PREMISFilePath: path,
				Agent:          premis.AgentDefault(),
				Type:           "validation",
				Detail:         "name=\"Check for disallowed file formats\"",
				OutcomeDetail:  "Format allowed",
				Failures:       nil,
			},
		).Get(ctx, &addPREMISEvent)
		if e != nil {
			return e
		}
	}

	// Add PREMIS events for file format validation.
	e = temporalsdk_workflow.ExecuteActivity(
		withFilesysActOpts(ctx),
		activities.AddPREMISValidationEventName,
		&activities.AddPREMISValidationEventParams{
			SIP:            sip,
			PREMISFilePath: path,
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

	// Add PREMIS events for metadata validation.
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

	// Add Enduro PREMIS agent.
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

	return nil
}

// ul formats a list of strings as an unordered, Markdown-style list.
func ul(items []string) string {
	if len(items) == 0 {
		return ""
	}

	var s string
	for _, i := range items {
		s += fmt.Sprintf("- %s\n", i)
	}

	return strings.TrimSuffix(s, "\n")
}
