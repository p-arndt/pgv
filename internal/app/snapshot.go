package app

import (
	"context"
	"fmt"

	"pgv/internal/services"

	"github.com/spf13/cobra"
)

var snapshotDelete bool
var snapshotForce bool

func commandBoolFlag(cmd *cobra.Command, name string) (bool, error) {
	flag := cmd.Flags().Lookup(name)
	if flag == nil || !flag.Changed {
		return false, nil
	}
	return cmd.Flags().GetBool(name)
}

var snapshotCmd = &cobra.Command{
	Use:     "snapshot <snapshot-id-or-label>",
	Aliases: []string{"snap"},
	Short:   "Manage snapshots",
	Long: `Manage snapshots.

Use -d to delete a snapshot. Use --force with -d to delete tagged snapshots
or snapshots that have child snapshots.`,
	Args: func(cmd *cobra.Command, args []string) error {
		deleteMode, err := commandBoolFlag(cmd, "delete")
		if err != nil {
			return err
		}
		forceMode, err := commandBoolFlag(cmd, "force")
		if err != nil {
			return err
		}
		if forceMode && !deleteMode {
			return fmt.Errorf("--force requires --delete (-d)")
		}
		if deleteMode {
			return cobra.ExactArgs(1)(cmd, args)
		}
		return fmt.Errorf("no action specified; use -d to delete a snapshot")
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return getSnapshotsForCompletion()
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		deleteMode, err := commandBoolFlag(cmd, "delete")
		if err != nil {
			return err
		}
		forceMode, err := commandBoolFlag(cmd, "force")
		if err != nil {
			return err
		}
		if !deleteMode {
			return fmt.Errorf("no action specified; use -d to delete a snapshot")
		}

		snapshotRef := args[0]

		_, db, repo, lock, err := getRepoContext()
		if err != nil {
			return err
		}
		defer lock.Unlock()
		defer db.Close()

		snapSvc, err := services.NewSnapshotService(db, repo.SnapshotDriver)
		if err != nil {
			return err
		}

		fmt.Printf("Deleting snapshot '%s'...\n", snapshotRef)
		if err := snapSvc.DeleteSnapshot(context.Background(), repo.ID, snapshotRef, forceMode); err != nil {
			return err
		}

		fmt.Printf("Snapshot deleted successfully: %s\n", snapshotRef)
		return nil
	},
}

func init() {
	snapshotCmd.Flags().BoolVarP(&snapshotDelete, "delete", "d", false, "Delete a snapshot")
	snapshotCmd.Flags().BoolVar(&snapshotForce, "force", false, "Force delete (use with -d)")
	rootCmd.AddCommand(snapshotCmd)
}
