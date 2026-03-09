# Architecture

PGV is designed to be safe, fast, and fully local. It does not version SQL text (like Flyway or Prisma); it versions **physical Postgres file states**.

## Core Concepts

*   **Repo**: One managed Postgres project (lives in the `.pgv/` directory).
*   **Snapshot**: An immutable, physical copy of a Postgres `PGDATA` directory at a specific point in time.
*   **Branch**: A writable clone derived from a Snapshot.
*   **Head**: The currently active branch.
*   **Instance**: A running Postgres Docker container attached to one specific branch.

## Directory Layout

When you run `pgv init`, it creates a `.pgv/` hidden folder in your project root:

```text
.pgv/
├── config.json              # Repository configuration (ports, credentials)
├── meta/
│   └── state.db             # SQLite database tracking branches/snapshots
├── run/
│   └── locks/
│       └── repo.lock        # Cross-platform file lock for safety
├── storage/
│   ├── branches/            # Writable PGDATA directories per branch
│   │   ├── main/
│   │   └── feature-a/
│   └── snapshots/           # Immutable, read-only PGDATA copies
│       ├── snap_1234abcd/
│       └── snap_5678efgh/
└── logs/                    # Operation logs (future use)
```

## How Operations Work

### Branching (`pgv branch`)
When you branch from a snapshot, PGV simply copies the physical files from `.pgv/storage/snapshots/<id>` into a new writable directory at `.pgv/storage/branches/<new-name>`.

### Checkpointing (`pgv checkpoint`)
When you create a checkpoint, PGV copies the physical files from the active branch into a new immutable snapshot directory. 
*Safety feature:* If the branch's Docker container is currently running, PGV temporarily stops it to ensure the files on disk are completely flushed and consistent before copying them.

### Restoring (`pgv restore`)
Restoring deletes the corrupted/unwanted files in your branch's directory and recursively clones the pristine files from the snapshot back into your branch folder.

### Running (`pgv start`)
PGV uses the Docker SDK to spin up a container. It dynamically maps the specific branch's `PGDATA` folder into the container and binds it to a unique host port (e.g., `5540`, `5541`). This ensures complete isolation.