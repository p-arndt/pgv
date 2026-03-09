# Getting Started with PGV

Welcome to PGV! This guide will help you install the CLI, initialize your first repository, and learn the basic workflow.

## 1. Installation

Currently, PGV is distributed as source. You will need Go 1.21+ installed.

```bash
# Clone the repository
git clone https://github.com/your-org/pgv.git
cd pgv

# Build the binary
go build -o pgv ./cmd/pgv

# Move to your PATH
mv pgv /usr/local/bin/
```

Verify the installation:
```bash
pgv --help
```

## 2. Onboarding an Existing Database

You likely already have a Postgres database for your project. PGV can seamlessly import it.

**Option A: Fast Physical Import (If you use Docker Compose locally)**
If you map a local folder to your Postgres container (e.g., `./postgres-data:/var/lib/postgresql/data`), you can import it instantly:

1. Stop your existing docker-compose database.
2. Initialize PGV pointing to that folder:
   ```bash
   pgv init --from-dir ./postgres-data
   ```
3. Start your PGV database:
   ```bash
   pgv start main
   ```

**Option B: Logical Import (From a live connection)**
If you don't have local physical files (e.g., you want to pull staging data):

1. Initialize a blank repo:
   ```bash
   pgv init
   pgv start main
   ```
2. Import the remote data (this will take a few minutes depending on size):
   ```bash
   pgv import postgres://user:pass@remote-host:5432/dbname
   ```
3. Save the imported state immediately:
   ```bash
   pgv checkpoint "initial import"
   ```

## 3. Basic Workflow

Once your database is managed by PGV, your workflow looks like Git.

### Connecting to your DB
Run `pgv url` to get your connection string. It will look something like this:
`postgres://postgres:postgres@127.0.0.1:5540/app?sslmode=disable`
Update your `.env` file to use this URL.

### Creating Checkpoints
Before you run a migration, drop a table, or run a destructive test, create a checkpoint:
```bash
pgv checkpoint "before dropping users table"
```

### Rolling Back
If you broke something, you can restore your database to the last checkpoint in seconds:
```bash
# Rollback to the previous state
pgv rollback
```

If you need to go back further in time, you can find a specific snapshot ID via `pgv list` and use `pgv restore <snapshot-id>`.

### Branching
Want to test a feature without touching your main database?
```bash
# Create a new branch from a snapshot
pgv branch "before dropping users table" feature-x

# Switch to the new branch
pgv checkout feature-x

# Start the feature branch
pgv start feature-x
```

> [!NOTE]
> Your feature branch will be assigned a new port automatically, allowing you to run `main` and `feature-x` simultaneously!

For more detailed command usage, see the [CLI Reference](cli-reference.md).