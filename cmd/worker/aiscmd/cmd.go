package aiscmd

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"go.artefactual.dev/tools/bucket"
	"go.artefactual.dev/tools/temporal"
	temporalsdk_client "go.temporal.io/sdk/client"
	temporalsdk_interceptor "go.temporal.io/sdk/interceptor"
	temporalsdk_worker "go.temporal.io/sdk/worker"
	"gocloud.dev/blob"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/ais"
)

const Name = "ais-worker"

type Main struct {
	logger         logr.Logger
	cfg            ais.Config
	temporalWorker temporalsdk_worker.Worker
	temporalClient temporalsdk_client.Client
	bucket         *blob.Bucket
}

func NewMain(logger logr.Logger, cfg ais.Config, tc temporalsdk_client.Client) *Main {
	return &Main{
		logger:         logger,
		cfg:            cfg,
		temporalClient: tc,
	}
}

func (m *Main) Run(ctx context.Context) error {
	w := temporalsdk_worker.New(m.temporalClient, m.cfg.Temporal.TaskQueue, temporalsdk_worker.Options{
		EnableSessionWorker:               true,
		MaxConcurrentSessionExecutionSize: m.cfg.Worker.MaxConcurrentSessions,
		Interceptors: []temporalsdk_interceptor.WorkerInterceptor{
			temporal.NewLoggerInterceptor(m.logger.WithName(Name)),
		},
	})
	m.temporalWorker = w

	b, err := bucket.NewWithConfig(ctx, &m.cfg.Bucket)
	if err != nil {
		return fmt.Errorf("Unable to open AIS bucket: %w", err)
	}
	m.bucket = b

	if err := ais.RegisterWorkflow(ctx, w, m.cfg, b); err != nil {
		return fmt.Errorf("AIS: %w", err)
	}

	if err := w.Start(); err != nil {
		m.logger.Error(err, "Worker failed to start.")
		return err
	}

	return nil
}

func (m *Main) Close() error {
	if m.temporalWorker != nil {
		m.temporalWorker.Stop()
	}

	if m.bucket != nil {
		return m.bucket.Close()
	}

	return nil
}
