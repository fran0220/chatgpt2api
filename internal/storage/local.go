package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
)

const (
	tokenFileName = "token.json"
	lockFileName  = ".storage.lock"
)

type LocalStorage struct {
	mu        sync.Mutex
	dataDir   string
	tokenFile string
	lockFile  string
}

func NewLocalStorage() *LocalStorage {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = defaultDataDir()
	}

	return &LocalStorage{
		dataDir:   dataDir,
		tokenFile: filepath.Join(dataDir, tokenFileName),
		lockFile:  filepath.Join(dataDir, lockFileName),
	}
}

func (s *LocalStorage) LoadTokens() (map[string][]TokenData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data := map[string][]TokenData{}
	err := s.withFileLock(func() error {
		payload, err := os.ReadFile(s.tokenFile)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if len(bytes.TrimSpace(payload)) == 0 {
			return nil
		}
		return json.Unmarshal(payload, &data)
	})
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *LocalStorage) SaveTokens(data map[string][]TokenData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.withFileLock(func() error {
		if err := os.MkdirAll(s.dataDir, 0o755); err != nil {
			return err
		}

		payload, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		payload = append(payload, '\n')

		tmpFile := s.tokenFile + ".tmp"
		if err := os.WriteFile(tmpFile, payload, 0o644); err != nil {
			return err
		}
		return os.Rename(tmpFile, s.tokenFile)
	})
}

func (s *LocalStorage) withFileLock(fn func() error) error {
	if err := os.MkdirAll(s.dataDir, 0o755); err != nil {
		return err
	}

	lockHandle, err := os.OpenFile(s.lockFile, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer lockHandle.Close()

	if err := syscall.Flock(int(lockHandle.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lockHandle.Fd()), syscall.LOCK_UN)

	return fn()
}

func defaultDataDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Clean(filepath.Join(".", "data"))
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "data"))
}
