package cowfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"pgv/internal/snapshot"
)

type Driver struct{}

func NewDriver() *Driver {
	return &Driver{}
}

func (d *Driver) Name() string {
	return "cowfs"
}

func (d *Driver) CreateSnapshot(ctx context.Context, req snapshot.CreateSnapshotRequest) (snapshot.CreateSnapshotResult, error) {
	if err := cloneDirectory(req.SourcePath, req.TargetPath); err != nil {
		return snapshot.CreateSnapshotResult{}, fmt.Errorf("failed to clone to snapshot: %w", err)
	}

	stats, err := d.StatObject(ctx, req.TargetPath)
	if err != nil {
		return snapshot.CreateSnapshotResult{}, err
	}

	return snapshot.CreateSnapshotResult{SizeBytes: stats.SizeBytes}, nil
}

func (d *Driver) CloneSnapshotToBranch(ctx context.Context, req snapshot.CloneSnapshotRequest) (snapshot.CloneSnapshotResult, error) {
	if err := cloneDirectory(req.SourcePath, req.TargetPath); err != nil {
		return snapshot.CloneSnapshotResult{}, fmt.Errorf("failed to clone snapshot to branch: %w", err)
	}
	return snapshot.CloneSnapshotResult{}, nil
}

func (d *Driver) DeleteSnapshot(ctx context.Context, req snapshot.DeleteSnapshotRequest) error {
	return os.RemoveAll(req.TargetPath)
}

func (d *Driver) DeleteBranchData(ctx context.Context, req snapshot.DeleteBranchDataRequest) error {
	return os.RemoveAll(req.TargetPath)
}

func (d *Driver) StatObject(ctx context.Context, path string) (snapshot.ObjectStats, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return snapshot.ObjectStats{}, err
	}
	return snapshot.ObjectStats{SizeBytes: size}, nil
}

func (d *Driver) Validate(ctx context.Context, req snapshot.ValidateDriverRequest) error {
	// Create a temporary file and try to clone it to test CoW support on the specific target volume.
	tempDir, err := os.MkdirTemp(req.TargetPath, "pgv-cowfs-validate-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	srcPath := filepath.Join(tempDir, "src.txt")
	if err := os.WriteFile(srcPath, []byte("test data"), 0644); err != nil {
		return fmt.Errorf("failed to write test file: %w", err)
	}

	dstPath := filepath.Join(tempDir, "dst.txt")
	if err := cloneFile(srcPath, dstPath); err != nil {
		return fmt.Errorf("CoW not supported on this filesystem: %w", err)
	}

	return nil
}

func cloneDirectory(srcDir, destDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(destDir, 0755)
		}
		return err
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(srcDir, entry.Name())
		destPath := filepath.Join(destDir, entry.Name())

		stat, err := os.Stat(sourcePath)
		if err != nil {
			return err
		}

		if stat.IsDir() {
			if err := cloneDirectory(sourcePath, destPath); err != nil {
				return err
			}
		} else {
			if err := cloneFile(sourcePath, destPath); err != nil {
				return err
			}
			// Best effort preserve permissions. cloneFile might have already done this depending on the OS.
			os.Chmod(destPath, stat.Mode())
		}
	}
	return nil
}
