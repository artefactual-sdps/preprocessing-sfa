package workflow_test

import (
	"path/filepath"
	"regexp"
	"testing"

	"github.com/artefactual-sdps/temporal-activities/bagit"
	"github.com/artefactual-sdps/temporal-activities/removefiles"
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

var premisRe *regexp.Regexp = regexp.MustCompile("(?i)_PREMIS.xml$")

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
		activities.NewCheckSIPStructure().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.CheckSIPStructureName},
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
		activities.NewTransformSIP().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.TransformSIPName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewCombinePREMISActivity().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.CombinePREMISName},
	)
	s.env.RegisterActivityWithOptions(
		removefiles.NewActivity().Execute,
		temporalsdk_activity.RegisterOptions{Name: removefiles.ActivityName},
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
		Type:         enums.SIPTypeVecteurAIP,
		Path:         sipPath,
		ContentPath:  filepath.Join(sipPath, "content", "content"),
		MetadataPath: filepath.Join(sipPath, "additional", "UpdatedAreldaMetadata.xml"),
		XSDPath:      filepath.Join(sipPath, "content", "header", "xsd", "arelda.xsd"),
		RemovePaths:  []string{filepath.Join(sipPath, "content"), filepath.Join(sipPath, "additional")},
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
		activities.CheckSIPStructureName,
		sessionCtx,
		&activities.CheckSIPStructureParams{SIP: expectedSIP},
	).Return(
		&activities.CheckSIPStructureResult{}, nil,
	)
	s.env.OnActivity(
		activities.AllowedFileFormatsName,
		sessionCtx,
		&activities.AllowedFileFormatsParams{ContentPath: expectedSIP.ContentPath},
	).Return(
		&activities.AllowedFileFormatsResult{}, nil,
	)
	s.env.OnActivity(
		activities.MetadataValidationName,
		sessionCtx,
		&activities.MetadataValidationParams{MetadataPath: expectedSIP.MetadataPath},
	).Return(
		&activities.MetadataValidationResult{}, nil,
	)
	s.env.OnActivity(
		activities.TransformSIPName,
		sessionCtx,
		&activities.TransformSIPParams{SIP: expectedSIP},
	).Return(
		&activities.TransformSIPResult{}, nil,
	)
	s.env.OnActivity(
		activities.CombinePREMISName,
		sessionCtx,
		&activities.CombinePREMISParams{Path: sipPath},
	).Return(
		&activities.CombinePREMISResult{}, nil,
	)
	s.env.OnActivity(
		removefiles.ActivityName,
		sessionCtx,
		&removefiles.ActivityParams{
			Path:           sipPath,
			RemovePatterns: []*regexp.Regexp{premisRe},
		},
	).Return(
		&removefiles.ActivityResult{}, nil,
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
