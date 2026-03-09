package locks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireRepoLock(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Acquire the lock successfully
	lock, err := AcquireRepoLock(tempDir)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}
	if lock == nil {
		t.Fatalf("Expected lock to be non-nil")
	}

	// 2. Ensure lock file exists
	lockFile := filepath.Join(tempDir, ".pgv", "run", "locks", "repo.lock")
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Fatalf("Lock file was not created: %s", lockFile)
	}

	// 3. Attempt to acquire the lock again (should fail)
	_, err2 := AcquireRepoLock(tempDir)
	if err2 == nil {
		t.Fatalf("Expected an error when trying to lock an already locked repo, but got nil")
	}
	expectedErrMsg := "PGV repository is currently locked by another process. Please wait and try again"
	if err2.Error() != expectedErrMsg {
		t.Errorf("Expected error message %q, got %q", expectedErrMsg, err2.Error())
	}

	// 4. Unlock
	err = lock.Unlock()
	if err != nil {
		t.Fatalf("Failed to unlock: %v", err)
	}

	// 5. Attempt to acquire lock after unlocking (should succeed)
	lock2, err := AcquireRepoLock(tempDir)
	if err != nil {
		t.Fatalf("Failed to acquire lock after unlocking: %v", err)
	}
	if lock2 == nil {
		t.Fatalf("Expected lock2 to be non-nil")
	}

	// Clean up second lock
	err = lock2.Unlock()
	if err != nil {
		t.Fatalf("Failed to unlock second lock: %v", err)
	}
}
