package downloader

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/rakyll/statik/fs"
	"golang.org/x/time/rate"
	"mkuznets.com/go/ytbackup/internal/database"
	"mkuznets.com/go/ytbackup/internal/pyfs"
	"mkuznets.com/go/ytbackup/internal/venv"
	"mkuznets.com/go/ytbackup/internal/volumes"
)

const (
	maxAttempts      = 3
	ytVideoURLFormat = "https://www.youtube.com/watch?v=%s"
)

type Downloader struct {
	venv    *venv.VirtualEnv
	volumes *volumes.Volumes
}

func New(vs *volumes.Volumes) (*Downloader, error) {
	root := filepath.Join(os.TempDir(), "ytbackup", "venv")

	scriptFS, err := fs.NewWithNamespace(pyfs.Python)
	if err != nil {
		return nil, fmt.Errorf("could not open pyfs: %v", err)
	}

	ve, err := venv.New(root,
		venv.WithFS(scriptFS),
	)
	if err != nil {
		return nil, err
	}

	dl := &Downloader{
		venv:    ve,
		volumes: vs,
	}

	return dl, nil
}

type DownloadResult struct {
	ID         string
	Title      string
	Uploader   string
	UploadDate ISODate `json:"upload_date"`
	File       string
	FileSize   int    `json:"filesize"`
	FileHash   string `json:"filehash"`
	Info       json.RawMessage
}

func (dl *Downloader) Download(ctx context.Context, videoID string) ([]*DownloadResult, error) {
	rctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	url := fmt.Sprintf(ytVideoURLFormat, videoID)

	cargs := []string{"/dl.py", "download", "--log=/tmp/ytbackup/dl.log", url}

	var result []*DownloadResult

	if err := dl.venv.RunScript(rctx, &result, cargs...); err != nil {
		return nil, err
	}
	return result, nil
}

func (dl *Downloader) Serve(ctx context.Context, db *sql.DB) error {
	downloadLimiter := rate.NewLimiter(rate.Every(5*time.Second), 3)

	for {
		select {
		default:
			found := false
			var (
				videoID  string
				attempts int
			)

			err := database.InTx(ctx, db, func(tx *sql.Tx) error {
				err := tx.QueryRowContext(ctx, `
				SELECT video_id, attempts FROM videos as v WHERE status='NEW' ORDER BY id LIMIT 1;
				`).Scan(&videoID, &attempts)

				switch {
				case err == sql.ErrNoRows:
					return nil
				case err != nil:
					return fmt.Errorf("query error: %v", err)
				}

				if attempts >= maxAttempts {
					_, err = tx.ExecContext(ctx, `UPDATE videos SET status='FAILED' WHERE video_id=$1`, videoID)
					if err != nil {
						return fmt.Errorf("update error: %v", err)
					}
					return nil
				}

				if _, err := tx.Exec(`UPDATE videos SET attempts=attempts+1 WHERE video_id=$1`, videoID); err != nil {
					return fmt.Errorf("update error: %v", err)
				}

				found = true

				log.Printf("found %v, attempt #%d", videoID, attempts)

				if err := downloadLimiter.Wait(ctx); err != nil {
					return fmt.Errorf("limiter wait: %v", err)
				}

				if err := dl.processVideo(ctx, tx, videoID); err != nil {
					return err
				}

				return nil
			})
			if err != nil {
				return fmt.Errorf("transaction error: %v", err)
			}

			if !found {
				time.Sleep(5 * time.Second)
			}

		case <-ctx.Done():
			return nil
		}
	}
}

func (dl *Downloader) processVideo(ctx context.Context, tx *sql.Tx, videoID string) error {
	results, err := dl.Download(ctx, videoID)
	if err != nil {
		log.Printf("download error: %v", err)
		return nil
	}

	for _, res := range results {
		dst := filepath.Join(
			res.UploadDate.Format("2006"),
			res.UploadDate.Format("01"),
			filepath.Base(res.File),
		)

		vol, err := dl.volumes.Put(res.File, dst)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `
					INSERT INTO videos
					(status, video_id, title, uploader, upload_date, info, volume, path, filesize, filehash)
					VALUES ('DOWNLOADED', $1, $2, $3, date($4), $5, $6, $7, $8, $9) 
					ON CONFLICT (video_id) DO UPDATE
					SET
						status=excluded.status,
						title=excluded.title,
						uploader=excluded.uploader,
						upload_date=excluded.upload_date,
						info=excluded.info,
						volume=excluded.volume,
						path=excluded.path,
						filesize=excluded.filesize,
						filehash=excluded.filehash
					WHERE videos.status != 'DOWNLOADED';
					`, res.ID, res.Title, res.Uploader, res.UploadDate, res.Info, vol, dst, res.FileSize, res.FileHash)
		if err != nil {
			return fmt.Errorf("insert error: %v", err)
		}

		if err := os.RemoveAll(filepath.Dir(res.File)); err != nil {
			log.Printf("[WARN] could not remove tmp files: %v", err)
			return nil
		}
	}

	return nil
}
