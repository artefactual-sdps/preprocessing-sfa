package workercmd

import (
	"context"

	"github.com/go-logr/logr"
	"go.artefactual.dev/tools/temporal"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/config"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/workflow"
)

const Name = "preprocessing-worker"

type Main struct {
	logger         logr.Logger
	cfg            config.Configuration
	temporalWorker worker.Worker
	temporalClient client.Client
}

func NewMain(logger logr.Logger, cfg config.Configuration) *Main {
	return &Main{
		logger: logger,
		cfg:    cfg,
	}
}

func (m *Main) Run(ctx context.Context) error {
	c, err := client.Dial(client.Options{
		HostPort:  m.cfg.Temporal.Address,
		Namespace: m.cfg.Temporal.Namespace,
		Logger:    temporal.Logger(m.logger.WithName("temporal")),
	})
	if err != nil {
		m.logger.Error(err, "Unable to create Temporal client.")
		return err
	}
	m.temporalClient = c

	w := worker.New(m.temporalClient, m.cfg.Temporal.TaskQueue, worker.Options{
		EnableSessionWorker:               true,
		MaxConcurrentSessionExecutionSize: m.cfg.Worker.MaxConcurrentSessions,
		Interceptors: []interceptor.WorkerInterceptor{
			temporal.NewLoggerInterceptor(m.logger.WithName("worker")),
		},
	})
	m.temporalWorker = w

	w.RegisterWorkflowWithOptions(
		workflow.NewPreprocessingWorkflow(m.cfg.SharedPath).Execute,
		temporalsdk_workflow.RegisterOptions{Name: m.cfg.Temporal.WorkflowName},
	)

	w.RegisterActivityWithOptions(
		activities.NewExtractPackage().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.ExtractPackageName},
	)
	w.RegisterActivityWithOptions(
		activities.NewCheckSipStructure().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.CheckSipStructureName},
	)
	w.RegisterActivityWithOptions(
		activities.NewAllowedFileFormatsActivity().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.AllowedFileFormatsName},
	)
	w.RegisterActivityWithOptions(
		activities.NewMetadataValidationActivity().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.MetadataValidationName},
	)
	w.RegisterActivityWithOptions(
		activities.NewSipCreationActivity().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.SipCreationName},
	)
	// w.RegisterActivityWithOptions(
	// 	   activities.NewSendToFailedBuckeActivity(ft, fs).Execute,
	// 	   temporalsdk_activity.RegisterOptions{Name: activities.SendToFailedBucketName},
	// )
	w.RegisterActivityWithOptions(
		activities.NewRemovePaths().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.RemovePathsName},
	)

	if err := w.Start(); err != nil {
		m.logger.Error(err, "Worker failed to start or fatal error during its execution.")
		return err
	}

	return nil
}

func (m *Main) Close() error {
	if m.temporalWorker != nil {
		m.temporalWorker.Stop()
	}

	if m.temporalClient != nil {
		m.temporalClient.Close()
	}

	return nil
}
