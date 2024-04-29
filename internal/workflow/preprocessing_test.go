package workflow_test

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_worker "go.temporal.io/sdk/worker"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/config"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/workflow"
)

const sharedPath = "/shared/path/"

type PreprocessingTestSuite struct {
	suite.Suite
	temporalsdk_testsuite.WorkflowTestSuite

	env      *temporalsdk_testsuite.TestWorkflowEnvironment
	workflow *workflow.PreprocessingWorkflow
}

func (s *PreprocessingTestSuite) SetupTest(cfg config.Configuration) {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.SetWorkerOptions(temporalsdk_worker.Options{EnableSessionWorker: true})

	// Register activities.
	s.env.RegisterActivityWithOptions(
		activities.NewExtractPackage().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.ExtractPackageName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewCheckSipStructure().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.CheckSipStructureName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewAllowedFileFormatsActivity().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.AllowedFileFormatsName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewMetadataValidationActivity().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.MetadataValidationName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewSipCreationActivity().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.SipCreationName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewRemovePaths().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.RemovePathsName},
	)

	s.workflow = workflow.NewPreprocessingWorkflow(sharedPath)
}

func (s *PreprocessingTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func TestPreprocessingWorkflow(t *testing.T) {
	suite.Run(t, new(PreprocessingTestSuite))
}

func (s *PreprocessingTestSuite) TestExecute() {
	relPath := "fake/path/to/sip.zip"
	finPath := "fake/path/to/sip_bag"
	iniPath := sharedPath + relPath
	extPath := sharedPath + "fake/path/to/sip"
	bagPath := sharedPath + finPath
	s.SetupTest(config.Configuration{})

	// Mock activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	s.env.OnActivity(
		activities.ExtractPackageName,
		sessionCtx,
		&activities.ExtractPackageParams{Path: iniPath},
	).Return(
		&activities.ExtractPackageResult{Path: extPath}, nil,
	)
	s.env.OnActivity(
		activities.CheckSipStructureName,
		sessionCtx,
		&activities.CheckSipStructureParams{SipPath: extPath},
	).Return(
		&activities.CheckSipStructureResult{Ok: true}, nil,
	)
	s.env.OnActivity(
		activities.AllowedFileFormatsName,
		sessionCtx,
		&activities.AllowedFileFormatsParams{SipPath: extPath},
	).Return(
		&activities.AllowedFileFormatsResult{Ok: true}, nil,
	)
	s.env.OnActivity(
		activities.MetadataValidationName,
		sessionCtx,
		&activities.MetadataValidationParams{SipPath: extPath},
	).Return(
		&activities.MetadataValidationResult{}, nil,
	)
	s.env.OnActivity(
		activities.SipCreationName,
		sessionCtx,
		&activities.SipCreationParams{SipPath: extPath},
	).Return(
		&activities.SipCreationResult{NewSipPath: bagPath}, nil,
	)
	s.env.OnActivity(
		activities.RemovePathsName,
		sessionCtx,
		&activities.RemovePathsParams{Paths: []string{extPath}},
	).Return(
		&activities.RemovePathsResult{}, nil,
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
		&result,
		&workflow.PreprocessingWorkflowResult{RelativePath: finPath},
	)
}
