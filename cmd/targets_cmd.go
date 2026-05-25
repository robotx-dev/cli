package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var targetsCmd = &cobra.Command{
	Use:   "targets",
	Short: "List local RobotX deploy targets",
	Long:  `List local RobotX deploy targets recorded in .robotx/targets.json.`,
	Args:  cobra.NoArgs,
	RunE:  runTargetsList,
}

var targetsRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a local RobotX deploy target",
	Long:  `Remove a local RobotX deploy target from .robotx/targets.json. This does not delete the remote RobotX project.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTargetsRemove,
}

type targetsResponse struct {
	WorkspaceRoot string          `json:"workspace_root"`
	TargetsFile   string          `json:"targets_file"`
	DefaultTarget string          `json:"default_target,omitempty"`
	Targets       []targetSummary `json:"targets"`
}

type targetSummary struct {
	Name          string `json:"name"`
	Label         string `json:"label,omitempty"`
	Default       bool   `json:"default,omitempty"`
	ProjectID     string `json:"project_id"`
	ProjectName   string `json:"project_name"`
	SourcePath    string `json:"source_path,omitempty"`
	OutputDir     string `json:"output_dir,omitempty"`
	ProductionURL string `json:"production_url,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
}

type targetRemoveResponse struct {
	WorkspaceRoot string          `json:"workspace_root"`
	TargetsFile   string          `json:"targets_file"`
	Removed       targetSummary   `json:"removed"`
	DefaultTarget string          `json:"default_target,omitempty"`
	Targets       []targetSummary `json:"targets"`
	RemoteDeleted bool            `json:"remote_deleted"`
}

func init() {
	rootCmd.AddCommand(targetsCmd)
	targetsCmd.AddCommand(targetsRemoveCmd)

	targetsCmd.PersistentFlags().StringVar(&workspaceRoot, "workspace-root", "", "Override local RobotX target record root")
}

func runTargetsList(cmd *cobra.Command, args []string) error {
	store, err := loadCurrentTargetStore()
	if err != nil {
		return err
	}
	resp := targetsResponse{
		WorkspaceRoot: store.WorkspaceRoot,
		TargetsFile:   store.Path,
		DefaultTarget: store.Data.DefaultTarget,
		Targets:       store.summaries(),
	}
	if err := emitSuccess(cmd.Name(), resp); err != nil {
		return newCLIError("output_error", "failed to render JSON output", 1, err)
	}
	if isJSONOutput() {
		return nil
	}
	printTargetsTable(resp)
	return nil
}

func runTargetsRemove(cmd *cobra.Command, args []string) error {
	store, err := loadCurrentTargetStore()
	if err != nil {
		return err
	}
	removedName := args[0]
	removedEntry, ok := store.remove(removedName)
	if !ok {
		return newCLIErrorWithDetails("target_not_found", fmt.Sprintf("target %q was not found", removedName), 1, map[string]interface{}{
			"targets_file": store.Path,
			"targets":      targetNames(store.Data.Targets),
		}, nil)
	}
	if err := store.save(); err != nil {
		return newCLIError("target_write_failed", "failed to write RobotX target record", 1, err)
	}

	removed := targetSummary{
		Name:          removedName,
		Label:         removedEntry.Label,
		ProjectID:     removedEntry.ProjectID,
		ProjectName:   removedEntry.Name,
		SourcePath:    removedEntry.SourcePath,
		OutputDir:     removedEntry.OutputDir,
		ProductionURL: removedEntry.ProductionURL,
		UpdatedAt:     removedEntry.UpdatedAt,
	}
	resp := targetRemoveResponse{
		WorkspaceRoot: store.WorkspaceRoot,
		TargetsFile:   store.Path,
		Removed:       removed,
		DefaultTarget: store.Data.DefaultTarget,
		Targets:       store.summaries(),
		RemoteDeleted: false,
	}
	if err := emitSuccess(cmd.Name(), resp); err != nil {
		return newCLIError("output_error", "failed to render JSON output", 1, err)
	}
	if isJSONOutput() {
		return nil
	}
	fmt.Fprintf(os.Stdout, "Removed local target %q. Remote RobotX project was not deleted.\n", removedName)
	if resp.DefaultTarget != "" {
		fmt.Fprintf(os.Stdout, "Default target is now %q.\n", resp.DefaultTarget)
	} else if len(resp.Targets) > 0 {
		fmt.Fprintln(os.Stdout, "No default target is set; pass --target when deploying.")
	} else {
		fmt.Fprintln(os.Stdout, "No local RobotX targets remain.")
	}
	return nil
}

func loadCurrentTargetStore() (*targetStore, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, newCLIError("invalid_workspace_root", "failed to read current directory", 1, err)
	}
	root, err := inferWorkspaceRoot(cwd, workspaceRoot)
	if err != nil {
		return nil, newCLIError("invalid_workspace_root", "invalid workspace root", 1, err)
	}
	store, err := loadTargetStore(root)
	if err != nil {
		return nil, newCLIError("target_config_error", "failed to read RobotX targets", 1, err)
	}
	return store, nil
}

func printTargetsTable(resp targetsResponse) {
	if len(resp.Targets) == 0 {
		fmt.Fprintf(os.Stdout, "No local RobotX targets found in %s.\n", resp.TargetsFile)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DEFAULT\tNAME\tLABEL\tPROJECT_NAME\tOUTPUT_DIR\tPRODUCTION_URL\tSOURCE_PATH")
	for _, target := range resp.Targets {
		marker := ""
		if target.Default {
			marker = "*"
		}
		fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			marker,
			valueOrDash(target.Name),
			valueOrDash(target.Label),
			valueOrDash(target.ProjectName),
			valueOrDash(target.OutputDir),
			valueOrDash(target.ProductionURL),
			valueOrDash(target.SourcePath),
		)
	}
	_ = w.Flush()
}
