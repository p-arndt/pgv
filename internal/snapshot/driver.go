package snapshot

import "context"

type CreateSnapshotRequest struct {
	SourcePath string
	TargetPath string
}

type CreateSnapshotResult struct {
	SizeBytes int64
}

type CloneSnapshotRequest struct {
	SourcePath string
	TargetPath string
}

type CloneSnapshotResult struct{}

type DeleteSnapshotRequest struct {
	TargetPath string
}

type DeleteBranchDataRequest struct {
	TargetPath string
}

type ObjectStats struct {
	SizeBytes int64
}

type ValidateDriverRequest struct {
	TargetPath string
}

type Driver interface {
	Name() string
	CreateSnapshot(ctx context.Context, req CreateSnapshotRequest) (CreateSnapshotResult, error)
	CloneSnapshotToBranch(ctx context.Context, req CloneSnapshotRequest) (CloneSnapshotResult, error)
	DeleteSnapshot(ctx context.Context, req DeleteSnapshotRequest) error
	DeleteBranchData(ctx context.Context, req DeleteBranchDataRequest) error
	StatObject(ctx context.Context, path string) (ObjectStats, error)
	Validate(ctx context.Context, req ValidateDriverRequest) error
}
