package ais

import (
	"go.artefactual.dev/tools/bucket"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/amss"
)

type Config struct {
	WorkingDir string
	Temporal   TemporalConfig
	Worker     WorkerConfig
	AMSS       amss.Config
	Bucket     bucket.Config
}

type TemporalConfig struct {
	Address      string
	Namespace    string
	TaskQueue    string
	WorkflowName string
}

type WorkerConfig struct {
	MaxConcurrentSessions int
}
