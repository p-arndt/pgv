# PGV — Local Postgres Version Control

## 1. Purpose

PGV is a local developer tool that provides **fast rollback, branching, and restore for PostgreSQL with data included**, without relying on slow logical dump/restore workflows.

It should feel conceptually similar to Git, but it does **not** version SQL text. It versions **physical database state**.

Primary user problem:

* A developer is running Postgres locally, often inside Docker.
* They apply migrations or destructive data changes.
* They want to return to an older database state **including all rows**, quickly.
* `pg_dump` / `pg_restore` is too slow for large datasets.

PGV solves this by managing:

* immutable snapshots
* writable branches
* local Postgres instances attached to branches
* fast restore/checkout operations

---

## 2. Product definition

### Core idea

A PGV repository manages one local Postgres workspace.

Key concepts:

* **Repo**: one managed Postgres project
* **Snapshot**: immutable point-in-time database state
* **Branch**: writable clone derived from a snapshot
* **Head**: current active branch
* **Instance**: running Postgres process/container attached to one branch
* **Tag**: human-friendly label for a snapshot

### What the product is

PGV is:

* a local CLI-first developer tool
* a branch/snapshot manager for PostgreSQL state
* optimized for local development workflows
* focused on speed, safety, and easy rollback

### What the product is not

PGV is not:

* a replacement for migration tooling
* a logical SQL diff engine
* a row-level merge system
* a production disaster recovery platform
* a hosted cloud branching service

---

## 3. Goals

### Primary goals

1. **Fast rollback with data**

   * restoring a prior state should be much faster than dump/restore

2. **Safe experimentation**

   * developers can create temporary branches before risky migrations or deletes

3. **Simple local UX**

   * Git-like mental model
   * clear CLI
   * minimal required setup

4. **Works with Docker-based local development**

   * support containerized Postgres as first-class scenario

5. **Clear implementation path**

   * v1 should be realistically implementable by an AI/code agent

### Secondary goals

* schema diff between branches
* migration wrappers
* automatic cleanup of temporary branches
* optional local UI later

---

## 4. Non-goals

These are explicitly out of scope for v1:

* merging two branches' data changes automatically
* generic full-row data diffing
* support for arbitrary production clusters
* multi-node Postgres topologies
* hot-swapping PGDATA under a live Postgres process
* cross-major-version physical compatibility
* table-level or schema-level partial restore

---

## 5. Primary use cases

### Use case A — safe migration testing

1. developer creates checkpoint from current main branch
2. developer creates branch `migration-test`
3. developer runs migrations on that branch
4. developer validates schema and app behavior
5. developer either keeps the branch or deletes it

### Use case B — rollback after destructive local change

1. developer has branch `main`
2. developer accidentally deletes or mutates large data
3. developer restores `main` from snapshot `before-delete`
4. database returns with original data, quickly

### Use case C — parallel local experiments

1. developer creates branches `feature-a-db` and `feature-b-db`
2. each branch runs in its own Postgres container and port
3. developer can compare branch schemas or run app against either instance

### Use case D — seeded local environments

1. developer imports a realistic dev dataset once
2. they create snapshots from that seeded state
3. new branches can be created repeatedly from that base

---

## 6. Product requirements

### Functional requirements

#### FR-1 Repository initialization

* initialize a PGV repo in a project directory
* detect local Postgres runtime configuration
* create metadata storage and managed directories

#### FR-2 Snapshot creation

* create immutable snapshots from a branch
* attach optional label/message
* store snapshot ancestry and metadata

#### FR-3 Branch creation

* create writable branch from snapshot
* allow naming branch
* track parent snapshot

#### FR-4 Instance management

* start branch as a Postgres container/process
* stop running branch
* report status and connection URL

#### FR-5 Checkout / switch

* switch active working branch
* update repo metadata to reflect new head

#### FR-6 Restore

* replace branch writable state with chosen snapshot state
* preserve branch identity while rewinding its contents

#### FR-7 Listing and inspection

* list snapshots, branches, tags, sizes, and ancestry
* show active branch and running instances

#### FR-8 Delete branch

* delete disposable branch
* protect active/running branch unless forced

#### FR-9 Diff

* support schema-only diff between two branches
* output human-readable diff

