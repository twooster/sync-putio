package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/cavaliercoder/grab"
	"github.com/putdotio/go-putio"
)

func scanAndSyncFiles(ctx context.Context, logger *log.Logger, cfg *Config) error {
	c := newPutIOClient(ctx, cfg.Config.Token)
	rootFiles, rootFolder, err := c.Files.List(ctx, 0)
	if err != nil {
		return fmt.Errorf("Error listing PutIO root folder: %w", err)
	}

	// Simple worker pool
	avail := make(chan struct{}, cfg.Config.MaxConcurrency)
	for i := 0; i < cfg.Config.MaxConcurrency; i++ {
		avail <- struct{}{}
	}
	defer close(avail)

	for _, sync := range cfg.Sync {
		if sync.Remote == "" {
			err := syncFolder(ctx, c, logger, rootFolder, sync.Local, false, avail)
			if err != nil {
				logger.Printf("Ecountered error syncing remote '%v' (id %v) to '%v', but continuing: %v\n", rootFolder.Name, rootFolder.ID, sync.Local, err)
			}
		} else {
			found := false
		inner:
			for _, f := range rootFiles {
				if f.Name == sync.Remote {
					found = true
					err := syncFolder(ctx, c, logger, f, sync.Local, false, avail)
					if err != nil {
						logger.Printf("Ecountered error syncing remote '%v' (id %v) to '%v', but continuing: %v\n", f.Name, f.ID, sync.Local, err)
					}
					break inner
				}
			}
			if !found {
				logger.Printf("Remote folder '%v' not found, not syncing\n", sync.Remote)
			}
		}
	}

	return nil
}

func syncFolder(ctx context.Context, c *putio.Client, logger *log.Logger, folder putio.File, dir string, canDelete bool, avail chan struct{}) error {
	stat, err := os.Stat(dir)
	if err != nil {
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			return fmt.Errorf("Error making local directory '%v': %w", dir, err)
		}
	} else if !stat.IsDir() {
		return fmt.Errorf("Target local path '%v' is not a directory", dir)
	}

	files, err := listFolder(ctx, c, folder.ID)
	if err != nil {
		return fmt.Errorf("Listing remote folder failed: %v", err)
	}

	if len(files) == 0 {
		logger.Printf("Remote folder %v (id: '%v') empty\n", folder.Name, folder.ID)
	} else {
		for _, f := range files {
			if f.IsDir() {
				subDir := path.Join(dir, f.Name)
				if err := syncFolder(ctx, c, logger, f, subDir, true, avail); err != nil {
					return err
				}
			} else {
				select {
				case <-ctx.Done():
					return errors.New("Aborted")
				case s := <-avail:
					if err := downloadFile(ctx, c, logger, f, dir); err != nil {
						canDelete = false
						logger.Printf("Encountered error, but continuing: %v\n", err)
					}
					avail <- s
				}
			}
		}
	}

	if canDelete {
		err = c.Files.Delete(ctx, folder.ID)
		if err != nil {
			return fmt.Errorf("Deleting folder '%v' (id %v) remotely failed: %v", folder.Name, folder.ID, err)
		}
		logger.Printf("Successfully deleted remote folder '%v' (id %v)\n", folder.Name, folder.ID)
	}

	return nil
}

func downloadFile(ctx context.Context, c *putio.Client, logger *log.Logger, file putio.File, dir string) error {
	url, err := c.Files.URL(ctx, file.ID, false)
	if err != nil {
		return fmt.Errorf("Generating URL failed: %w", err)
	}
	logger.Printf("Downloading '%v' (id %v) to %v\n", file.Name, file.ID, dir)
	_, err = grab.Get(dir, url)
	if err != nil {
		return fmt.Errorf("Downloading failed: %w", err)
	}
	logger.Printf("Successfully downloaded '%v' (id %v)\n", file.Name, file.ID)
	err = c.Files.Delete(ctx, file.ID)
	if err != nil {
		return fmt.Errorf("Deleting remote file failed: %w", err)
	}
	logger.Printf("Successfully deleted remote file '%v' (id %v)\n", file.Name, file.ID)

	return nil
}
