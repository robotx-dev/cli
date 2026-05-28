package cmd

import (
	"fmt"
	"strings"

	"github.com/robotx-dev/cli/pkg/client"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish a build to production",
	Long:  `Publish a specific build to the production environment.`,
	RunE:  runPublish,
}

var (
	publishProjectID string
	publishBuildID   string
)

type publishResponse struct {
	ProjectID     string `json:"project_id"`
	BuildID       string `json:"build_id"`
	ProductionURL string `json:"production_url,omitempty"`
}

func init() {
	rootCmd.AddCommand(publishCmd)

	publishCmd.Flags().StringVarP(&publishProjectID, "project-id", "p", "", "Project ID (required)")
	publishCmd.Flags().StringVarP(&publishBuildID, "build-id", "b", "", "Build ID (required)")
	publishCmd.MarkFlagRequired("project-id")
	publishCmd.MarkFlagRequired("build-id")
}

func runPublish(cmd *cobra.Command, args []string) error {
	baseURL := viper.GetString("base_url")
	apiKey := viper.GetString("api_key")

	if baseURL == "" {
		return newCLIError("missing_base_url", "base URL is required", 1, nil)
	}
	if apiKey == "" {
		return newCLIError("missing_api_key", "API key is required", 1, nil)
	}

	c := client.NewClient(baseURL, apiKey)

	logf("🚀 Publishing build %s to production...\n", publishBuildID)
	publicPath, err := c.PublishBuild(publishProjectID, publishBuildID)
	if err != nil {
		return newCLIError("publish_failed", "failed to publish", 4, err)
	}

	logf("✅ Published successfully!\n")
	prodURL := strings.TrimSpace(publicPath)
	if prodURL == "" {
		if project, err := c.GetProject(publishProjectID); err == nil {
			prodURL = resolvePublishURL(baseURL, project)
		}
	}
	if prodURL == "" {
		prodURL = fmt.Sprintf("%s/%s", strings.TrimSuffix(baseURL, "/"), publishProjectID)
	}
	logf("🌐 Production URL: %s\n", prodURL)

	if err := emitSuccess(cmd.Name(), publishResponse{
		ProjectID:     publishProjectID,
		BuildID:       publishBuildID,
		ProductionURL: prodURL,
	}); err != nil {
		return newCLIError("output_error", "failed to render JSON output", 1, err)
	}

	return nil
}
