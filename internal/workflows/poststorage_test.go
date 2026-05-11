package workflows_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/artefactual-sdps/enduro/pkg/childwf"
	"github.com/artefactual-sdps/temporal-activities/removepaths"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_worker "go.temporal.io/sdk/worker"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/amss"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/apis"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/config"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/workflows"
)

type TestSuite struct {
	suite.Suite
	temporalsdk_testsuite.WorkflowTestSuite

	env      *temporalsdk_testsuite.TestWorkflowEnvironment
	workflow *workflows.Poststorage
	testDir  string
}

func (s *TestSuite) setup(cfg *config.PoststorageConfig) {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.SetWorkerOptions(temporalsdk_worker.Options{EnableSessionWorker: true})
	s.testDir = s.T().TempDir()
	cfg.WorkingDir = s.testDir

	s.env.RegisterActivityWithOptions(
		amss.NewGetAIPPathActivity(nil).Execute,
		temporalsdk_activity.RegisterOptions{Name: amss.GetAIPPathActivityName},
	)
	s.env.RegisterActivityWithOptions(
		amss.NewFetchActivity(nil).Execute,
		temporalsdk_activity.RegisterOptions{Name: amss.FetchActivityName},
	)
	s.env.RegisterActivityWithOptions(
		removepaths.New().Execute,
		temporalsdk_activity.RegisterOptions{Name: removepaths.Name},
	)

	s.workflow = workflows.NewPoststorage(*cfg)
}

func TestWorkflow(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) TestWorkflowSuccess() {
	aipUUID := uuid.MustParse("9390594f-84c2-457d-bd6a-618f21f7c954")

	s.setup(&config.PoststorageConfig{})
	s.mockActivitiesSuccess(aipUUID)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&childwf.PostStorageParams{
			AIPUUID: aipUUID.String(),
			CustomMetadata: childwf.CustomMetadata{
				apis.CustomMetadataKey: json.RawMessage(
					`{"importTaskId":"task-000001","decision":"Continue and append"}`,
				),
			},
		},
	)

	s.True(s.env.IsWorkflowCompleted())
	s.env.AssertExpectations(s.T())

	var result childwf.PostStorageResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)

	s.Equal(result, childwf.PostStorageResult{})
}

func (s *TestSuite) TestWorkflowErrorsWhenAIPUUIDIsInvalid() {
	s.setup(&config.PoststorageConfig{})

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&childwf.PostStorageParams{
			AIPUUID: "not-a-uuid",
		},
	)

	s.True(s.env.IsWorkflowCompleted())
	s.ErrorContains(s.env.GetWorkflowError(), "parse AIP UUID")
	s.env.AssertExpectations(s.T())
}

func (s *TestSuite) mockActivitiesSuccess(aipUUID uuid.UUID) {
	aipUUIDString := aipUUID.String()
	aipName := "test-" + aipUUIDString
	searchMDName := fmt.Sprintf("search-md_%s", aipName)
	localDir := filepath.Join(s.testDir, searchMDName)

	// Mock activities.
	s.env.OnActivity(
		amss.GetAIPPathActivityName,
		mock.AnythingOfType("*context.timerCtx"),
		&amss.GetAIPPathActivityParams{AIPUUID: aipUUID},
	).Return(
		&amss.GetAIPPathActivityResult{
			Path: "9390/594f/84c2/457d/bd6a/618f/21f7/c954/test-9390594f-84c2-457d-bd6a-618f21f7c954.zip",
		}, nil,
	)

	// Mock session activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	metsName := fmt.Sprintf("METS.%s.xml", aipUUIDString)
	metsPath := filepath.Join(localDir, metsName)
	s.env.OnActivity(
		amss.FetchActivityName,
		sessionCtx,
		&amss.FetchActivityParams{
			AIPUUID:      aipUUID,
			RelativePath: fmt.Sprintf("%s/data/%s", aipName, metsName),
			Destination:  metsPath,
		},
	).Return(
		&amss.FetchActivityResult{}, nil,
	)
}
