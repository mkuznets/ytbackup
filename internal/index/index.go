package index

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	bolt "go.etcd.io/bbolt"
)

var (
	maxAttempts      = 3
	retryDelay       = 30 * time.Second
	bucketNew        = []byte("NEW")
	bucketInProgress = []byte("INPROGRESS")
	bucketDone       = []byte("DONE")
	bucketFailed     = []byte("FAILED")
	allBuckets       = [4][]byte{bucketNew, bucketInProgress, bucketDone, bucketFailed}
)

type RetryMode uint8

const (
	RetryInfinite RetryMode = 1 + iota
	RetryLimited
)

type Index struct {
	path               string
	db                 *bolt.DB
	timeout            time.Duration
	timeoutCheckPeriod time.Duration
	wg                 *sync.WaitGroup
	cancel             context.CancelFunc
}

func New(path string) *Index {
	return &Index{
		path:               path,
		timeout:            20 * time.Minute,
		timeoutCheckPeriod: time.Second,
		wg:                 &sync.WaitGroup{},
	}
}

func (st *Index) Init() error {
	if st.db != nil {
		return nil
	}

	db, err := bolt.Open(st.path, os.FileMode(0644), nil)
	if err != nil {
		return fmt.Errorf("could not open database: %v", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		for _, name := range allBuckets {
			_, err := tx.CreateBucketIfNotExists(name)
			if err != nil {
				return fmt.Errorf("create bucket: %s", err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	st.cancel = cancel

	st.wg.Add(1)
	go st.ensureTimeout(ctx)

	st.db = db

	return nil
}

func (st *Index) Close() error {
	if st.db == nil {
		return nil
	}
	st.cancel()
	st.wg.Wait()

	err := st.db.Close()
	if err != nil {
		return err
	}
	log.Debug().Msg("Index closed")
	return nil
}

func (st *Index) Push(ids []string) (int, error) {
	total := 0

	err := st.db.Update(func(tx *bolt.Tx) error {
	IDS:
		for _, id := range ids {
			for _, name := range allBuckets {
				if tx.Bucket(name).Get([]byte(id)) != nil {
					continue IDS
				}
			}
			if err := put(tx, bucketNew, &Video{ID: id}); err != nil {
				return err
			}
			total++
		}
		return nil
	})

	return total, err
}

func (st *Index) Pop(n int) ([]*Video, error) {
	items := make([]*Video, 0, n)

	err := st.db.Update(func(tx *bolt.Tx) error {
		bNew := tx.Bucket(bucketNew)

		c := bNew.Cursor()
		i := 0

		for key, value := c.First(); key != nil && i < n; key, value = c.Next() {
			var video Video
			if err := json.Unmarshal(value, &video); err != nil {
				return err
			}

			if video.RetryAfter != nil && video.RetryAfter.After(time.Now()) {
				continue
			}

			if err := bNew.Delete(key); err != nil {
				return err
			}

			if video.Attempt > maxAttempts {
				log.Info().Str("id", video.ID).Msg("Retry limit reached")
				if err := put(tx, bucketFailed, &video); err != nil {
					return err
				}
				continue
			}

			deadline := time.Now().Add(st.timeout)
			video.Deadline = &deadline

			if err := put(tx, bucketInProgress, &video); err != nil {
				return err
			}

			items = append(items, &video)
			i++
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (st *Index) Retry(id string, mode RetryMode) error {
	log.Info().Str("id", id).Msg("Retry")

	return st.db.Update(func(tx *bolt.Tx) error {
		key := []byte(id)
		value := tx.Bucket(bucketInProgress).Get(key)
		if value == nil {
			return nil
		}

		var video Video
		if err := json.Unmarshal(value, &video); err != nil {
			return err
		}

		if mode == RetryLimited {
			video.Attempt++
		}

		after := time.Now().Add(retryDelay)
		video.RetryAfter = &after

		if err := tx.Bucket(bucketInProgress).Delete(key); err != nil {
			return err
		}
		if err := put(tx, bucketNew, &video); err != nil {
			return err
		}

		return nil
	})
}

func (st *Index) Done(video *Video) error {
	return st.db.Update(func(tx *bolt.Tx) error {
		v := *video
		v.Deadline = nil

		if err := put(tx, bucketDone, &v); err != nil {
			return err
		}

		for _, name := range [2][]byte{bucketNew, bucketInProgress} {
			if err := tx.Bucket(name).Delete(video.Key()); err != nil {
				return err
			}
		}

		return nil
	})
}

func (st *Index) ListDone(f func(*Video) error) error {
	return st.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketDone).ForEach(func(key, value []byte) error {
			var video Video
			if err := json.Unmarshal(value, &video); err != nil {
				return err
			}
			if err := f(&video); err != nil {
				return fmt.Errorf("list callback: %v", err)
			}
			return nil
		})
	})
}

func (st *Index) ensureTimeout(ctx context.Context) {
	ticker := time.NewTicker(st.timeoutCheckPeriod)
	for {
		select {
		case <-ctx.Done():
			st.wg.Done()
			return
		case <-ticker.C:
			if err := st.ensureTimeoutOnce(); err != nil {
				log.Warn().Err(err).Msg("ensureTimeout error")
			}
		}
	}
}

func (st *Index) ensureTimeoutOnce() error {
	return st.db.Update(func(tx *bolt.Tx) error {
		bInProgress := tx.Bucket(bucketInProgress)

		return bInProgress.ForEach(func(key, value []byte) error {
			var video Video

			if err := json.Unmarshal(value, &video); err != nil {
				return fmt.Errorf("could not unmarshal key %s in %s: %v", key, bucketInProgress, err)
			}

			if video.Deadline == nil {
				log.Error().
					Bytes("id", key).
					Bytes("bucket", bucketInProgress).
					Msg("Video does not have a deadline")
				return nil
			}

			if video.Deadline.Before(time.Now()) {
				log.Warn().Bytes("id", key).Msg("Download timed out, retrying")
				video.Deadline = nil

				if err := put(tx, bucketNew, &video); err != nil {
					return err
				}
				if err := bInProgress.Delete(key); err != nil {
					return err
				}
			}

			return nil
		})
	})
}

func put(tx *bolt.Tx, bucket []byte, video *Video) error {
	value, err := json.Marshal(video)
	if err != nil {
		return fmt.Errorf("could not serialise Video: %v", err)
	}
	return tx.Bucket(bucket).Put(video.Key(), value)
}
