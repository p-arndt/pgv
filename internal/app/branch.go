package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"pgv/internal/metadata"
	"pgv/internal/services"
)

var branchCmd = &cobra.Command{
	Use:   "branch <new-branch-name> [source]",
	Short: "Create a writable branch from a snapshot or another branch",
	Long: `Create a new writable branch. 
If [source] is omitted, it branches from the active branch.
[source] can be a branch name or a snapshot ID. 
If a branch is specified, an automatic checkpoint is created first.`,
	Args: cobra.RangeArgs(1, 2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 1 {
			// Complete source: branches and snapshots
			comps, _ := getBranchesForCompletion()
			snaps, _ := getSnapshotsForCompletion()
			comps = append(comps, snaps...)
			return comps, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		branchName := args[0]

		cfg, db, repo, lock, err := getRepoContext()
		if err != nil {
			return err
		}
		defer lock.Unlock()
		defer db.Close()

		source := ""
		if len(args) == 2 {
			source = args[1]
		}

		// Determine the source snapshot ID
		var sourceSnapID string

		if source == "" {
			if repo.ActiveBranchID == "" {
				return fmt.Errorf("no source specified and no active branch")
			}
			// Use active branch
			var activeBranch metadata.Branch
			if err := db.Get(&activeBranch, "SELECT * FROM branches WHERE id = ?", repo.ActiveBranchID); err != nil {
				return fmt.Errorf("could not load active branch: %w", err)
			}
			source = activeBranch.Name
		}

		// Check if source is a branch
		var sourceBranch metadata.Branch
		err = db.Get(&sourceBranch, "SELECT * FROM branches WHERE repo_id = ? AND name = ?", repo.ID, source)
		if err == nil {
			// Source is a branch. We need to take a checkpoint.
			snapSvc, err := services.NewSnapshotService(db, repo.SnapshotDriver)
			if err != nil {
				return err
			}
			fmt.Printf("Source is branch '%s'. Creating an auto-checkpoint to branch from...\n", sourceBranch.Name)
			snapID, err := snapSvc.CreateCheckpoint(context.Background(), cfg, repo.ID, sourceBranch.ID, fmt.Sprintf("Auto-checkpoint before branching to %s", branchName))
			if err != nil {
				return fmt.Errorf("failed to create checkpoint of branch '%s': %w", sourceBranch.Name, err)
			}
			sourceSnapID = snapID
		} else {
			// Source might be a snapshot ID
			var snap metadata.Snapshot
			if err := db.Get(&snap, "SELECT * FROM snapshots WHERE repo_id = ? AND id = ?", repo.ID, source); err == nil {
				sourceSnapID = snap.ID
			} else {
				// Try by label (just taking the most recent one if multiple)
				if err := db.Get(&snap, "SELECT * FROM snapshots WHERE repo_id = ? AND label = ? ORDER BY created_at DESC LIMIT 1", repo.ID, source); err == nil {
					sourceSnapID = snap.ID
				} else {
					return fmt.Errorf("could not find branch, snapshot ID, or snapshot label matching '%s'", source)
				}
			}
		}

		branchSvc, err := services.NewBranchService(db, repo.SnapshotDriver)
		if err != nil {
			return err
		}

		fmt.Printf("Creating branch '%s' from snapshot '%s'...\n", branchName, sourceSnapID)
		_, err = branchSvc.CreateBranch(context.Background(), repo.ID, sourceSnapID, branchName)
		if err != nil {
			return err
		}

		fmt.Printf("Branch created successfully: %s\n", branchName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(branchCmd)
}
