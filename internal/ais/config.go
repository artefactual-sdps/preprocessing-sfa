package ais

import (
	"github.com/artefactual-sdps/preprocessing-sfa/internal/amss"
	"go.artefactual.dev/tools/bucket"
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
