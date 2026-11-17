package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketUsers = []byte("users")
	bucketStats = []byte("stats")
)

type User struct {
	ID                int64     `json:"id"`
	JoinDate          string    `json:"join_date"`
	LastUsedOn        string    `json:"last_used_on"`
	AsFile            bool      `json:"as_file"`
	WatermarkText     string    `json:"watermark_text"`
	WatermarkColor    int       `json:"watermark_color"`
	WatermarkPosition int       `json:"watermark_position"`
	FontSize          int       `json:"font_size"`
	SampleDuration    int       `json:"sample_duration"`
	ScreenshotMode    int       `json:"screenshot_mode"` // 0: uniform, 1: random
	IsBanned          bool      `json:"is_banned"`
	BanReason         string    `json:"ban_reason"`
	BannedUntil       time.Time `json:"banned_until"`
}

type Storage interface {
	GetUser(id int64) (*User, error)
	SaveUser(u *User) error
	GetOrCreateUser(id int64) (*User, error)
	TotalUsersCount() (int, error)
	GetAllUserIDs() ([]int64, error)
	Close() error
}

type BoltStorage struct {
	db    *bolt.DB
	cache sync.Map
}

func NewBoltStorage(path string) (*BoltStorage, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open bbolt database: %w", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(bucketUsers); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(bucketStats); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize buckets: %w", err)
	}

	return &BoltStorage{db: db}, nil
}

func (s *BoltStorage) GetUser(id int64) (*User, error) {
	if val, ok := s.cache.Load(id); ok {
		if u, ok := val.(*User); ok {
			return u, nil
		}
	}

	var u User
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketUsers)
		data := b.Get([]byte(strconv.FormatInt(id, 10)))
		if data == nil {
			return fmt.Errorf("user not found")
		}
		return json.Unmarshal(data, &u)
	})
	if err != nil {
		return nil, err
	}

	s.cache.Store(id, &u)
	return &u, nil
}

func (s *BoltStorage) SaveUser(u *User) error {
	s.cache.Store(u.ID, u)

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketUsers)
		data, err := json.Marshal(u)
		if err != nil {
			return err
		}
		return b.Put([]byte(strconv.FormatInt(u.ID, 10)), data)
	})
}

func (s *BoltStorage) GetOrCreateUser(id int64) (*User, error) {
	u, err := s.GetUser(id)
	if err == nil {
		return u, nil
	}

	today := time.Now().Format("2006-01-02")
	newUser := &User{
		ID:                id,
		JoinDate:          today,
		LastUsedOn:        today,
		AsFile:            false,
		WatermarkText:     "",
		WatermarkColor:    0, // white
		WatermarkPosition: 4, // Centered
		FontSize:          1, // Medium
		SampleDuration:    30,
		ScreenshotMode:    0, // uniform
		IsBanned:          false,
	}

	if err := s.SaveUser(newUser); err != nil {
		return nil, err
	}
	return newUser, nil
}

func (s *BoltStorage) TotalUsersCount() (int, error) {
	count := 0
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketUsers)
		count = b.Stats().KeyN
		return nil
	})
	return count, err
}

func (s *BoltStorage) GetAllUserIDs() ([]int64, error) {
	var ids []int64
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketUsers)
		return b.ForEach(func(k, v []byte) error {
			id, err := strconv.ParseInt(string(k), 10, 64)
			if err == nil {
				ids = append(ids, id)
			}
			return nil
		})
	})
	return ids, err
}

func (s *BoltStorage) Close() error {
	return s.db.Close()
}
