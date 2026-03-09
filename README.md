# PGV — Local Postgres Version Control

PGV is a local developer tool that provides **fast rollback, branching, and restore for PostgreSQL with data included**, without relying on slow logical dump/restore workflows. It operates like Git, but versions your physical database state.

## Why PGV?

* You're running Postgres locally (e.g. in Docker).
* You apply migrations, run destructive tests, or accidentally delete data.
* You want to instantly return to an older database state **including all rows**, without waiting 15 minutes for `pg_dump` and `pg_restore`.
* You want to test a feature on an isolated branch with a copy of your main database.

## Installation

Ensure you have Docker installed and running.

Clone this repository and compile the binary:
```bash
go build -o pgv ./cmd/pgv
mv pgv /usr/local/bin/ # Or anywhere in your PATH
```

---

## Onboarding an Existing Database

You probably already have a database running via Docker Compose or locally. PGV makes it incredibly easy to bring that existing data into its versioned workflow. 

There are two primary ways to do this:

### Option 1: Fast Physical Import (Recommended for local Docker)

If you have an existing physical `PGDATA` directory (e.g., mapped as a volume in `docker-compose.yml`), you can initialize PGV using that data directory directly. This is virtually instantaneous since it bypasses logical dumps.

1. **Stop your existing database container.**
   ```bash
   docker compose stop db
   ```
2. **Initialize PGV pointing to your old data directory.**
   ```bash
   pgv init --from-dir ./path/to/your/db/data
   ```
3. **Start your new PGV-managed database.**
   ```bash
   pgv start main
   ```
   *Your database is now available on port 5540, fully managed by PGV!*

### Option 2: Logical Import (For active connections or remote DBs)

If you prefer keeping your database running, or if you want to pull down data from a remote staging/production environment:

1. **Initialize a fresh PGV repository and start the main branch.**
   ```bash
   pgv init
   pgv start main
   ```
2. **Import data directly via connection string.**
   ```bash
   pgv import postgres://user:password@host:port/dbname
   ```
   *PGV uses Docker to stream the dump directly into your branch, requiring no host binaries!*
3. **Save your new state.**
   ```bash
   pgv checkpoint "initial import"
   ```

---

## Daily Workflow

Once your data is inside PGV, your workflow feels just like Git:

**Create a checkpoint before doing something dangerous:**
```bash
pgv checkpoint "before dropping users table"
```

**Branch off to try a new feature isolated from main:**
```bash
pgv branch "before dropping users table" feature-branch
pgv start feature-branch
```
*Your feature branch will start on a new port (e.g., 5541).*

**Oops, I broke my database! Let's restore it:**
```bash
pgv stop main
pgv restore "before dropping users table"
pgv start main
```

**List all your branches and snapshots:**
```bash
pgv list
```

---

## Architecture

* **Snapshots:** Immutable physical checkpoints of your PGDATA directory.
* **Branches:** Writable clones derived from snapshots.
* **Instances:** PGV spins up a dedicated Docker container for every active branch, mapping a unique host port to each.

Enjoy safe, instant database rollbacks locally!