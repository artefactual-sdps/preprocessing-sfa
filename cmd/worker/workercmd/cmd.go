package workercmd

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"ariga.io/sqlcomment"
	"entgo.io/ent/dialect/sql"
	bagit_gython "github.com/artefactual-labs/bagit-gython"
	"github.com/artefactual-sdps/temporal-activities/archiveextract"
	"github.com/artefactual-sdps/temporal-activities/bagcreate"
	"github.com/artefactual-sdps/temporal-activities/bagvalidate"
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
	"github.com/artefactual-sdps/preprocessing-sfa/internal/persistence"
	entclient "github.com/artefactual-sdps/preprocessing-sfa/internal/persistence/ent/client"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/persistence/ent/db"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/workflow"
)

const Name = "preprocessing-worker"

type Main struct {
	logger         logr.Logger
	cfg            config.Configuration
	temporalWorker temporalsdk_worker.Worker
	temporalClient temporalsdk_client.Client
	bagValidator   *bagit_gython.BagIt
	dbClient       *db.Client
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

	m.bagValidator, err = bagit_gython.NewBagIt()
	if err != nil {
		m.logger.Error(err, "Error creating BagIt validator")
		return err
	}

	var psvc persistence.Service
	if m.cfg.CheckDuplicates {
		sqlDB, err := persistence.Open(m.cfg.Persistence.Driver, m.cfg.Persistence.DSN)
		if err != nil {
			m.logger.Error(err, "Error initializing database pool.")
			return err
		}
		m.dbClient = db.NewClient(
			db.Driver(
				sqlcomment.NewDriver(
					sql.OpenDB(m.cfg.Persistence.Driver, sqlDB),
					sqlcomment.WithDriverVerTag(),
					sqlcomment.WithTags(sqlcomment.Tags{
						sqlcomment.KeyApplication: Name,
					}),
				),
			),
		)
		if m.cfg.Persistence.Migrate {
			err = m.dbClient.Schema.Create(ctx)
			if err != nil {
				m.logger.Error(err, "Error migrating database.")
				return err
			}
		}
		psvc = entclient.New(m.dbClient)
	}

	veraPDFValidator := fvalidate.NewVeraPDFValidator(m.cfg.FileValidate.VeraPDF.Path, fvalidate.RunCommand, m.logger)

	veraPDFVersion, err := veraPDFValidator.Version()
	if err != nil {
		return err
	}

	w.RegisterWorkflowWithOptions(
		workflow.NewPreprocessingWorkflow(m.cfg.SharedPath, m.cfg.CheckDuplicates, veraPDFVersion, psvc).Execute,
		temporalsdk_workflow.RegisterOptions{Name: m.cfg.Temporal.WorkflowName},
	)

	w.RegisterActivityWithOptions(
		activities.NewChecksumSIP().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.ChecksumSIPName},
	)
	w.RegisterActivityWithOptions(
		archiveextract.New(archiveextract.Config{}).Execute,
		temporalsdk_activity.RegisterOptions{Name: archiveextract.Name},
	)
	w.RegisterActivityWithOptions(
		bagvalidate.New(m.bagValidator).Execute,
		temporalsdk_activity.RegisterOptions{Name: bagvalidate.Name},
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
		activities.NewValidateSIPName().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.ValidateSIPNameName},
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
			veraPDFValidator,
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
		activities.NewAddPREMISValidationEvent(veraPDFValidator).Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISValidationEventName},
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
	var e error

	if m.temporalWorker != nil {
		m.temporalWorker.Stop()
	}

	if m.temporalClient != nil {
		m.temporalClient.Close()
	}

	if m.bagValidator != nil {
		if err := m.bagValidator.Cleanup(); err != nil {
			e = errors.Join(e, fmt.Errorf("Couldn't clean up bag validator: %v", err))
		}
	}

	if m.dbClient != nil {
		if err := m.dbClient.Close(); err != nil {
			e = errors.Join(e, fmt.Errorf("Couldn't close database client: %v", err))
		}
	}

	return e
}
