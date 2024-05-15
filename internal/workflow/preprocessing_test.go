package workflow_test

import (
	"path/filepath"
	"testing"

	"github.com/artefactual-sdps/temporal-activities/removefiles"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_worker "go.temporal.io/sdk/worker"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/config"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
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
		activities.NewTransformVecteurAIPActivity().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.TransformVecteurAIPName},
	)
	s.env.RegisterActivityWithOptions(
		removefiles.NewActivity(removefiles.Config{}).Execute,
		temporalsdk_activity.RegisterOptions{Name: removefiles.ActivityName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewCreateBagActivity().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.CreateBagName},
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

func (s *PreprocessingTestSuite) TestSIP() {
	relPath := "fake/path/to/sip"
	finPath := "fake/path/to/sip_bag"
	sipPath := sharedPath + relPath
	bagPath := sharedPath + finPath
	s.SetupTest(config.Configuration{})

	// Mock activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	s.env.OnActivity(
		activities.CheckSipStructureName,
		sessionCtx,
		&activities.CheckSipStructureParams{SipPath: sipPath},
	).Return(
		&activities.CheckSipStructureResult{Ok: true, SIP: &sip.SFASip{}}, nil,
	)
	s.env.OnActivity(
		activities.AllowedFileFormatsName,
		sessionCtx,
		&activities.AllowedFileFormatsParams{SipPath: sipPath},
	).Return(
		&activities.AllowedFileFormatsResult{Ok: true}, nil,
	)
	s.env.OnActivity(
		activities.MetadataValidationName,
		sessionCtx,
		&activities.MetadataValidationParams{MetadataPath: filepath.Join(sipPath, "header", "metadata.xml")},
	).Return(
		&activities.MetadataValidationResult{}, nil,
	)
	s.env.OnActivity(
		activities.SipCreationName,
		sessionCtx,
		&activities.SipCreationParams{SipPath: sipPath},
	).Return(
		&activities.SipCreationResult{NewSipPath: bagPath}, nil,
	)
	s.env.OnActivity(
		activities.RemovePathsName,
		sessionCtx,
		&activities.RemovePathsParams{Paths: []string{sipPath}},
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

func (s *PreprocessingTestSuite) TestVecteurAIP() {
	relPath := "fake/path/to/aip"
	sipPath := sharedPath + relPath
	s.SetupTest(config.Configuration{})

	// Mock activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	s.env.OnActivity(
		activities.CheckSipStructureName,
		sessionCtx,
		&activities.CheckSipStructureParams{SipPath: sipPath},
	).Return(
		&activities.CheckSipStructureResult{Ok: true}, nil,
	)
	s.env.OnActivity(
		activities.AllowedFileFormatsName,
		sessionCtx,
		&activities.AllowedFileFormatsParams{SipPath: sipPath},
	).Return(
		&activities.AllowedFileFormatsResult{Ok: true}, nil,
	)
	s.env.OnActivity(
		activities.MetadataValidationName,
		sessionCtx,
		&activities.MetadataValidationParams{
			MetadataPath: filepath.Join(sipPath, "content", "header", "old", "SIP", "metadata.xml"),
		},
	).Return(
		&activities.MetadataValidationResult{}, nil,
	)
	s.env.OnActivity(
		activities.TransformVecteurAIPName,
		sessionCtx,
		&activities.TransformVecteurAIPParams{Path: sipPath},
	).Return(
		&activities.TransformVecteurAIPResult{}, nil,
	)
	s.env.OnActivity(
		removefiles.ActivityName,
		sessionCtx,
		&removefiles.ActivityParams{Path: sipPath},
	).Return(
		&removefiles.ActivityResult{}, nil,
	)
	s.env.OnActivity(
		activities.CreateBagName,
		sessionCtx,
		&activities.CreateBagParams{Path: sipPath},
	).Return(
		&activities.CreateBagResult{}, nil,
	)
	s.env.OnActivity(
		activities.RemovePathsName,
		sessionCtx,
		&activities.RemovePathsParams{Paths: []string(nil)},
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
		&workflow.PreprocessingWorkflowResult{RelativePath: relPath},
	)
}
