package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/oklog/run"
	"github.com/spf13/pflag"
	"go.artefactual.dev/tools/log"

	"github.com/artefactual-sdps/preprocessing-sfa/cmd/worker/aiscmd"
	"github.com/artefactual-sdps/preprocessing-sfa/cmd/worker/workercmd"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/config"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/version"
)

const appName = "preprocessing-sfa-worker"

func main() {
	p := pflag.NewFlagSet(workercmd.Name, pflag.ExitOnError)
	p.String("config", "", "Configuration file")
	p.Bool("version", false, "Show version information")
	if err := p.Parse(os.Args[1:]); err == flag.ErrHelp {
		os.Exit(1)
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if v, _ := p.GetBool("version"); v {
		fmt.Println(version.Info(appName))
		os.Exit(0)
	}

	var cfg config.Configuration
	configFile, _ := p.GetString("config")
	configFileFound, configFileUsed, err := config.Read(&cfg, configFile)
	if err != nil {
		fmt.Printf("Failed to read configuration: %v\n", err)
		os.Exit(1)
	}

	logger := log.New(os.Stderr,
		log.WithName(workercmd.Name),
		log.WithDebug(cfg.Debug),
		log.WithLevel(cfg.Verbosity),
	)
	defer log.Sync(logger)

	keys := []interface{}{
		"version", version.Long,
		"pid", os.Getpid(),
		"go", runtime.Version(),
	}
	if version.GitCommit != "" {
		keys = append(keys, "commit", version.GitCommit)
	}
	logger.Info("Starting...", keys...)

	if configFileFound {
		logger.Info("Configuration file loaded.", "path", configFileUsed)
	} else {
		logger.Info("Configuration file not found.")
	}

	ctx, cancel := context.WithCancel(context.Background())
	var g run.Group

	// Preprocessing worker.
	{
		done := make(chan struct{})
		m := workercmd.NewMain(logger, cfg)
		g.Add(
			func() error {
				if err := m.Run(ctx); err != nil {
					return err
				}
				<-done
				return nil
			},
			func(err error) {
				if err := m.Close(); err != nil {
					logger.Error(err, "Failed to close preprocessing worker.")
				}
				close(done)
			},
		)
	}

	// AIS worker.
	{
		done := make(chan struct{})
		m := aiscmd.NewMain(logger, cfg.AIS)
		g.Add(
			func() error {
				if err := m.Run(ctx); err != nil {
					return err
				}
				<-done
				return nil
			},
			func(err error) {
				if err := m.Close(); err != nil {
					logger.Error(err, "Failed to close AIS worker.")
				}
				close(done)
			},
		)
	}

	// Signal handler.
	{
		var (
			cancelInterrupt = make(chan struct{})
			ch              = make(chan os.Signal, 2)
		)
		defer close(ch)

		g.Add(
			func() error {
				signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

				select {
				case <-ch:
				case <-cancelInterrupt:
				}

				return nil
			},
			func(err error) {
				logger.Info("Quitting...")
				close(cancelInterrupt)
				cancel()
				signal.Stop(ch)
			},
		)
	}

	err = g.Run()
	if err != nil {
		logger.Error(err, "Application failure.")
		os.Exit(1)
	}
	logger.Info("Bye!")
}
