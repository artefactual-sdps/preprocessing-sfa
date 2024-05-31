package workflow_test

import (
	"path/filepath"
	"testing"

	"github.com/artefactual-sdps/temporal-activities/bagit"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_worker "go.temporal.io/sdk/worker"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/config"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
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

	s.workflow = workflow.NewPreprocessingWorkflow(sharedPath)
}

func (s *PreprocessingTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func TestPreprocessingWorkflow(t *testing.T) {
	suite.Run(t, new(PreprocessingTestSuite))
}

func (s *PreprocessingTestSuite) TestPreprocessingWorkflowSuccess() {
	relPath := "fake/path/to/sip"
	sipPath := sharedPath + relPath
	expectedSIP := sip.SIP{
		Type:          enums.SIPTypeVecteurAIP,
		Path:          sipPath,
		ContentPath:   filepath.Join(sipPath, "content", "content"),
		MetadataPath:  filepath.Join(sipPath, "additional", "UpdatedAreldaMetadata.xml"),
		XSDPath:       filepath.Join(sipPath, "content", "header", "xsd", "arelda.xsd"),
		TopLevelPaths: []string{filepath.Join(sipPath, "content"), filepath.Join(sipPath, "additional")},
	}
	s.SetupTest(config.Configuration{})

	// Mock activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
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
		activities.ValidateFileFormatsName,
		sessionCtx,
		&activities.ValidateFileFormatsParams{ContentPath: expectedSIP.ContentPath},
	).Return(
		&activities.ValidateFileFormatsResult{}, nil,
	)
	s.env.OnActivity(
		activities.ValidateMetadataName,
		sessionCtx,
		&activities.ValidateMetadataParams{MetadataPath: expectedSIP.MetadataPath},
	).Return(
		&activities.ValidateMetadataResult{}, nil,
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
		&result,
		&workflow.PreprocessingWorkflowResult{RelativePath: relPath},
	)
}
