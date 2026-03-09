package copydir

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"pgv/internal/snapshot"
)

type Driver struct{}

func NewDriver() *Driver {
	return &Driver{}
}

func (d *Driver) Name() string {
	return "copydir"
}

func (d *Driver) CreateSnapshot(ctx context.Context, req snapshot.CreateSnapshotRequest) (snapshot.CreateSnapshotResult, error) {
	if err := copyDirectory(req.SourcePath, req.TargetPath); err != nil {
		return snapshot.CreateSnapshotResult{}, fmt.Errorf("failed to copy to snapshot: %w", err)
	}

	stats, err := d.StatObject(ctx, req.TargetPath)
	if err != nil {
		return snapshot.CreateSnapshotResult{}, err
	}

	return snapshot.CreateSnapshotResult{SizeBytes: stats.SizeBytes}, nil
}

func (d *Driver) CloneSnapshotToBranch(ctx context.Context, req snapshot.CloneSnapshotRequest) (snapshot.CloneSnapshotResult, error) {
	if err := copyDirectory(req.SourcePath, req.TargetPath); err != nil {
		return snapshot.CloneSnapshotResult{}, fmt.Errorf("failed to copy snapshot to branch: %w", err)
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
	return nil
}

// copyDirectory recursively copies a directory tree.
func copyDirectory(scrDir, dest string) error {
	entries, err := os.ReadDir(scrDir)
	if err != nil {
		if os.IsNotExist(err) {
			// If source doesn't exist, we just create empty dest directory
			return os.MkdirAll(dest, 0755)
		}
		return err
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(scrDir, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		stat, err := os.Stat(sourcePath)
		if err != nil {
			return err
		}

		if stat.IsDir() {
			err = copyDirectory(sourcePath, destPath)
			if err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(sourcePath, destPath, stat); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst string, stat os.FileInfo) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}

	// Try to sync to disk
	if err := out.Sync(); err != nil {
		return err
	}

	// Preserve permissions
	return os.Chmod(dst, stat.Mode())
}
