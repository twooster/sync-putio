package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"log"
	"os"
	"path"
	"sync"
	"time"

	"github.com/cavaliercoder/grab"
	"github.com/putdotio/go-putio"
)

type Syncer struct {
	RateLimiter grab.RateLimiter
	PutioClient *putio.Client
	GrabClient  *grab.Client
	Ctx         context.Context
	Logger      *log.Logger
	Syncs       []syncSection
	pool        chan struct{}
}

func NewSyncer(ctx context.Context, logger *log.Logger, c *Config) *Syncer {
	rateLimiter := NewLimiter(c.Config.MaxKbPerSecond * 1024)

	var pool chan struct{}
	if c.Config.MaxConcurrency > 0 {
		pool = make(chan struct{}, c.Config.MaxConcurrency)
	}

	return &Syncer{
		PutioClient: newPutIOClient(ctx, c.Config.Token),
		GrabClient:  grab.NewClient(),
		Ctx:         ctx,
		RateLimiter: rateLimiter,
		Logger:      logger,
		Syncs:       c.Sync,
		pool:        pool,
	}
}

func (s *Syncer) fetchUrlToFile(url string, dst string, checksum []byte) error {
	req, err := grab.NewRequest(dst, url)
	if err != nil {
		return err
	}
	req = req.WithContext(s.Ctx)
	req.RateLimiter = s.RateLimiter
	if checksum != nil {
		req.SetChecksum(crc32.NewIEEE(), checksum, true)
	}

	s.Logger.Printf("Downloading '%v' to '%v'", url, dst)

	resp := s.GrabClient.Do(req)
	t := time.NewTicker(30 * time.Second)

loop:
	for {
		select {
		case <-t.C:
			s.Logger.Printf("Status '%v': %v (%.1f%%) @ %v/s", url, bytesToHuman(resp.BytesComplete()), resp.Progress()*100, bytesToHuman(int64(resp.BytesPerSecond())))
		case <-resp.Done:
			t.Stop()
			break loop
		}
	}

	if err := resp.Err(); err != nil {
		return err
	}

	s.Logger.Printf("Completed download '%v'\n", url)
	return nil
}

func (s *Syncer) fetchPutioFileToFile(file putio.File, dst string) error {
	url, err := s.PutioClient.Files.URL(s.Ctx, file.ID, false)
	if err != nil {
		return fmt.Errorf("Generating URL failed: %w", err)
	}

	var crc32Bytes []byte
	if file.CRC32 != "" {
		crc32Bytes, err = hex.DecodeString(file.CRC32)
		if err != nil {
			s.Logger.Printf("Error decoding CRC32 for '%v': %v\n", file.Name, err)
			crc32Bytes = nil
		}
	}

	err = s.fetchUrlToFile(url, dst, crc32Bytes)
	if err != nil {
		return fmt.Errorf("Downloading '%v' failed: %w", file.Name, err)
	}

	err = s.PutioClient.Files.Delete(s.Ctx, file.ID)
	if err != nil {
		return fmt.Errorf("Deleting remote file failed: %w", err)
	}
	s.Logger.Printf("Successfully deleted remote file '%v'\n", file.Name)

	return nil
}

func (s *Syncer) acquire() {
	if s.pool != nil {
		s.pool <- struct{}{}
	}
}

func (s *Syncer) release() {
	if s.pool != nil {
		<-s.pool
	}
}

func (s *Syncer) Sync() error {
	rootFiles, rootFolder, err := s.PutioClient.Files.List(s.Ctx, 0)
	if err != nil {
		return fmt.Errorf("Error listing PutIO root folder: %w", err)
	}

outer:
	for _, sync := range s.Syncs {
		found := false
	inner:
		for _, f := range rootFiles {
			if f.Name == sync.Remote {
				found = true
				err := s.syncFolder(f, sync.Local, false)
				if err != nil {
					if err == context.Canceled {
						break outer
					}
					s.Logger.Printf("Encountered error syncing remote '%v' to '%v', but continuing: %v\n", rootFolder.Name, sync.Local, err)
				}
				break inner
			}
		}
		if !found {
			s.Logger.Printf("Remote folder '%v' not found, not syncing\n", sync.Remote)
		}
	}

	return nil
}

func ensureDir(dir string) error {
	stat, err := os.Stat(dir)
	if err != nil {
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			return fmt.Errorf("Error making local directory '%v': %w", dir, err)
		}
	} else if !stat.IsDir() {
		return fmt.Errorf("Target local path '%v' is not a directory", dir)
	}
	return nil
}

func (s *Syncer) syncFolder(folder putio.File, dir string, deleteAfter bool) error {
	err := ensureDir(dir)
	if err != nil {
		return err
	}

	files, _, err := s.PutioClient.Files.List(s.Ctx, folder.ID)
	if err != nil {
		return fmt.Errorf("Listing remote folder failed: %v", err)
	}

	if len(files) == 0 {
		s.Logger.Printf("Remote folder '%v' empty\n", folder.Name)
	} else {
		wg := sync.WaitGroup{}

		for _, f := range files {
			if err := s.Ctx.Err(); err != nil {
				break
			}

			f := f
			wg.Add(1)
			go func() {
				defer wg.Done()
				var err error
				if f.IsDir() {
					subDir := path.Join(dir, f.Name)
					err = s.syncFolder(f, subDir, true)
				} else {
					err = s.downloadFile(f, dir)

				}
				if err != nil {
					if err != context.Canceled {
						s.Logger.Printf("Encountered error, but continuing: %v\n", err)
					}
					deleteAfter = false
				}
			}()
		}

		wg.Wait()
	}

	if deleteAfter {
		err = s.PutioClient.Files.Delete(s.Ctx, folder.ID)
		if err != nil {
			return fmt.Errorf("Deleting folder '%v' remotely failed: %v", folder.Name, err)
		}
		s.Logger.Printf("Successfully deleted remote folder '%v'\n", folder.Name)
	}

	return nil
}

func (s *Syncer) downloadFile(f putio.File, dir string) error {
	s.Logger.Printf("Enqueue %v\n", f.Name)
	s.acquire()
	defer s.release()

	return s.fetchPutioFileToFile(f, dir)
}
