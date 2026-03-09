package locks

import (
	"fmt"
	"path/filepath"

	"github.com/gofrs/flock"
	"pgv/internal/util"
)

type RepoLock struct {
	fileLock *flock.Flock
}

func AcquireRepoLock(repoPath string) (*RepoLock, error) {
	lockDir := filepath.Join(repoPath, ".pgv", "run", "locks")
	if err := util.EnsureDir(lockDir); err != nil {
		return nil, fmt.Errorf("failed to create locks dir: %w", err)
	}

	lockFile := filepath.Join(lockDir, "repo.lock")
	fileLock := flock.New(lockFile)

	locked, err := fileLock.TryLock()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire repo lock: %w", err)
	}

	if !locked {
		return nil, fmt.Errorf("PGV repository is currently locked by another process. Please wait and try again")
	}

	return &RepoLock{fileLock: fileLock}, nil
}

func (l *RepoLock) Unlock() error {
	if l.fileLock != nil {
		return l.fileLock.Unlock()
	}
	return nil
}