#### FR-10 Cleanup / GC

* clean orphan snapshots and expired temp branches
* prune unneeded storage and metadata

### Non-functional requirements

#### NFR-1 Speed

* checkpoint/branch/restore should avoid logical dump/restore in normal path
* fast path should use physical copy-on-write or file-level cloning where available

#### NFR-2 Safety

* operations must use locks
* must not corrupt a running branch
* must not mutate immutable snapshots

#### NFR-3 Predictability

* every state transition should be explicit in metadata
* failures should be resumable or recoverable

#### NFR-4 Portability

* baseline implementation should work on common Docker + local filesystem setups
* high-performance driver can be filesystem-specific

#### NFR-5 Developer ergonomics

* clear CLI naming
* good defaults
* minimal required manual configuration

---

## 7. Architecture overview

PGV consists of the following major components:

1. **CLI**
2. **Optional local daemon**
3. **Metadata store (SQLite)**
4. **Snapshot driver interface**
5. **Runtime manager (Docker/process orchestration)**
6. **Postgres control layer**
7. **Diff engine**
8. **Garbage collector**

### Architecture principles

* immutable snapshots, mutable branches
* one running instance per branch
* branch state is always backed by a real data directory
* metadata is the source of truth for orchestration
* never edit physical snapshot contents after creation

---

## 8. High-level system design

### 8.1 Repo model

A single repo manages one Postgres cluster workspace.

A repo has:

* a config file
* a SQLite metadata database
* directories for snapshots and branches
* WAL/archive storage if needed
* local runtime state (locks, sockets, logs)

### 8.2 State model

The lifecycle is:

* initialize repo
* create snapshots from a branch
* create branches from snapshots
* start/stop branch instances
* restore branch contents to older snapshot
* delete or garbage-collect temporary state

### 8.3 Storage model

Use two core storage abstractions:

* **Snapshot storage**: immutable data directories
* **Branch storage**: writable data directories

Snapshots can be created by:

* copy-on-write filesystem clone/snapshot
* physical base-backup copy
* optional file-copy fallback

Branches can be created by:

* cloning a snapshot into a writable directory

---

## 9. Component design

## 9.1 CLI

### Responsibilities

* parse user commands
* call daemon or direct service layer
* render human-readable status
* expose scripting-friendly flags/output

### Command groups

```bash
pgv init
pgv checkpoint <message>
pgv tag <snapshot> <tag>
pgv branch <source> <branch-name>
pgv start <branch>
pgv stop <branch>
pgv checkout <branch>
pgv restore <snapshot> [--branch <name>]
pgv list
pgv status
pgv diff <left> <right>
pgv delete branch <name>
pgv gc
pgv url [branch]
pgv safe-migrate -- <command...>
```

### Output modes

* default human-readable
* `--json` for automation

---

## 9.2 Optional daemon

### Why a daemon exists

The daemon simplifies:

* long-running operations
* branch start/stop orchestration
* concurrent CLI access
* lock management
* local status socket/API

### v1 recommendation

* daemon optional
* CLI can operate directly in-process for v1
* introduce daemon in v2 if needed

### If implemented

* communication over Unix domain socket
* request/response JSON API
* single daemon per repo or machine

---

## 9.3 Metadata store

Use SQLite as local state database.

### Tables

#### repos

* `id`
* `name`
* `root_path`
* `postgres_image`
* `postgres_version`
* `snapshot_driver`
* `active_branch_id`
* `created_at`
* `updated_at`

#### branches

* `id`
* `repo_id`
* `name`
* `base_snapshot_id`
* `head_snapshot_id`
* `data_path`
* `status` (`stopped`, `running`, `broken`)
* `port`
* `is_head`
* `ttl_expires_at`
* `created_at`
* `updated_at`

#### snapshots

* `id`
* `repo_id`
* `parent_snapshot_id`
* `source_branch_id`
* `label`
* `kind` (`checkpoint`, `import`, `base`, `auto`)
* `data_path`
* `driver_type`
* `restore_point_name`
* `lsn`
* `size_bytes`
* `created_at`

#### tags

* `id`
* `repo_id`
* `snapshot_id`
* `name`
* `created_at`

#### instances

