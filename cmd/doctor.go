package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/robotx-dev/cli/pkg/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check local RobotX CLI configuration",
	Long:  `Check the RobotX CLI version, PATH, configuration, credentials, and API reachability without changing projects.`,
	Args:  cobra.NoArgs,
	RunE:  runDoctor,
}

type doctorResponse struct {
	Version string        `json:"version"`
	Checks  []doctorCheck `json:"checks"`
}

type doctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	checks := []doctorCheck{
		doctorVersionCheck(),
		doctorPathCheck(),
		doctorConfigCheck(),
	}

	baseURL := strings.TrimSpace(viper.GetString("base_url"))
	apiKey := strings.TrimSpace(viper.GetString("api_key"))
	if baseURL == "" {
		checks = append(checks, doctorCheck{Name: "base_url", Status: "fail", Message: "base URL is missing"})
	} else {
		checks = append(checks, doctorCheck{Name: "base_url", Status: "pass", Message: baseURL})
	}
	if apiKey == "" {
		checks = append(checks, doctorCheck{Name: "api_key", Status: "fail", Message: "API key is missing"})
	} else {
		checks = append(checks, doctorCheck{Name: "api_key", Status: "pass", Message: "API key is configured"})
	}

	if baseURL == "" || apiKey == "" {
		checks = append(checks, doctorCheck{Name: "api_reachability", Status: "skip", Message: "base URL or API key is missing"})
	} else {
		checks = append(checks, doctorAPICheck(baseURL, apiKey))
	}

	resp := doctorResponse{
		Version: version,
		Checks:  checks,
	}
	if hasDoctorFailure(checks) {
		if !isJSONOutput() {
			printDoctor(resp)
		}
		return newCLIErrorWithDetails("doctor_failed", "RobotX doctor found configuration problems", 1, resp, nil)
	}
	if err := emitSuccess(cmd.Name(), resp); err != nil {
		return newCLIError("output_error", "failed to render JSON output", 1, err)
	}
	if isJSONOutput() {
		return nil
	}
	printDoctor(resp)
	return nil
}

func doctorVersionCheck() doctorCheck {
	if strings.TrimSpace(version) == "" {
		return doctorCheck{Name: "version", Status: "warn", Message: "version is empty"}
	}
	return doctorCheck{Name: "version", Status: "pass", Message: version}
}

func doctorPathCheck() doctorCheck {
	path, err := exec.LookPath("robotx")
	if err != nil {
		return doctorCheck{Name: "path", Status: "warn", Message: "robotx is not found on PATH"}
	}
	return doctorCheck{Name: "path", Status: "pass", Message: path}
}

func doctorConfigCheck() doctorCheck {
	configFile := strings.TrimSpace(viper.ConfigFileUsed())
	if configFile == "" {
		return doctorCheck{Name: "config", Status: "warn", Message: "no config file loaded; using flags or environment only"}
	}
	return doctorCheck{Name: "config", Status: "pass", Message: configFile}
}

func doctorAPICheck(baseURL, apiKey string) doctorCheck {
	c := client.NewClient(baseURL, apiKey)
	if _, err := c.ListProjects(1); err != nil {
		return doctorCheck{Name: "api_reachability", Status: "fail", Message: err.Error()}
	}
	return doctorCheck{Name: "api_reachability", Status: "pass", Message: "authenticated API request succeeded"}
}

func hasDoctorFailure(checks []doctorCheck) bool {
	for _, check := range checks {
		if check.Status == "fail" {
			return true
		}
	}
	return false
}

func printDoctor(resp doctorResponse) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CHECK\tSTATUS\tMESSAGE")
	for _, check := range resp.Checks {
		fmt.Fprintf(w, "%s\t%s\t%s\n", check.Name, check.Status, valueOrDash(check.Message))
	}
	_ = w.Flush()
}
