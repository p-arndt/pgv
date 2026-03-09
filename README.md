# PGV — Local Postgres Version Control

PGV is a local developer tool that provides **fast rollback, branching, and restore for PostgreSQL with data included**, without relying on slow logical dump/restore workflows.

It should feel conceptually similar to Git, but it does **not** version SQL text. It versions **physical database state**.

## Installation

### Prerequisites

- Go 1.21 or later
- Docker (for runtime management)
- PostgreSQL (for the database you want to version)

### Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd verges

# Build the binary
go build -o pgv ./cmd/pgv

# Add to your PATH (optional)
# mv pgv /usr/local/bin/
```

### Quick Start

1. Initialize a PGV repository in your project directory:
   ```bash
   pgv init
   ```

2. Create your first checkpoint:
   ```bash
   pgv checkpoint "Initial state"
   ```

3. Start your Postgres instance:
   ```bash
   pgv start main
   ```

## Usage

### Repository Management

#### Initialize a Repository
```bash
pgv init
```
Creates a `.pgv` directory with metadata storage and configuration.

#### List Repositories
```bash
pgv list
```
Shows all snapshots, branches, and their status.

#### Check Status
```bash
pgv status
```
Displays active branch and running instances.

### Snapshots

#### Create a Checkpoint
```bash
pgv checkpoint "before adding users table"
```
Creates an immutable snapshot of the current database state.

#### Tag a Snapshot
```bash
pgv tag <snapshot-id> <tag-name>
```
Assign a human-readable tag to a snapshot.

#### Restore to a Snapshot
```bash
pgv restore <snapshot-id> --branch <branch-name>
```
Restore a branch to a previous snapshot state.

### Branches

#### Create a Branch
```bash
pgv branch <source-snapshot> <branch-name>
```
Create a new writable branch from a snapshot.

#### Switch Branches
```bash
pgv checkout <branch-name>
```
Switch the active branch.

#### Delete a Branch
```bash
pgv delete branch <branch-name>
```
Remove a branch and its data (unless it's active or running).

### Instance Management

#### Start a Branch
```bash
pgv start <branch-name>
```
Start a Postgres instance for the specified branch.

#### Stop a Branch
```bash
pgv stop <branch-name>
```
Stop a running Postgres instance.

#### Get Connection URL
```bash
pgv url <branch-name>
```
Display the connection URL for a branch.

### Advanced Operations

#### Schema Diff
```bash
pgv diff <branch-a> <branch-b>
```
Show schema differences between two branches.

#### Safe Migration
```bash
pgv safe-migrate -- <command>
```
Run migration commands with automatic checkpointing for safety.

#### Garbage Collection
```bash
pgv gc
```
Clean up expired snapshots and temporary branches.

## Workflow Examples

### Safe Migration Testing

1. Create a checkpoint from your current state:
   ```bash
   pgv checkpoint "before migration"
   ```

2. Create a test branch:
   ```bash
   pgv branch before-migration migration-test
   ```

3. Start the test branch:
   ```bash
   pgv start migration-test
   ```

4. Run your migrations on the test branch

5. If something goes wrong, restore the original branch:
   ```bash
   pgv restore before-migration --branch main
   ```

### Parallel Development

Create multiple branches for different features:
```bash
pgv branch main feature-a
pgv branch main feature-b
pgv start feature-a
pgv start feature-b
```

Each branch runs on a different port, allowing you to test features simultaneously.

## Configuration

PGV stores configuration in `.pgv/config.json`. Example configuration:

```json
{
  "repoName": "my-app-db",
  "runtime": "docker",
  "postgresImage": "postgres:17",
  "snapshotDriver": "basebackup",
  "defaultBranch": "main",
  "basePort": 5540,
  "pgUser": "postgres",
  "pgPassword": "postgres",
  "pgDatabase": "app",
  "walArchiveEnabled": true,
  "walArchivePath": ".pgv/wal/archive",
  "retention": {
    "autoSnapshots": 20,
    "tempBranchTTL": "24h"
  }
}
```

## Architecture

PGV manages:
- **Snapshots**: Immutable point-in-time database states
- **Branches**: Writable clones derived from snapshots
- **Instances**: Running Postgres processes/containers attached to branches

The tool uses:
- SQLite for metadata storage
- Docker for runtime management
- Filesystem snapshots for efficient branch creation

## Limitations

- PGV is designed for local development, not production
- It versions physical database state, not SQL migrations
- Data merging between branches is not supported
- Requires Docker for containerized Postgres

## Contributing

Please refer to `PLAN.md` for the project roadmap and implementation details.

## License

[License information to be added]