* `id`
* `repo_id`
* `branch_id`
* `runtime_type` (`docker`, `local`)
* `container_name`
* `port`
* `status`
* `pid`
* `started_at`
* `stopped_at`

#### operations

* `id`
* `repo_id`
* `type`
* `status`
* `payload_json`
* `error_text`
* `started_at`
* `finished_at`

#### gc_marks

* `id`
* `repo_id`
* `object_type`
* `object_id`
* `reason`
* `created_at`

### Metadata invariants

* branch names unique per repo
* tag names unique per repo
* snapshots immutable once committed
* active branch must exist
* running instance belongs to exactly one branch

---

## 9.4 Snapshot driver interface

This is the core abstraction.

### Interface

```go
type SnapshotDriver interface {
    Name() string
    CreateSnapshot(ctx context.Context, req CreateSnapshotRequest) (CreateSnapshotResult, error)
    CloneSnapshotToBranch(ctx context.Context, req CloneSnapshotRequest) (CloneSnapshotResult, error)
    DeleteSnapshot(ctx context.Context, req DeleteSnapshotRequest) error
    DeleteBranchData(ctx context.Context, req DeleteBranchDataRequest) error
    StatObject(ctx context.Context, path string) (ObjectStats, error)
    Validate(ctx context.Context, req ValidateDriverRequest) error
}
```

### Driver implementations

#### Driver A — `cowfs`

Preferred fast path.

Use when host filesystem supports efficient copy-on-write clone/snapshot semantics.

Characteristics:

* fastest branch creation
* low storage amplification until pages change
* ideal local UX

#### Driver B — `basebackup`

Portable fallback.

Characteristics:

* slower than CoW
* still avoids logical dump/restore
* robust baseline implementation

#### Driver C — `copydir`

Last-resort fallback.

Characteristics:

* simple full file copy
* slower
* mainly useful as safety fallback or for initial prototype

### Driver selection

At `pgv init`:

1. detect whether configured PGDATA path sits on supported CoW filesystem
2. if yes, choose `cowfs`
3. else choose `basebackup`
4. if neither available, allow `copydir` in explicit degraded mode

---

## 9.5 Runtime manager

Responsible for starting/stopping Postgres per branch.

### Supported runtime types

#### Docker runtime (first-class)

Most important initial target.

Responsibilities:

* create named container per branch
* mount branch data path as PGDATA
* expose unique local port
* pass environment/config
* run health checks

#### Local process runtime (optional later)

Allow running local `postgres` directly for non-Docker users.

### Runtime interface

```go
type RuntimeManager interface {
    StartBranch(ctx context.Context, req StartBranchRequest) (StartBranchResult, error)
    StopBranch(ctx context.Context, req StopBranchRequest) error
    RemoveBranchRuntime(ctx context.Context, req RemoveBranchRuntimeRequest) error
    Status(ctx context.Context, req RuntimeStatusRequest) (RuntimeStatusResult, error)
}
```

### Runtime rules

* one active Postgres process/container per branch data dir
* never mount one writable branch data dir into multiple live containers
* never swap PGDATA under a live process

---

## 9.6 Postgres control layer

This component manages direct DB interactions.

### Responsibilities

* connect to running branch
* create named restore points
* force optional checkpoints
* run simple validation queries
* get version and settings
* support schema dump for diffing

### Main operations

* `Ping`
* `CreateRestorePoint(name)`
* `Checkpoint()`
* `GetServerVersion()`
* `GetCurrentLSN()`
* `DumpSchema()`

### Migration wrapper support

For `pgv safe-migrate -- <cmd>`:

1. create auto-checkpoint
2. run external command
3. keep branch running
4. optionally create post-migration snapshot on success

---

## 9.7 Diff engine

### v1 scope

Schema-only diff.

### Flow

1. connect to both branches
2. obtain schema-only dumps
3. normalize volatile output
4. text diff normalized dumps
5. render readable output

### Why not data diff in v1

* expensive
* hard to present well
* misleading to market as generic Git merge/diff for rows

---

## 9.8 Garbage collector

### Responsibilities

* delete expired temp branches
* delete branch data after branch removal
* prune unreachable snapshots if allowed by retention policy
* clean stale runtime metadata
* remove old logs and lockfiles

### GC policies

