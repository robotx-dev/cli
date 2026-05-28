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

var accessCmd = &cobra.Command{
	Use:   "access",
	Short: "Manage project access policy",
	Long:  `Manage whether a RobotX project is open, login-only, or private allowlist protected.`,
}

var accessStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show project access policy",
	Args:  cobra.NoArgs,
	RunE:  runAccessStatus,
}

var accessOpenCmd = &cobra.Command{
	Use:   "open",
	Short: "Allow anonymous public access",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAccessSet(cmd, "open")
	},
}

var accessLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Require RobotX platform login",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAccessSet(cmd, "login")
	},
}

var accessPrivateCmd = &cobra.Command{
	Use:   "private",
	Short: "Restrict access to the project allowlist",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAccessSet(cmd, "private")
	},
}

var accessProjectID string

type accessPolicyResponse struct {
	ProjectID string               `json:"project_id"`
	Mode      string               `json:"mode"`
	Policy    *client.AccessPolicy `json:"policy,omitempty"`
	Version   int                  `json:"version,omitempty"`
}

func init() {
	rootCmd.AddCommand(accessCmd)
	accessCmd.AddCommand(accessStatusCmd, accessOpenCmd, accessLoginCmd, accessPrivateCmd)
	accessCmd.PersistentFlags().StringVarP(&accessProjectID, "project-id", "p", "", "Project ID")
	_ = accessCmd.MarkPersistentFlagRequired("project-id")
}

func runAccessStatus(cmd *cobra.Command, args []string) error {
	c, err := newConfiguredClient()
	if err != nil {
		return err
	}
	projectID := strings.TrimSpace(accessProjectID)
	logf("🔐 Fetching access policy...\n")
	policy, err := c.GetAccessPolicy(projectID)
	if err != nil {
		return newCLIError("api_error", "failed to get access policy", 2, err)
	}
	resp := accessPolicyResponse{
		ProjectID: projectID,
		Mode:      accessModeFromPolicy(policy),
		Policy:    policy,
	}
	if policy != nil {
		resp.Version = policy.Version
	}
	if err := emitSuccess(cmd.Name(), resp); err != nil {
		return newCLIError("output_error", "failed to render JSON output", 1, err)
	}
	if isJSONOutput() {
		return nil
	}
	printAccessPolicy(resp)
	return nil
}

func runAccessSet(cmd *cobra.Command, mode string) error {
	c, err := newConfiguredClient()
	if err != nil {
		return err
	}
	input, err := accessPolicyInputForMode(mode)
	if err != nil {
		return err
	}
	projectID := strings.TrimSpace(accessProjectID)
	logf("🔐 Updating access policy to %s...\n", mode)
	version, err := c.UpdateAccessPolicy(projectID, input)
	if err != nil {
		return newCLIError("api_error", "failed to update access policy", 2, err)
	}
	policy, err := c.GetAccessPolicy(projectID)
	if err != nil {
		return newCLIError("api_error", "failed to refresh access policy", 2, err)
	}
	resp := accessPolicyResponse{
		ProjectID: projectID,
		Mode:      accessModeFromPolicy(policy),
		Policy:    policy,
	}
	if version != nil {
		resp.Version = version.Version
	} else if policy != nil {
		resp.Version = policy.Version
	}
	if err := emitSuccess(cmd.Name(), resp); err != nil {
		return newCLIError("output_error", "failed to render JSON output", 1, err)
	}
	if isJSONOutput() {
		return nil
	}
	fmt.Fprintf(os.Stdout, "Access policy updated: %s\n", accessModeLabel(resp.Mode))
	if resp.Version > 0 {
		fmt.Fprintf(os.Stdout, "Version: %d\n", resp.Version)
	}
	return nil
}

func accessPolicyInputForMode(mode string) (client.AccessPolicyInput, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "open":
		return client.AccessPolicyInput{RequirePlatformLogin: false}, nil
	case "login":
		return client.AccessPolicyInput{RequirePlatformLogin: true}, nil
	case "private":
		return client.AccessPolicyInput{
			RequirePlatformLogin: true,
			Credentials:          &client.CredentialInput{Allowlist: true},
		}, nil
	default:
		return client.AccessPolicyInput{}, newCLIError("invalid_access_mode", "invalid access mode (expected open, login, private, or unchanged)", 1, nil)
	}
}

func accessModeFromPolicy(policy *client.AccessPolicy) string {
	if policy == nil {
		return "unknown"
	}
	creds := policy.Credentials
	if creds == nil || (creds.InviteCode == nil && !creds.AllowSignedLink && !creds.Allowlist) {
		if policy.RequirePlatformLogin {
			return "login"
		}
		return "open"
	}
	if policy.RequirePlatformLogin && creds.Allowlist && creds.InviteCode == nil && !creds.AllowSignedLink {
		return "private"
	}
	return "advanced"
}

func accessModeLabel(mode string) string {
	switch mode {
	case "open":
		return "open (anonymous access allowed)"
	case "login":
		return "login (RobotX login required)"
	case "private":
		return "private (allowlist required)"
	case "advanced":
		return "advanced"
	default:
		return "unknown"
	}
}

func printAccessPolicy(resp accessPolicyResponse) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "PROJECT_ID:\t%s\n", resp.ProjectID)
	fmt.Fprintf(w, "MODE:\t%s\n", accessModeLabel(resp.Mode))
	if resp.Version > 0 {
		fmt.Fprintf(w, "VERSION:\t%d\n", resp.Version)
	}
	_ = w.Flush()
}

func newConfiguredClient() (*client.Client, error) {
	baseURL := viper.GetString("base_url")
	apiKey := viper.GetString("api_key")
	if strings.TrimSpace(baseURL) == "" {
		return nil, newCLIError("missing_base_url", "base URL is required", 1, nil)
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, newCLIError("missing_api_key", "API key is required", 1, nil)
	}
	return client.NewClient(baseURL, apiKey), nil
}
