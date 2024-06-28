package workflow_test

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/artefactual-sdps/temporal-activities/bagit"
	cp "github.com/otiai10/copy"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_worker "go.temporal.io/sdk/worker"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/config"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/eventlog"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/premis"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/workflow"
)

const relPath = "sip"

var testTime = time.Date(2024, 6, 6, 15, 8, 39, 0, time.UTC)

type PreprocessingTestSuite struct {
	suite.Suite
	temporalsdk_testsuite.WorkflowTestSuite

	env      *temporalsdk_testsuite.TestWorkflowEnvironment
	workflow *workflow.PreprocessingWorkflow
	testDir  string
}

func (s *PreprocessingTestSuite) SetupTest(cfg config.Configuration) {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.SetStartTime(testTime)
	s.env.SetWorkerOptions(temporalsdk_worker.Options{EnableSessionWorker: true})
	s.testDir = s.T().TempDir()
	sipPath := filepath.Join(s.testDir, relPath)

	if err := cp.Copy("./testdata/little-Test-AIP-Digitization", sipPath); err != nil {
		s.Failf("couldn't copy test data: %s", err.Error())
	}
	cfg.SharedPath = s.testDir

	// Register activities.
	s.env.RegisterActivityWithOptions(
		activities.NewIdentifySIP().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.IdentifySIPName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewValidateStructure().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.ValidateStructureName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewValidateFileFormats().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.ValidateFileFormatsName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewAddPREMISObjects().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISObjectsName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewAddPREMISEvent().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISEventName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewAddPREMISAgent().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISAgentName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewValidateMetadata().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.ValidateMetadataName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewTransformSIP().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.TransformSIPName},
	)
	s.env.RegisterActivityWithOptions(
		bagit.NewCreateBagActivity(cfg.Bagit).Execute,
		temporalsdk_activity.RegisterOptions{Name: bagit.CreateBagActivityName},
	)

	s.workflow = workflow.NewPreprocessingWorkflow(s.testDir)
}

func (s *PreprocessingTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func TestPreprocessingWorkflow(t *testing.T) {
	suite.Run(t, new(PreprocessingTestSuite))
}

func vecteurAIP(path string) sip.SIP {
	return sip.SIP{
		Type:         enums.SIPTypeVecteurAIP,
		Path:         path,
		ContentPath:  filepath.Join(path, "content", "content"),
		MetadataPath: filepath.Join(path, "additional", "UpdatedAreldaMetadata.xml"),
		XSDPath:      filepath.Join(path, "content", "header", "xsd", "arelda.xsd"),
		TopLevelPaths: []string{
			filepath.Join(path, "content"),
			filepath.Join(path, "additional"),
		},
	}
}

func (s *PreprocessingTestSuite) TestPreprocessingWorkflowSuccess() {
	s.SetupTest(config.Configuration{})
	sipPath := filepath.Join(s.testDir, relPath)
	expectedSIP := vecteurAIP(sipPath)

	// Mock activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	premisFilePath := filepath.Join(expectedSIP.Path, "metadata", "premis.xml")

	s.env.OnActivity(
		activities.IdentifySIPName,
		sessionCtx,
		&activities.IdentifySIPParams{Path: sipPath},
	).Return(
		&activities.IdentifySIPResult{SIP: expectedSIP}, nil,
	)
	s.env.OnActivity(
		activities.ValidateStructureName,
		sessionCtx,
		&activities.ValidateStructureParams{SIP: expectedSIP},
	).Return(
		&activities.ValidateStructureResult{}, nil,
	)
	s.env.OnActivity(
		activities.AddPREMISObjectsName,
		sessionCtx,
		&activities.AddPREMISObjectsParams{
			PREMISFilePath: premisFilePath,
			ContentPath:    expectedSIP.ContentPath,
		},
	).Return(
		&activities.AddPREMISObjectsResult{}, nil,
	)
	s.env.OnActivity(
		activities.AddPREMISEventName,
		sessionCtx,
		&activities.AddPREMISEventParams{
			PREMISFilePath: premisFilePath,
			Agent:          premis.AgentDefault(),
			Type:           "validateStructure",
			Failures:       nil,
		},
	).Return(
		&activities.AddPREMISEventResult{}, nil,
	)
	s.env.OnActivity(
		activities.ValidateFileFormatsName,
		sessionCtx,
		&activities.ValidateFileFormatsParams{
			ContentPath:    expectedSIP.ContentPath,
			PREMISFilePath: premisFilePath,
			Agent:          premis.AgentDefault(),
		},
	).Return(
		&activities.ValidateFileFormatsResult{}, nil,
	)
	s.env.OnActivity(
		activities.AddPREMISEventName,
		sessionCtx,
		&activities.AddPREMISEventParams{
			PREMISFilePath: premisFilePath,
			Agent:          premis.AgentDefault(),
			Type:           "validateFileFormats",
			Failures:       nil,
		},
	).Return(
		&activities.AddPREMISEventResult{}, nil,
	)
	s.env.OnActivity(
		activities.ValidateMetadataName,
		sessionCtx,
		&activities.ValidateMetadataParams{MetadataPath: expectedSIP.MetadataPath},
	).Return(
		&activities.ValidateMetadataResult{}, nil,
	)
	s.env.OnActivity(
		activities.AddPREMISEventName,
		sessionCtx,
		&activities.AddPREMISEventParams{
			PREMISFilePath: premisFilePath,
			Agent:          premis.AgentDefault(),
			Type:           "validateMetadata",
			Failures:       nil,
		},
	).Return(
		&activities.AddPREMISEventResult{}, nil,
	)
	s.env.OnActivity(
		activities.AddPREMISAgentName,
		sessionCtx,
		&activities.AddPREMISAgentParams{
			PREMISFilePath: premisFilePath,
			Agent:          premis.AgentDefault(),
		},
	).Return(
		&activities.AddPREMISAgentResult{}, nil,
	)
	s.env.OnActivity(
		activities.TransformSIPName,
		sessionCtx,
		&activities.TransformSIPParams{SIP: expectedSIP},
	).Return(
		&activities.TransformSIPResult{}, nil,
	)
	s.env.OnActivity(
		bagit.CreateBagActivityName,
		sessionCtx,
		&bagit.CreateBagActivityParams{SourcePath: sipPath},
	).Return(
		&bagit.CreateBagActivityResult{}, nil,
	)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&workflow.PreprocessingWorkflowParams{RelativePath: relPath},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result workflow.PreprocessingWorkflowResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		&workflow.PreprocessingWorkflowResult{
			Outcome:      workflow.OutcomeSuccess,
			RelativePath: relPath,
			PreservationTasks: []eventlog.Event{
				{
					Name:        "Identify SIP structure",
					Message:     "SIP structure identified: VecteurAIP",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Validate SIP structure",
					Message:     "SIP structure matches validation criteria",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Validate SIP file formats",
					Message:     "No disallowed file formats found",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Validate SIP metadata",
					Message:     "Metadata validation successful",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Restructure SIP",
					Message:     "SIP has been restructured",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Bag SIP",
					Message:     "SIP has been bagged",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
			},
		},
		&result,
	)
}

func (s *PreprocessingTestSuite) TestPreprocessingWorkflowIdentifySIPFails() {
	s.SetupTest(config.Configuration{})
	sipPath := filepath.Join(s.testDir, relPath)

	// Mock activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	s.env.OnActivity(
		activities.IdentifySIPName,
		sessionCtx,
		&activities.IdentifySIPParams{Path: sipPath},
	).Return(
		nil, fmt.Errorf("IdentifySIP: NewSIP: stat : no such file or directory"),
	)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&workflow.PreprocessingWorkflowParams{RelativePath: relPath},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result workflow.PreprocessingWorkflowResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		&workflow.PreprocessingWorkflowResult{
			Outcome:      workflow.OutcomeSystemError,
			RelativePath: relPath,
			PreservationTasks: []eventlog.Event{
				{
					Name:        "Identify SIP structure",
					Message:     "System error: SIP structure identification has failed",
					Outcome:     enums.EventOutcomeSystemFailure,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
			},
		},
		&result,
	)
}