* snapshots referenced by a branch or tag are protected
* auto snapshots can be deleted after configured retention window
* temp branches may expire via TTL

---

## 10. Filesystem layout

```text
.project/
  .pgv/
    config.json
    meta/
      state.db
    storage/
      branches/
        main/
          PGDATA/
        feature-a/
          PGDATA/
      snapshots/
        snap_0001/
        snap_0002/
    wal/
      archive/
    run/
      locks/
      daemon.sock
    logs/
      pgv.log
```

### Notes

* `.pgv` lives in the project root by default
* branch data paths and snapshots should be controlled exclusively by PGV
* logs and temporary runtime artifacts must be isolated from actual PGDATA

---

## 11. Config format

Example `config.json`:

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

### Config rules

* config should be editable but validated
* generated defaults should work out of the box
* secrets should support env-variable override

---

## 12. Detailed workflows

## 12.1 Repo initialization

### Input

* project directory
* optional Docker image and DB credentials

### Steps

1. create `.pgv` layout
2. create SQLite metadata DB
3. validate runtime availability
4. validate Postgres image/version
5. choose snapshot driver
6. create default branch metadata (`main`)
7. optionally initialize managed Postgres instance

### Result

A fully managed repo with `main` branch and configured storage/runtime.

---

## 12.2 Checkpoint creation

Command:

```bash
pgv checkpoint "before add users table"
```

### Steps

1. resolve current head branch
2. ensure branch runtime is healthy
3. acquire repo and branch lock
4. create restore point with normalized name
5. optionally trigger checkpoint for stronger snapshot boundary
6. ask driver to materialize immutable snapshot
7. persist snapshot metadata
8. release lock

### Output

* snapshot ID
* optional tag/label
* size estimate

### Notes

* snapshots are immutable
* checkpoint names should be slugged and made unique

---

## 12.3 Branch creation

Command:

```bash
pgv branch before-add-users migration-test
```

### Steps

1. resolve source snapshot or tag
2. validate target branch name
3. acquire repo lock
4. driver clones snapshot into writable branch data path
5. create branch metadata
6. optionally auto-start branch
7. release lock

### Output

* branch name
* source snapshot
* connection details if started

---

## 12.4 Start branch

Command:

```bash
pgv start migration-test
```

### Steps

1. resolve branch
2. ensure no runtime already active
3. allocate free port
4. create runtime/container name
5. start Docker container with branch data path
6. wait for health check success
7. persist instance metadata

### Example runtime mapping

* branch `main` -> port 5540
* branch `migration-test` -> port 5541

---

## 12.5 Checkout branch

Command:

```bash
pgv checkout migration-test
```

### Meaning

Checkout only changes which branch is considered HEAD by PGV.

### Steps

1. resolve target branch
2. update metadata to set target as active branch
3. optionally print branch URL
4. optionally update `.env.local` helper file if configured

### Important note

Checkout does not need to mutate files or copy state by itself.
It only changes the active branch pointer.

---

## 12.6 Restore branch to snapshot

Command:

```bash
pgv restore before-add-users --branch main
```

### Steps

1. resolve target branch and target snapshot
2. acquire branch lock
3. stop runtime if running
4. delete existing writable branch data dir
5. clone snapshot into fresh writable branch dir
6. update branch head snapshot metadata
7. restart runtime if it was previously running
8. release lock

### Result

Branch identity remains the same, but its contents now match the selected snapshot.

---

## 12.7 Delete branch

Command:

```bash
pgv delete branch migration-test
```

### Steps

1. resolve branch
2. refuse if branch is active unless `--force`
3. stop runtime if running
4. delete runtime/container
5. delete writable branch data dir
6. delete branch metadata
7. mark now-unreferenced snapshots for GC if eligible

---

## 12.8 Safe migration wrapper

Command:

```bash
pgv safe-migrate -- drizzle-kit push
```

### Steps

1. detect current head branch
2. create auto-checkpoint with timestamp
3. run external command
4. stream stdout/stderr
5. on success optionally create post-migration checkpoint
6. on failure keep checkpoint available for immediate restore

### Why useful

This gives developers a single safe entrypoint around risky schema changes.

---

## 13. Concurrency and locking

Locking is mandatory.

### Lock types

