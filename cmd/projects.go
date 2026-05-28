package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/robotx-dev/cli/pkg/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List projects",
	Long:  `List projects for the current account.`,
	Args:  cobra.NoArgs,
	RunE:  runProjects,
}

var projectsDeleteCmd = &cobra.Command{
	Use:     "delete [project-id]",
	Aliases: []string{"remove", "rm"},
	Short:   "Delete a project",
	Long:    `Delete a remote RobotX project. This is a destructive operation and requires --yes.`,
	Args:    cobra.MaximumNArgs(1),
	RunE:    runProjectsDelete,
}

var (
	projectsLimit           int
	projectsDeleteProjectID string
	projectsDeleteYes       bool
)

type projectsResponse struct {
	Limit    int               `json:"limit,omitempty"`
	Projects []*client.Project `json:"projects"`
}

type projectDeleteResponse struct {
	ProjectID string `json:"project_id"`
	Deleted   bool   `json:"deleted"`
}

func init() {
	rootCmd.AddCommand(projectsCmd)
	projectsCmd.AddCommand(projectsDeleteCmd)

	projectsCmd.Flags().IntVar(&projectsLimit, "limit", 50, "Number of projects to list (max enforced by server)")
	projectsDeleteCmd.Flags().StringVarP(&projectsDeleteProjectID, "project-id", "p", "", "Project ID to delete")
	projectsDeleteCmd.Flags().BoolVarP(&projectsDeleteYes, "yes", "y", false, "Confirm deleting the remote RobotX project")
}

func runProjects(cmd *cobra.Command, args []string) error {
	baseURL := viper.GetString("base_url")
	apiKey := viper.GetString("api_key")

	if baseURL == "" {
		return newCLIError("missing_base_url", "base URL is required", 1, nil)
	}
	if apiKey == "" {
		return newCLIError("missing_api_key", "API key is required", 1, nil)
	}

	c := client.NewClient(baseURL, apiKey)
	logf("📋 Listing projects...\n")
	projects, err := c.ListProjects(projectsLimit)
	if err != nil {
		return newCLIError("api_error", "failed to list projects", 2, err)
	}

	resp := projectsResponse{
		Limit:    projectsLimit,
		Projects: projects,
	}
	if err := emitSuccess(cmd.Name(), resp); err != nil {
		return newCLIError("output_error", "failed to render JSON output", 1, err)
	}
	if isJSONOutput() {
		return nil
	}

	if len(projects) == 0 {
		fmt.Fprintln(os.Stdout, "No projects found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROJECT_ID\tNAME\tVISIBILITY\tCREATED_AT\tUPDATED_AT\tPREVIEW_URL\tPRODUCTION_URL")
	for _, project := range projects {
		fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			project.ProjectID,
			valueOrDash(project.Name),
			valueOrDash(project.Visibility),
			formatBuildTime(project.CreatedAt),
			formatBuildTime(project.UpdatedAt),
			valueOrDash(projectPreviewURL(project, baseURL)),
			valueOrDash(resolvePublishURL(baseURL, project)),
		)
	}
	_ = w.Flush()

	return nil
}

func runProjectsDelete(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProjectDeleteID(args, projectsDeleteProjectID)
	if err != nil {
		return err
	}
	if !projectsDeleteYes {
		return newCLIError("confirmation_required", "deleting a project requires --yes", 1, nil)
	}

	c, err := newConfiguredClient()
	if err != nil {
		return err
	}

	logf("🗑️  Deleting project: %s\n", projectID)
	if err := c.DeleteProject(projectID); err != nil {
		return newCLIError("api_error", "failed to delete project", 2, err)
	}

	resp := projectDeleteResponse{
		ProjectID: projectID,
		Deleted:   true,
	}
	if err := emitSuccess(cmd.Name(), resp); err != nil {
		return newCLIError("output_error", "failed to render JSON output", 1, err)
	}
	if isJSONOutput() {
		return nil
	}
	fmt.Fprintf(os.Stdout, "Deleted project: %s\n", projectID)
	return nil
}

func resolveProjectDeleteID(args []string, flagProjectID string) (string, error) {
	argProjectID := ""
	if len(args) > 0 {
		argProjectID = strings.TrimSpace(args[0])
	}
	flagProjectID = strings.TrimSpace(flagProjectID)
	if argProjectID != "" && flagProjectID != "" && argProjectID != flagProjectID {
		return "", newCLIError("invalid_project_id", "project ID positional argument and --project-id do not match", 1, nil)
	}
	if argProjectID == "" {
		argProjectID = flagProjectID
	}
	if argProjectID == "" {
		return "", newCLIError("missing_project_id", "project ID is required", 1, nil)
	}
	return argProjectID, nil
}
