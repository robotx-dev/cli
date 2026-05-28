package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/robotx-dev/cli/pkg/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var versionsCmd = &cobra.Command{
	Use:     "versions",
	Aliases: []string{"builds"},
	Short:   "List recent build versions for a project",
	Long:    `List recent build versions for a project, useful for multi-version management and selecting a build to publish.`,
	RunE:    runVersions,
}

var (
	versionsProjectID string
	versionsLimit     int
)

type versionsResponse struct {
	ProjectID string          `json:"project_id"`
	Limit     int             `json:"limit"`
	Builds    []*client.Build `json:"builds"`
}

func init() {
	rootCmd.AddCommand(versionsCmd)
	versionsCmd.Flags().StringVarP(&versionsProjectID, "project-id", "p", "", "Project ID (required)")
	versionsCmd.Flags().IntVar(&versionsLimit, "limit", 20, "Number of recent versions to list (max 100 on server)")
	versionsCmd.MarkFlagRequired("project-id")
}

func runVersions(cmd *cobra.Command, args []string) error {
	baseURL := viper.GetString("base_url")
	apiKey := viper.GetString("api_key")

	if baseURL == "" {
		return newCLIError("missing_base_url", "base URL is required", 1, nil)
	}
	if apiKey == "" {
		return newCLIError("missing_api_key", "API key is required", 1, nil)
	}

	c := client.NewClient(baseURL, apiKey)
	logf("📋 Listing recent versions for project: %s\n", versionsProjectID)
	builds, err := c.ListBuildsForProject(versionsProjectID, versionsLimit)
	if err != nil {
		return newCLIError("api_error", "failed to list project versions", 2, err)
	}

	resp := versionsResponse{
		ProjectID: versionsProjectID,
		Limit:     versionsLimit,
		Builds:    builds,
	}
	if err := emitSuccess(cmd.Name(), resp); err != nil {
		return newCLIError("output_error", "failed to render JSON output", 1, err)
	}
	if isJSONOutput() {
		return nil
	}

	if len(builds) == 0 {
		fmt.Fprintln(os.Stdout, "No build versions found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "BUILD_ID\tSEQ\tLABEL\tSOURCE_REF\tSTATUS\tCOMMIT_ID\tCREATED_AT\tFINISHED_AT")
	for _, b := range builds {
		fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			b.BuildID,
			formatBuildVersionSeq(b.VersionSeq),
			valueOrDash(b.VersionLabel),
			valueOrDash(b.SourceRef),
			b.Status,
			b.CommitID,
			formatBuildTime(b.CreatedAt),
			formatBuildTimePtr(b.FinishedAt),
		)
	}
	_ = w.Flush()

	return nil
}

func formatBuildTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Format("2006-01-02 15:04:05")
}

func formatBuildTimePtr(value *time.Time) string {
	if value == nil || value.IsZero() {
		return "-"
	}
	return value.Format("2006-01-02 15:04:05")
}
