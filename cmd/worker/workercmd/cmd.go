package workercmd

import (
	"context"
	"crypto/rand"

	"github.com/artefactual-sdps/temporal-activities/bagcreate"
	"github.com/artefactual-sdps/temporal-activities/ffvalidate"
	"github.com/artefactual-sdps/temporal-activities/xmlvalidate"
	"github.com/go-logr/logr"
	"go.artefactual.dev/tools/temporal"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_client "go.temporal.io/sdk/client"
	temporalsdk_interceptor "go.temporal.io/sdk/interceptor"
	temporalsdk_worker "go.temporal.io/sdk/worker"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/config"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/fformat"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/fvalidate"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/workflow"
)

const Name = "preprocessing-worker"

type Main struct {
	logger         logr.Logger
	cfg            config.Configuration
	temporalWorker temporalsdk_worker.Worker
	temporalClient temporalsdk_client.Client
}

func NewMain(logger logr.Logger, cfg config.Configuration) *Main {
	return &Main{
		logger: logger,
		cfg:    cfg,
	}
}

func (m *Main) Run(ctx context.Context) error {
	c, err := temporalsdk_client.Dial(temporalsdk_client.Options{
		HostPort:  m.cfg.Temporal.Address,
		Namespace: m.cfg.Temporal.Namespace,
		Logger:    temporal.Logger(m.logger.WithName("preprocessing-temporal")),
	})
	if err != nil {
		m.logger.Error(err, "Unable to create Temporal client.")
		return err
	}
	m.temporalClient = c

	w := temporalsdk_worker.New(m.temporalClient, m.cfg.Temporal.TaskQueue, temporalsdk_worker.Options{
		EnableSessionWorker:               true,
		MaxConcurrentSessionExecutionSize: m.cfg.Worker.MaxConcurrentSessions,
		Interceptors: []temporalsdk_interceptor.WorkerInterceptor{
			temporal.NewLoggerInterceptor(m.logger.WithName(Name)),
		},
	})
	m.temporalWorker = w

	w.RegisterWorkflowWithOptions(
		workflow.NewPreprocessingWorkflow(m.cfg.SharedPath).Execute,
		temporalsdk_workflow.RegisterOptions{Name: m.cfg.Temporal.WorkflowName},
	)

	w.RegisterActivityWithOptions(
		activities.NewUnbag().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.UnbagName},
	)
	w.RegisterActivityWithOptions(
		activities.NewIdentifySIP().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.IdentifySIPName},
	)
	w.RegisterActivityWithOptions(
		activities.NewValidateStructure().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.ValidateStructureName},
	)
	w.RegisterActivityWithOptions(
		activities.NewVerifyManifest().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.VerifyManifestName},
	)
	w.RegisterActivityWithOptions(
		ffvalidate.New(m.cfg.FileFormat).Execute,
		temporalsdk_activity.RegisterOptions{Name: ffvalidate.Name},
	)
	w.RegisterActivityWithOptions(
		activities.NewValidateFiles(
			fformat.NewSiegfriedEmbed(),
			fvalidate.NewVeraPDFValidator(m.cfg.FileValidate.VeraPDF.Path, m.logger),
		).Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.ValidateFilesName},
	)
	w.RegisterActivityWithOptions(
		activities.NewAddPREMISObjects(rand.Reader).Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISObjectsName},
	)
	w.RegisterActivityWithOptions(
		activities.NewAddPREMISEvent().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISEventName},
	)
	w.RegisterActivityWithOptions(
		activities.NewAddPREMISAgent().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISAgentName},
	)
	w.RegisterActivityWithOptions(
		xmlvalidate.New(xmlvalidate.NewXMLLintValidator()).Execute,
		temporalsdk_activity.RegisterOptions{Name: xmlvalidate.Name},
	)
	w.RegisterActivityWithOptions(
		activities.NewValidatePREMIS(xmlvalidate.NewXMLLintValidator()).Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.ValidatePREMISName},
	)
	w.RegisterActivityWithOptions(
		activities.NewTransformSIP().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.TransformSIPName},
	)
	w.RegisterActivityWithOptions(
		activities.NewWriteIdentifierFile().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.WriteIdentifierFileName},
	)
	w.RegisterActivityWithOptions(
		bagcreate.New(m.cfg.Bagit).Execute,
		temporalsdk_activity.RegisterOptions{Name: bagcreate.Name},
	)

	if err := w.Start(); err != nil {
		m.logger.Error(err, "Preprocessing worker failed to start.")
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
