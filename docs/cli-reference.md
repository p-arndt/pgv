# CLI Reference

PGV uses a simple verb-noun command structure.

## Setup & Admin

### `pgv init [repo-name] [--from-dir <path>]`
Initializes a new PGV repository in the current directory.
- `[repo-name]`: Optional name for the repository. Defaults to the current folder name.
- `--from-dir`: Optional path to an existing `PGDATA` directory to instantly import as your `main` branch.

### `pgv list` (or `pgv ls`, `pgv log`)
Lists all branches, their current status (stopped/running), ports, and all available snapshots.

### `pgv url [branch-name]`
Outputs the `postgres://` connection string for the specified branch. If no branch is specified, it outputs the URL for the active branch.

### `pgv status` (or `pgv st`)
Displays the active branch, its status (running/stopped), current port, and the most recent snapshot applied.

## Branch Management

### `pgv start <branch-name>`
Starts an isolated Postgres Docker container for the specified branch.

### `pgv stop <branch-name>`
Stops the Postgres Docker container for the specified branch.

### `pgv checkout <branch-name>` (or `pgv switch <branch-name>`, `pgv co <branch-name>`)
Switches your active working branch (HEAD).

### `pgv branch <new-branch-name> [source]`
Creates a new writable branch isolated from the rest of your database. 
- If `[source]` is omitted, it branches directly from the active branch.
- `[source]` can be a branch name or an immutable snapshot ID. If a branch is specified as the source, PGV automatically creates a checkpoint first to branch from.

### `pgv branch -d <branch-name> [--force]`
Deletes a branch via the Git-style `branch` flow.
- `-d`: Safe delete (fails for active/running branches).
- `--force`: Force delete (explicit, for active/running branches).

### `pgv snapshot -d <snapshot-id-or-label> [--force]` (or `pgv snap -d <snapshot-id-or-label> [--force]`)
Deletes a snapshot.
- `-d`: Safe delete (fails when the snapshot has child snapshots or tags, or if it is referenced by any branch).
- `--force`: Deletes even when tagged or when child snapshots exist (children are re-parented).

## State Management

### `pgv checkpoint <message>` (or `pgv commit <message>`)
Takes an immutable physical snapshot of the active branch. 

> [!NOTE]
> If the branch is running, PGV will automatically stop it, take the physical snapshot, and restart it to prevent data corruption.


### `pgv restore <snapshot-id> [--branch <branch-name>]` (or `pgv reset <snapshot-id> [--branch <branch-name>]`)
Replaces the physical state of the specified branch with the contents of a snapshot.
- `--branch`: Optional. If not provided, restores the currently active branch.

> [!NOTE]
> If the branch is running, PGV will automatically stop it, restore the data, and restart it.

### `pgv rollback`
A quick helper command that automatically restores the active branch to its most recent snapshot (`HEAD`). This is the fastest way to undo a local mistake if you just took a checkpoint.

> [!NOTE]
> If the branch is running, PGV will automatically stop it, roll back the data, and restart it.

### `pgv import <postgres-url>`
Performs a logical `pg_dump` from the provided URL directly into the active, running PGV branch. Uses internal Docker piping so no host binaries are required.

## Shell Autocompletion

PGV provides a `completion` command to generate autocompletion scripts for various shells (Bash, Zsh, Fish, PowerShell). This enables you to press `TAB` to auto-complete commands, flags, branch names, and snapshot IDs.

### Windows (PowerShell)

To enable autocompletion in PowerShell, you need to add the completion script to your PowerShell profile.

1. Open PowerShell.
2. Check if you have a profile script set up:
   ```powershell
   Test-Path $PROFILE
   ```
3. If it returns `False`, create one:
   ```powershell
   New-Item -Type File -Path $PROFILE -Force
   ```
4. Open your profile in Notepad or your preferred editor:
   ```powershell
   notepad $PROFILE
   ```
5. Add the following line to the file, save, and close it:
   ```powershell
   Invoke-Expression (& pgv completion powershell)
   ```
6. Restart your PowerShell session. Now, typing `pgv checkout <TAB>` will suggest your branch names!

### Linux / macOS (Zsh)

If you use Oh My Zsh or standard Zsh, you can add this to your `~/.zshrc`:

```zsh
source <(pgv completion zsh)
```

Restart your terminal or run `source ~/.zshrc` to apply the changes.

### Linux / macOS (Bash)

If you use Bash, you can add this to your `~/.bashrc`:

```bash
source <(pgv completion bash)
```

Restart your terminal or run `source ~/.bashrc` to apply the changes.