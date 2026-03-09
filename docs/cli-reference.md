# CLI Reference

PGV uses a simple verb-noun command structure.

## Setup & Admin

### `pgv init [repo-name] [--from-dir <path>]`
Initializes a new PGV repository in the current directory.
- `[repo-name]`: Optional name for the repository. Defaults to the current folder name.
- `--from-dir`: Optional path to an existing `PGDATA` directory to instantly import as your `main` branch.

### `pgv list` (or `pgv ls`)
Lists all branches, their current status (stopped/running), ports, and all available snapshots.

### `pgv url [branch-name]`
Outputs the `postgres://` connection string for the specified branch. If no branch is specified, it outputs the URL for the active branch.

## Branch Management

### `pgv start <branch-name>`
Starts an isolated Postgres Docker container for the specified branch.

### `pgv stop <branch-name>`
Stops the Postgres Docker container for the specified branch.

### `pgv checkout <branch-name>`
Switches your active working branch (HEAD).

### `pgv branch <source-snapshot-id> <new-branch-name>`
Creates a new writable branch isolated from the rest of your database, based on a specific immutable snapshot.

### `pgv delete branch <branch-name> [--force]`
Permanently deletes a branch and its physical data. 
- `--force` (or `-f`): Forces deletion even if the branch is currently running or is the active HEAD.

## State Management

### `pgv checkpoint <message>`
Takes an immutable physical snapshot of the active branch. 
*Note: If the branch is running, PGV will automatically stop it, take the physical snapshot, and restart it to prevent data corruption.*

### `pgv restore <snapshot-id> [--branch <branch-name>]`
Replaces the physical state of the specified branch with the contents of a snapshot.
- `--branch`: Optional. If not provided, restores the currently active branch.
*Note: You must run `pgv stop <branch>` before attempting to restore its state.*

### `pgv import <postgres-url>`
Performs a logical `pg_dump` from the provided URL directly into the active, running PGV branch. Uses internal Docker piping so no host binaries are required.