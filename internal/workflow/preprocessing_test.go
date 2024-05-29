package workflow_test

import (
	"path/filepath"
	"regexp"
	"testing"

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
		activities.NewSipCreationActivity().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.SipCreationName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewTransformVecteurAIPActivity().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.TransformVecteurAIPName},
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

func (s *PreprocessingTestSuite) TestVecteurSIP() {
	relPath := "fake/path/to/sip"
	finPath := "fake/path/to/sip_bag"
	sipPath := sharedPath + relPath
	bagPath := sharedPath + finPath
	expectedSIP := sip.SIP{
		Type:         enums.SIPTypeVecteurSIP,
		Path:         sipPath,
		ContentPath:  filepath.Join(sipPath, "content"),
		MetadataPath: filepath.Join(sipPath, "header", "metadata.xml"),
		XSDPath:      filepath.Join(sipPath, "header", "xsd", "arelda.xsd"),
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
	expectedSIP := sip.SIP{
		Type:         enums.SIPTypeVecteurAIP,
		Path:         sipPath,
		ContentPath:  filepath.Join(sipPath, "content", "content"),
		MetadataPath: filepath.Join(sipPath, "additional", "UpdatedAreldaMetadata.xml"),
		XSDPath:      filepath.Join(sipPath, "content", "header", "xsd", "arelda.xsd"),
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
		&activities.MetadataValidationParams{
			MetadataPath: filepath.Join(sipPath, "additional", "UpdatedAreldaMetadata.xml"),
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