* repo lock
* branch lock
* runtime lock
* GC lock

### Rules

* only one mutating repo operation at a time unless explicitly safe
* branch restore/start/stop operations require branch lock
* GC must not run against active mutation

### Lock implementation

* OS file locks in `.pgv/run/locks`
* metadata record of active operation for recovery/debugging

---

## 14. Failure handling

### Failure classes

#### Snapshot creation failure

* partial snapshot data may exist
* mark operation failed
* clean partial path on next GC or immediate rollback

#### Runtime start failure

* container may exist but be unhealthy
* remove failed runtime and mark branch stopped/broken

#### Restore interruption

* branch may be temporarily without valid writable dir
* restore must use atomic temp-path swap when possible

#### Metadata write failure

* operation should use transaction boundaries
* never commit metadata that points to nonexistent paths if avoidable

### Recovery design

* all mutating operations write operation record first
* on next CLI start, unfinished operations can be detected and either finalized or rolled back

---

## 15. Safety rules

These are strict invariants that implementation must preserve.

1. snapshots are immutable
2. a writable branch data path must not be mounted by more than one running instance
3. branch restore must stop the branch first
4. snapshot deletion must fail if referenced by any branch or tag
5. Postgres major version mismatch must be blocked
6. destructive operations must require explicit confirmation or `--force`
7. PGV never edits files inside a running PGDATA unless the operation is known-safe and controlled

---

## 16. Postgres-specific implementation notes

### Why physical state is the right abstraction

The product should treat Postgres state physically, not logically.
That means it should version:

* data files
* WAL-related restore points / recovery boundaries
* cluster-level physical state

### Practical implementation guidance

* prefer dedicated managed PGDATA for each branch
* do not rely on SQL dump/restore for normal branch switching
* branch operations should produce complete valid data directories
* create restore points before checkpoints/snapshots when possible

### Docker guidance

* mount the branch PGDATA path directly
* use predictable container names and labels
* isolate each branch in its own container
* expose unique host port per branch

### Version guidance

* one repo should target a single Postgres major version
* upgrading Postgres version is a separate workflow, not a normal branch operation

---

## 17. API/service layer design

Even if v1 is CLI-only, internal code should use service objects.

Suggested services:

* `RepoService`
* `SnapshotService`
* `BranchService`
* `RuntimeService`
* `DiffService`
* `GCService`
* `ConfigService`

### Example responsibilities

#### RepoService

* init repo
* load config
* open metadata DB

#### SnapshotService

* create snapshot
* tag snapshot
* list snapshots
* validate ancestry

#### BranchService

* create branch
* restore branch
* checkout branch
* delete branch

#### RuntimeService

* start/stop/status
* port allocation
* container naming

---

## 18. Suggested Go project structure

```text
cmd/
  pgv/
    main.go
internal/
  app/
    init.go
    root.go
    checkpoint.go
    branch.go
    start.go
    stop.go
    restore.go
    diff.go
    gc.go
  config/
    config.go
    validate.go
  metadata/
    db.go
    models.go
    migrations.go
    queries.go
  services/
    repo_service.go
    snapshot_service.go
    branch_service.go
    runtime_service.go
    diff_service.go
    gc_service.go
  runtime/
    docker/
      manager.go
      health.go
  snapshot/
    driver.go
    cowfs/
      driver.go
    basebackup/
      driver.go
    copydir/
      driver.go
  postgres/
    client.go
    restore_points.go
    schema_dump.go
    version.go
  locks/
    locks.go
  ops/
    operations.go
  util/
    fs.go
    ports.go
    names.go
    time.go
pkg/
  api/
```

### Tech choices

* Go
* Cobra for CLI
* pgx for Postgres
* SQLite driver for metadata
* Docker SDK or shell wrapper for runtime

---

## 19. CLI UX details

### Naming style

Use short, memorable verbs.

Examples:

```bash
pgv checkpoint "before delete"
pgv branch before-delete test-delete
pgv start test-delete
pgv url test-delete
pgv restore before-delete --branch main
```

### Helpful aliases

* `pgv save` -> alias for checkpoint
* `pgv ls` -> alias for list
* `pgv rm branch` -> alias for delete branch

### Human output examples

