package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/haibingtown/robotx_cli/pkg/client"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get project or build status",
	Long:  `Get the status of a project or specific build.`,
	RunE:  runStatus,
}

var (
	statusProjectID string
	statusBuildID   string
	showLogs        bool
)

type statusResponse struct {
	Project *client.Project `json:"project,omitempty"`
	Build   *client.Build   `json:"build,omitempty"`
	URLs    *statusURLs     `json:"urls,omitempty"`
}

type statusURLs struct {
	PreviewURL    string `json:"preview_url,omitempty"`
	ProductionURL string `json:"production_url,omitempty"`
}

func init() {
	rootCmd.AddCommand(statusCmd)

	statusCmd.Flags().StringVarP(&statusProjectID, "project-id", "p", "", "Project ID")
	statusCmd.Flags().StringVarP(&statusBuildID, "build-id", "b", "", "Build ID (optional)")
	statusCmd.Flags().BoolVarP(&showLogs, "logs", "l", false, "Deprecated: build logs are no longer available")
}

func runStatus(cmd *cobra.Command, args []string) error {
	if statusProjectID == "" && statusBuildID == "" {
		return newCLIError("missing_argument", "at least one of --project-id or --build-id is required", 1, nil)
	}
	if showLogs {
		return newCLIError("unsupported_feature", "build logs are unavailable because RobotX no longer runs remote builds", 1, nil)
	}

	baseURL := viper.GetString("base_url")
	apiKey := viper.GetString("api_key")

	if baseURL == "" {
		return newCLIError("missing_base_url", "base URL is required", 1, nil)
	}
	if apiKey == "" {
		return newCLIError("missing_api_key", "API key is required", 1, nil)
	}

	c := client.NewClient(baseURL, apiKey)
	resp := statusResponse{}

	if statusProjectID != "" {
		logf("📦 Fetching project information...\n")
		project, err := c.GetProject(statusProjectID)
		if err != nil {
			return newCLIError("api_error", "failed to get project", 2, err)
		}
		resp.Project = project
	}

	if statusBuildID != "" {
		logf("\n🔨 Fetching build information...\n")
		build, err := c.GetBuild(statusProjectID, statusBuildID)
		if err != nil {
			return newCLIError("api_error", "failed to get build", 2, err)
		}
		resp.Build = build

		if resp.Project == nil && build.ProjectID != "" {
			project, err := c.GetProject(build.ProjectID)
			if err == nil {
				resp.Project = project
			}
		}
	}

	urlProjectID := statusProjectID
	if urlProjectID == "" {
		if resp.Project != nil {
			urlProjectID = resp.Project.ProjectID
		} else if resp.Build != nil {
			urlProjectID = resp.Build.ProjectID
		}
	}
	if resp.Project != nil {
		resp.URLs = &statusURLs{
			PreviewURL:    projectPreviewURL(resp.Project, baseURL),
			ProductionURL: resolvePublishURL(baseURL, resp.Project),
		}
	} else if urlProjectID != "" {
		resp.URLs = &statusURLs{
			PreviewURL:    fmt.Sprintf("%s/preview/%s", baseURL, urlProjectID),
			ProductionURL: fmt.Sprintf("%s/%s", baseURL, urlProjectID),
		}
	}

	if err := emitSuccess(cmd.Name(), resp); err != nil {
		return newCLIError("output_error", "failed to render JSON output", 1, err)
	}
	if isJSONOutput() {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if resp.Project != nil {
		fmt.Fprintf(w, "\n📋 Project Information:\n")
		fmt.Fprintf(w, "ID:\t%s\n", resp.Project.ProjectID)
		fmt.Fprintf(w, "Name:\t%s\n", resp.Project.Name)
		fmt.Fprintf(w, "Visibility:\t%s\n", resp.Project.Visibility)
		fmt.Fprintf(w, "Created:\t%s\n", resp.Project.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(w, "Updated:\t%s\n", resp.Project.UpdatedAt.Format("2006-01-02 15:04:05"))
	}
	if resp.Build != nil {
		fmt.Fprintf(w, "\n📋 Build Information:\n")
		fmt.Fprintf(w, "ID:\t%s\n", resp.Build.BuildID)
		fmt.Fprintf(w, "Status:\t%s\n", resp.Build.Status)
		fmt.Fprintf(w, "Version Seq:\t%s\n", formatBuildVersionSeq(resp.Build.VersionSeq))
		fmt.Fprintf(w, "Version Label:\t%s\n", valueOrDash(resp.Build.VersionLabel))
		fmt.Fprintf(w, "Source Ref:\t%s\n", valueOrDash(resp.Build.SourceRef))
		fmt.Fprintf(w, "Commit:\t%s\n", resp.Build.CommitID)
		fmt.Fprintf(w, "Created:\t%s\n", resp.Build.CreatedAt.Format("2006-01-02 15:04:05"))
		if resp.Build.FinishedAt != nil {
			fmt.Fprintf(w, "Finished:\t%s\n", resp.Build.FinishedAt.Format("2006-01-02 15:04:05"))
		}
	}
	w.Flush()
	if resp.URLs != nil {
		fmt.Printf("\n🌐 URLs:\n")
		fmt.Printf("Preview: %s\n", resp.URLs.PreviewURL)
		fmt.Printf("Production: %s\n", resp.URLs.ProductionURL)
	}

	return nil
}

func formatBuildVersionSeq(seq int64) string {
	if seq <= 0 {
		return "-"
	}
	return fmt.Sprintf("%d", seq)
}
