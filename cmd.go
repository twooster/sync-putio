package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var configFile = flag.String("config", "", "location of configuration file")

func main() {
	flag.Parse()

	if *configFile == "" {
		fmt.Println("-config must be specified")
		os.Exit(1)
	}

	config, err := LoadConfigFromFile(*configFile)
	if err != nil {
		fmt.Printf("Error loading config file: %v\n", err)
		os.Exit(2)
	}

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ctx, cancel := context.WithCancel(context.Background())

	cancelOnSignals(ctx, logger, cancel)

	done := ctx.Done()

loop:
	for {
		logger.Println("Scanning remote files")
		err := scanAndSyncFiles(ctx, logger, config)
		if err != nil {
			select {
			case <-done:
				break loop
			default:
			}
			logger.Printf("Encountered error, but continuing: %v\n", err)
		}

		logger.Printf("Scan complete, sleeping %v\n", config.Config.ScanInterval.Duration)
		timer := time.NewTimer(config.Config.ScanInterval.Duration)
		select {
		case <-done:
			timer.Stop()
			break loop
		case <-timer.C:
			continue
		}
	}
}

func cancelOnSignals(ctx context.Context, logger *log.Logger, cancel func()) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		logger.Printf("Caught %v, shutting down", sig)
		cancel()
	}()
}