```text
Active branch: main
Running branches:
  main            port 5540   healthy
  migration-test  port 5541   healthy

Snapshots:
  snap_0004  before add users   main   2m ago
  snap_0003  seeded             main   15m ago
```

---

## 20. Acceptance criteria for MVP

The MVP is successful if all of the following are true:

1. A developer can initialize PGV in a local project.
2. They can create a checkpoint from `main`.
3. They can branch from that checkpoint.
4. They can start the new branch in Docker on a separate port.
5. They can mutate schema/data on the branch.
6. They can restore `main` to a previous snapshot.
7. Restoring a snapshot is materially faster than re-importing a large logical dump in common local workflows.
8. Schema diff between two branches works.
9. PGV prevents unsafe deletion/restore of running state.

---

## 21. Phased implementation plan

## Phase 1 — minimal valuable product

### Scope

* init repo
* metadata storage
* Docker runtime manager
* `copydir` or `basebackup` driver
* checkpoint
* branch
* start/stop
* restore
* list/status

### Goal

Get an end-to-end working prototype.

---

## Phase 2 — strong practical version

### Scope

* `cowfs` driver implemented for fast clones on Linux/Windows
* schema diff
* tags
* safe-migrate wrapper
* better health checks
* operation recovery improvements
* TTL temp branches

### Goal

Make the tool pleasant and fast for daily local use.

---

## Phase 3 — polish

### Scope

* optional daemon
* TUI or local web UI
* richer inspect/status output
* editor/db-client helpers
* branch notes
* better GC and retention controls

### Goal

Turn prototype into polished developer tool.

---

## 22. Testing strategy

### Unit tests

* metadata operations
* name resolution
* lock behavior
* config validation
* branch ancestry logic
* retention logic

### Integration tests

* init repo end-to-end
* snapshot creation
* branch start/stop in Docker
* restore branch from snapshot
* schema diff on two branches
* failed runtime startup cleanup
* interrupted operation recovery

### Performance tests

* measure checkpoint/restore times on representative dataset sizes
* compare against `pg_dump`/`pg_restore` baseline

### Compatibility tests

* supported Postgres versions
* Docker runtime on Linux/macOS
* degraded driver path when CoW unavailable

---

## 23. Open questions

These are valid follow-up design questions but should not block MVP.

1. Should the first implementation be direct CLI only, or CLI + daemon from the start?
2. Should basebackup support require WAL archive in v1, or should v1 initially use simpler copydir fallback?
3. Should PGV manage the original developer container, or create its own managed containers only?
4. How aggressively should GC prune auto-snapshots by default?
5. Should checkout optionally rewrite project `.env.local` for active DB URL?

---

## 24. Recommended implementation choices

If an AI/code agent is implementing this, the recommended choices are:

* language: **Go**
* metadata: **SQLite**
* runtime: **Docker first**
* v1 driver: **basebackup or copydir**, whichever is simpler and more reliable to get working end-to-end
* v2 driver: **cowfs** for fast clones
* diff: **schema-only**
* branch model: **one writable data dir + one Postgres instance per branch**

---

## 25. Explicit build instructions for the implementing AI

Build PGV as a local CLI application in Go.

Implementation order:

1. set up CLI scaffolding and config loading
2. implement SQLite metadata schema and migrations
3. implement repository initialization
4. implement Docker runtime manager
5. implement simplest working snapshot driver
6. implement checkpoint/branch/restore/list commands
7. add status and URL output
8. add schema diff
9. add locks and recovery hardening
10. add advanced driver(s) later

Important constraints:

* do not implement logical dump/restore as the normal branch/restore path
* do not attempt branch merge for data
* do not mutate a running branch’s data directory
* keep snapshots immutable
* make all commands safe and idempotent where possible

Definition of done for first usable release:

* a user can checkpoint current local Postgres state, create branch, start branch, test migration, and restore old state with data faster than their current dump/restore workflow

---

## 26. Summary

PGV should be implemented as a **local Postgres physical-state versioning tool** with:

* immutable snapshots
* writable branches
* one Postgres runtime per branch
* fast restore and rollback
* Docker-first workflow
* clear CLI and safe operations

That is the correct abstraction for solving the real user problem: **go back to an earlier Postgres state with data, quickly, during local development.**