func (s *PreprocessingTestSuite) TestPreprocessingWorkflowValidationFails() {
	s.SetupTest(config.Configuration{})
	sipPath := filepath.Join(s.testDir, relPath)
	expectedSIP := vecteurAIP(sipPath)

	// Mock activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	premisFilePath := filepath.Join(expectedSIP.Path, "metadata", "premis.xml")
	s.env.OnActivity(
		activities.IdentifySIPName,
		sessionCtx,
		&activities.IdentifySIPParams{Path: sipPath},
	).Return(
		&activities.IdentifySIPResult{SIP: expectedSIP}, nil,
	)
	s.env.OnActivity(
		activities.ValidateStructureName,
		sessionCtx,
		&activities.ValidateStructureParams{SIP: expectedSIP},
	).Return(
		&activities.ValidateStructureResult{
			Failures: []string{
				"XSD folder is missing",
				"UpdatedAreldaMetadata.xml is missing",
			},
		},
		nil,
	)
	s.env.OnActivity(
		activities.ValidateFileFormatsName,
		sessionCtx,
		&activities.ValidateFileFormatsParams{
			ContentPath:    expectedSIP.ContentPath,
			PREMISFilePath: premisFilePath,
			Agent:          premis.AgentDefault(),
		},
	).Return(
		&activities.ValidateFileFormatsResult{Failures: []string{
			`file format fmt/11 not allowed: "fake/path/to/sip/dir/file1.png"`,
			`file format fmt/11 not allowed: "fake/path/to/sip/file2.png"`,
		}},
		nil,
	)
	s.env.OnActivity(
		activities.ValidateMetadataName,
		sessionCtx,
		&activities.ValidateMetadataParams{MetadataPath: expectedSIP.MetadataPath},
	).Return(
		&activities.ValidateMetadataResult{Failures: []string{
			`fake/path/to/sip/additional/UpdatedAreldaMetadata.xml does not match expected metadata requirements`,
		}}, nil,
	)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&workflow.PreprocessingWorkflowParams{RelativePath: relPath},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result workflow.PreprocessingWorkflowResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		&workflow.PreprocessingWorkflowResult{
			Outcome:      workflow.OutcomeContentError,
			RelativePath: relPath,
			PreservationTasks: []eventlog.Event{
				{
					Name:        "Identify SIP structure",
					Message:     "SIP structure identified: VecteurAIP",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name: "Validate SIP structure",
					Message: `Content error: SIP structure validation has failed:
XSD folder is missing
UpdatedAreldaMetadata.xml is missing`,
					Outcome:     enums.EventOutcomeValidationFailure,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name: "Validate SIP file formats",
					Message: `Content error: file format validation has failed. One or more file formats are not allowed:
file format fmt/11 not allowed: "fake/path/to/sip/dir/file1.png"
file format fmt/11 not allowed: "fake/path/to/sip/file2.png"`,
					Outcome:     enums.EventOutcomeValidationFailure,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Validate SIP metadata",
					Message:     "Content error: metadata validation has failed: fake/path/to/sip/additional/UpdatedAreldaMetadata.xml does not match expected metadata requirements",
					Outcome:     enums.EventOutcomeValidationFailure,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
			},
		},
		&result,
	)
}
