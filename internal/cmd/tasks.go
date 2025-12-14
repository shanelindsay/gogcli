package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/steipete/gogcli/internal/googleapi"
	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

var newTasksService = googleapi.NewTasks

func newTasksCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "Google Tasks",
	}
	cmd.AddCommand(newTasksListsCmd(flags))
	cmd.AddCommand(newTasksListCmd(flags))
	return cmd
}

func newTasksListsCmd(flags *rootFlags) *cobra.Command {
	var max int64
	var page string

	cmd := &cobra.Command{
		Use:   "lists",
		Short: "List task lists",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}

			svc, err := newTasksService(cmd.Context(), account)
			if err != nil {
				return err
			}

			call := svc.Tasklists.List().MaxResults(max).PageToken(page)
			resp, err := call.Do()
			if err != nil {
				return err
			}

			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(os.Stdout, map[string]any{
					"tasklists":     resp.Items,
					"nextPageToken": resp.NextPageToken,
				})
			}

			if len(resp.Items) == 0 {
				u.Err().Println("No task lists")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tTITLE")
			for _, tl := range resp.Items {
				fmt.Fprintf(tw, "%s\t%s\n", tl.Id, tl.Title)
			}
			_ = tw.Flush()
			if resp.NextPageToken != "" {
				u.Err().Printf("# Next page: --page %s", resp.NextPageToken)
			}
			return nil
		},
	}

	cmd.Flags().Int64Var(&max, "max", 100, "Max results (max allowed: 1000)")
	cmd.Flags().StringVar(&page, "page", "", "Page token")
	return cmd
}

func newTasksListCmd(flags *rootFlags) *cobra.Command {
	var max int64
	var page string
	var showCompleted bool
	var showDeleted bool
	var showHidden bool
	var showAssigned bool
	var dueMin string
	var dueMax string
	var completedMin string
	var completedMax string
	var updatedMin string

	cmd := &cobra.Command{
		Use:   "list <tasklistId>",
		Short: "List tasks in a task list",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}
			tasklistID := strings.TrimSpace(args[0])
			if tasklistID == "" {
				return errors.New("empty tasklistId")
			}

			svc, err := newTasksService(cmd.Context(), account)
			if err != nil {
				return err
			}

			call := svc.Tasks.List(tasklistID).
				MaxResults(max).
				PageToken(page).
				ShowCompleted(showCompleted).
				ShowDeleted(showDeleted).
				ShowHidden(showHidden).
				ShowAssigned(showAssigned)
			if strings.TrimSpace(dueMin) != "" {
				call = call.DueMin(strings.TrimSpace(dueMin))
			}
			if strings.TrimSpace(dueMax) != "" {
				call = call.DueMax(strings.TrimSpace(dueMax))
			}
			if strings.TrimSpace(completedMin) != "" {
				call = call.CompletedMin(strings.TrimSpace(completedMin))
			}
			if strings.TrimSpace(completedMax) != "" {
				call = call.CompletedMax(strings.TrimSpace(completedMax))
			}
			if strings.TrimSpace(updatedMin) != "" {
				call = call.UpdatedMin(strings.TrimSpace(updatedMin))
			}

			resp, err := call.Do()
			if err != nil {
				return err
			}

			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(os.Stdout, map[string]any{
					"tasks":         resp.Items,
					"nextPageToken": resp.NextPageToken,
				})
			}

			if len(resp.Items) == 0 {
				u.Err().Println("No tasks")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tTITLE\tSTATUS\tDUE\tUPDATED")
			for _, t := range resp.Items {
				status := strings.TrimSpace(t.Status)
				if status == "" {
					status = "needsAction"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", t.Id, t.Title, status, strings.TrimSpace(t.Due), strings.TrimSpace(t.Updated))
			}
			_ = tw.Flush()

			if resp.NextPageToken != "" {
				u.Err().Printf("# Next page: --page %s", resp.NextPageToken)
			}
			return nil
		},
	}

	cmd.Flags().Int64Var(&max, "max", 20, "Max results (max allowed: 100)")
	cmd.Flags().StringVar(&page, "page", "", "Page token")

	cmd.Flags().BoolVar(&showCompleted, "show-completed", true, "Include completed tasks (requires --show-hidden for some clients)")
	cmd.Flags().BoolVar(&showDeleted, "show-deleted", false, "Include deleted tasks")
	cmd.Flags().BoolVar(&showHidden, "show-hidden", false, "Include hidden tasks")
	cmd.Flags().BoolVar(&showAssigned, "show-assigned", false, "Include tasks assigned to current user")

	cmd.Flags().StringVar(&dueMin, "due-min", "", "Lower bound for due date filter (RFC3339)")
	cmd.Flags().StringVar(&dueMax, "due-max", "", "Upper bound for due date filter (RFC3339)")
	cmd.Flags().StringVar(&completedMin, "completed-min", "", "Lower bound for completion date filter (RFC3339)")
	cmd.Flags().StringVar(&completedMax, "completed-max", "", "Upper bound for completion date filter (RFC3339)")
	cmd.Flags().StringVar(&updatedMin, "updated-min", "", "Lower bound for updated time filter (RFC3339)")
	return cmd
}

