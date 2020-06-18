package index

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	bolt "go.etcd.io/bbolt"
)

var (
	maxAttempts    = 3
	retryDelay     = 30 * time.Second
	bucketItems    = []byte("items")
	bucketStatuses = []byte("statuses")
	ErrStop        = errors.New("map stop")
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
	beatLock           sync.Mutex
	beats              map[string]time.Time
}

func New(path string) *Index {
	return &Index{
		path:               path,
		timeout:            5 * time.Minute,
		timeoutCheckPeriod: time.Minute,
		wg:                 &sync.WaitGroup{},
		beats:              make(map[string]time.Time),
	}
}

func (st *Index) Init() error {
	if st.db != nil {
		return nil
	}

	db, err := bolt.Open(st.path, os.FileMode(0644), &bolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		if errors.Is(err, bolt.ErrTimeout) {
			return fmt.Errorf("index database is locked (probably, by another ytbackup instance)")
		}
		return fmt.Errorf("could not open index database at %s: %v", st.path, err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		for _, name := range [2][]byte{bucketItems, bucketStatuses} {
			_, err := tx.CreateBucketIfNotExists(name)
			if err != nil {
				return fmt.Errorf("could not create index bucket: %s", err)
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
		for _, id := range ids {
			ok, err := put(tx, &Video{ID: id, Status: StatusNew}, false)
			if err != nil {
				return err
			}
			if ok {
				total++
			}
		}
		return nil
	})

	return total, err
}

func (st *Index) Pop(n int) ([]*Video, error) {
	items := make([]*Video, 0, n)

	err := st.db.Update(func(tx *bolt.Tx) error {
		i := 0

		return mapItems(tx, StatusEnqueued, func(video *Video) error {
			if video.RetryAfter != nil && video.RetryAfter.After(time.Now()) {
				return nil
			}

			deadline := time.Now().Add(st.timeout)
			video.Deadline = &deadline
			video.Status = StatusInProgress

			_, err := put(tx, video, true)
			if err != nil {
				return err
			}

			items = append(items, video)
			i++
			if i >= n {
				return ErrStop
			}

			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (st *Index) Get(status Status, n int) ([]*Video, error) {
	i := 0
	videos := make([]*Video, 0, n)

	err := st.Map(status, func(video *Video) error {
		videos = append(videos, video)
		i++
		if i >= n {
			return ErrStop
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return videos, nil
}

func (st *Index) Retry(id string, mode RetryMode) error {
	return st.db.Update(func(tx *bolt.Tx) error {
		value := tx.Bucket(bucketItems).Get([]byte(id))
		if value == nil {
			return nil
		}

		var video Video
		if err := json.Unmarshal(value, &video); err != nil {
			return err
		}

		video.Status = StatusEnqueued

		if mode == RetryLimited {
			video.Attempt++
			after := time.Now().Add(retryDelay)
			video.RetryAfter = &after

			if video.Attempt > maxAttempts {
				log.Info().Str("id", video.ID).Msg("Retry limit reached")
				video.Status = StatusFailed
			}
		}

		log.Info().Str("id", id).Msg("Retry later")
		if _, err := put(tx, &video, true); err != nil {
			return err
		}

		return nil
	})
}

func (st *Index) Put(videos ...*Video) error {
	return st.db.Update(func(tx *bolt.Tx) error {
		for _, video := range videos {
			v := *video
			v.ClearSystem()
			if _, err := put(tx, &v, true); err != nil {
				return err
			}
		}
		return nil
	})
}

func (st *Index) Map(status Status, f func(*Video) error) error {
	return st.db.View(func(tx *bolt.Tx) error {
		return mapItems(tx, status, func(video *Video) error {
			return f(video)
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
		return mapItems(tx, StatusInProgress, func(video *Video) error {
			if video.Deadline == nil {
				log.Error().
					Str("id", video.ID).
					Str("status", string(StatusInProgress)).
					Msg("Video does not have a deadline")
				return nil
			}

			now := time.Now()

			if beat, ok := st.beats[video.ID]; ok && beat.After(now) {
				video.Deadline = &beat
				if _, err := put(tx, video, true); err != nil {
					return err
				}
				return nil
			}

			if video.Deadline.Before(now) {
				log.Debug().Str("id", video.ID).Msg("Download timed out, retrying")
				video.Deadline = nil
				video.Status = StatusEnqueued

				if _, err := put(tx, video, true); err != nil {
					return err
				}
			}
			return nil
		})
	})
}

func (st *Index) Beat(id string) {
	st.beatLock.Lock()
	defer st.beatLock.Unlock()
	for k := range st.beats {
		delete(st.beats, k)
	}
	st.beats[id] = time.Now().Add(st.timeout)
}

func (st *Index) Check() error {
	items := make(map[string]*Video)
	statusIDs := make(map[string]string)

	return st.db.View(func(tx *bolt.Tx) error {
		err := tx.Bucket(bucketItems).ForEach(func(k, v []byte) error {
			var video Video
			if err := json.Unmarshal(v, &video); err != nil {
				return err
			}
			if !bytes.Equal(video.Key(), k) {
				return fmt.Errorf("invalid item key: [%q] = Video{ID: %q}", k, video.ID)
			}
			items[video.ID] = &video
			return nil
		})
		if err != nil {
			return err
		}

		err = tx.Bucket(bucketStatuses).ForEach(func(k, v []byte) error {
			ps := bytes.Split(k, []byte("::"))
			if len(ps) != 2 {
				return fmt.Errorf("invalid status key: %q", k)
			}
			status, id := ps[0], ps[1]
			if !bytes.Equal(id, v) {
				return fmt.Errorf("invalid status value: [%q] = %q", k, v)
			}

			st, ok := statusIDs[string(v)]
			if ok {
				return fmt.Errorf("multiple statuses for id=%q: %s and %s", id, st, status)
			}
			statusIDs[string(v)] = string(status)

			video, ok := items[string(v)]
			if !ok {
				return fmt.Errorf("missing item for status %s", k)
			}

			if string(video.Status) != string(status) {
				return fmt.Errorf("status mismatch: %q vs Video{ID: %q, Status: %q}", k, video.ID, video.Status)
			}

			return nil
		})
		if err != nil {
			return err
		}
		return nil
	})
}

func put(tx *bolt.Tx, video *Video, replace bool) (ok bool, err error) {
	if video.Status == "" {
		panic(".Status is required")
	}

	oldItem := tx.Bucket(bucketItems).Get(video.Key())
	if oldItem != nil {
		if !replace {
			return false, nil
		}

		var oldVideo Video
		if err := json.Unmarshal(oldItem, &oldVideo); err != nil {
			return false, fmt.Errorf("could not unmarshal key %s: %v", video.Key(), err)
		}
		if err := tx.Bucket(bucketStatuses).Delete(oldVideo.StatusKey()); err != nil {
			return false, err
		}
	}

	value, err := json.Marshal(video)
	if err != nil {
		return false, fmt.Errorf("could not serialise Video: %v", err)
	}

	if err := tx.Bucket(bucketItems).Put(video.Key(), value); err != nil {
		return false, err
	}
	if err := tx.Bucket(bucketStatuses).Put(video.StatusKey(), video.Key()); err != nil {
		return false, err
	}

	return true, nil
}

func mapItems(tx *bolt.Tx, status Status, f func(*Video) error) error {
	cur := tx.Bucket(bucketStatuses).Cursor()
	prefix := []byte(status)

	for key, videoID := cur.Seek(prefix); key != nil && bytes.HasPrefix(key, prefix); key, videoID = cur.Next() {
		data := tx.Bucket(bucketItems).Get(videoID)
		if data == nil {
			return fmt.Errorf("inconsistent index: %s", videoID)
		}
		var video Video
		if err := json.Unmarshal(data, &video); err != nil {
			return err
		}
		if err := f(&video); err != nil {
			if errors.Is(err, ErrStop) {
				return nil
			}
			return err
		}
	}

	return nil
}
