package cmd

import "github.com/spf13/cobra"

var logsCmd = &cobra.Command{
	Use:   "logs [build-id]",
	Short: "Deprecated: build logs are unavailable",
	Long:  "Deprecated: RobotX no longer provides remote build logs because build execution happens locally.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLogs,
}

var (
	logsProjectID string
	logsBuildID   string
	logsFollow    bool
)

type logsResponse struct {
	ProjectID string `json:"project_id,omitempty"`
	BuildID   string `json:"build_id"`
	Logs      string `json:"logs"`
}

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().StringVarP(&logsProjectID, "project-id", "p", "", "Project ID (optional)")
	logsCmd.Flags().StringVarP(&logsBuildID, "build-id", "b", "", "Build ID")
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow logs in realtime (not implemented yet)")
}

func runLogs(cmd *cobra.Command, args []string) error {
	_ = cmd
	_ = args
	_ = logsProjectID
	_ = logsBuildID
	_ = logsFollow
	return newCLIError("unsupported_feature", "build logs are unavailable because RobotX no longer runs remote builds", 1, nil)
}
