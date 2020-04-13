package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/cavaliercoder/grab"
	"github.com/putdotio/go-putio"
)

func scanAndSyncFiles(ctx context.Context, log *log.Logger, cfg *Config) error {
	c := newPutIOClient(ctx, cfg.Config.Token)
	files, err := listRootFolder(ctx, c)
	if err != nil {
		return err
	}

	avail := make(chan struct{}, cfg.Config.MaxConcurrency)
	for i := 0; i < cfg.Config.MaxConcurrency; i++ {
		avail <- struct{}{}
	}

	for _, sync := range cfg.Sync {
		folderID, err := findDirectoryID(files, sync.Remote)
		if err != nil {
			continue
		}

		syncFolder(ctx, c, log, folderID, sync.Local, false, avail)
	}

	close(avail)

	return nil
}

func syncFolder(ctx context.Context, c *putio.Client, log *log.Logger, folderID int64, dir string, canDelete bool, avail chan struct{}) error {
	stat, err := os.Stat(dir)
	if err != nil {
		log.Printf("Making missing directory %v", dir)
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			return fmt.Errorf("Error making %v: %w", dir, err)
		}
	} else if !stat.IsDir() {
		return fmt.Errorf("Target path %v is not a directory", dir)
	}

	files, err := listFolder(ctx, c, folderID)
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() {
			subDir := path.Join(dir, f.Name)
			if err := syncFolder(ctx, c, log, f.ID, subDir, true, avail); err != nil {
				return err
			}
		} else {
			select {
			case <-ctx.Done():
				return nil
			case s := <-avail:
				err := downloadFile(ctx, c, log, f, dir)
				if err != nil {
					canDelete = false
				}
				avail <- s
			}
		}
	}

	if canDelete {
		c.Files.Delete(ctx, folderID)
		if err != nil {
			return err
		}
		log.Printf("Successfully deleted folder id %v", folderID)
	}

	return nil
}

func downloadFile(ctx context.Context, c *putio.Client, log *log.Logger, file putio.File, dir string) error {
	url, err := c.Files.URL(ctx, file.ID, false)
	if err != nil {
		return err
	}

	log.Printf("Downloading %v (id %v) to %v", file.Name, file.ID, dir)
	_, err = grab.Get(dir, url)
	if err != nil {
		return err
	}
	log.Printf("Successfully downloaded %v (id %v), now deleting", file.Name, file.ID)
	err = c.Files.Delete(ctx, file.ID)
	if err != nil {
		return err
	}
	log.Printf("Successfully deleted %v (id %v)", file.Name, file.ID)

	return nil
}
