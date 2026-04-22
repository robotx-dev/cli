package cmd

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/haibingtown/robotx_cli/pkg/client"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [project-path]",
	Short: "Deploy a project to RobotX",
	Long: `Deploy a project to RobotX platform. This command will:
1. Resolve project by name (create-or-update)
2. Package and upload source code
3. Build locally in your current workspace
4. Upload build artifacts to the created build
5. Wait for build completion if needed
5. Publish to production by default (use --publish=false to disable)`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDeploy,
}

var (
	projectName  string
	visibility   string
	publish      bool
	wait         bool
	timeout      int
	localBuild   bool
	installCmd   string
	buildCmd     string
	outputDir    string
	versionLabel string
	sourceRef    string
)

var projectNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,61}[a-z0-9]$`)

type deployResponse struct {
	ProjectID     string `json:"project_id"`
	ProjectName   string `json:"project_name,omitempty"`
	CommitID      string `json:"commit_id,omitempty"`
	BuildID       string `json:"build_id,omitempty"`
	VersionSeq    int64  `json:"version_seq,omitempty"`
	VersionLabel  string `json:"version_label,omitempty"`
	SourceRef     string `json:"source_ref,omitempty"`
	BuildStatus   string `json:"build_status,omitempty"`
	PreviewURL    string `json:"preview_url,omitempty"`
	ProductionURL string `json:"production_url,omitempty"`
	Published     bool   `json:"published"`
	Waited        bool   `json:"waited"`
	LocalBuild    bool   `json:"local_build"`
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVarP(&projectName, "name", "n", "", "Project name (create-or-update for current owner)")
	deployCmd.Flags().StringVarP(&visibility, "visibility", "v", "private", "Project visibility (public/private)")
	deployCmd.Flags().BoolVar(&publish, "publish", true, "Publish to production after successful build")
	deployCmd.Flags().BoolVar(&wait, "wait", true, "Wait for build completion")
	deployCmd.Flags().IntVar(&timeout, "timeout", 600, "Build timeout in seconds")
	deployCmd.Flags().BoolVar(&localBuild, "local-build", true, "Build locally and upload artifacts (must remain true; RobotX cloud build is no longer supported)")
	deployCmd.Flags().StringVar(&installCmd, "install-command", "", "Override install command for local build")
	deployCmd.Flags().StringVar(&buildCmd, "build-command", "", "Override build command for local build")
	deployCmd.Flags().StringVar(&outputDir, "output-dir", "", "Override output directory for local build")
	deployCmd.Flags().StringVar(&versionLabel, "version-label", "", "Optional build version label (e.g. v1.2.3)")
	deployCmd.Flags().StringVar(&sourceRef, "source-ref", "", "Optional source reference (e.g. tag:v1.2.3, branch:main@<sha>)")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}

	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return newCLIError("invalid_project_path", "invalid project path", 1, err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return newCLIError("invalid_project_path", fmt.Sprintf("project path does not exist: %s", absPath), 1, nil)
	}

	baseURL := viper.GetString("base_url")
	apiKey := viper.GetString("api_key")

	if baseURL == "" {
		return newCLIError("missing_base_url", "base URL is required (use --base-url or set ROBOTX_BASE_URL)", 1, nil)
	}
	if apiKey == "" {
		return newCLIError("missing_api_key", "API key is required (use --api-key or set ROBOTX_API_KEY)", 1, nil)
	}
	if !localBuild {
		return newCLIError("unsupported_feature", "RobotX no longer supports remote build; remove --local-build=false and run the build locally", 1, nil)
	}

	c := client.NewClient(baseURL, apiKey)
	usedProjectName := strings.TrimSpace(projectName)
	var previewURL string
	var productionURL string

	if usedProjectName == "" {
		usedProjectName = filepath.Base(absPath)
	}
	usedProjectName = strings.ToLower(strings.TrimSpace(usedProjectName))
	if err := validateProjectName(usedProjectName); err != nil {
		return newCLIError("invalid_project_name", err.Error(), 1, nil)
	}

	version := resolveBuildVersionInput()
	if version != nil {
		logf("🏷️  Build version label: %s\n", valueOrDash(version.VersionLabel))
		logf("🔖 Source ref: %s\n", valueOrDash(version.SourceRef))
	}

	logf("📦 Resolving project by name (create-or-update): %s\n", usedProjectName)
	proj, err := c.CreateProject(client.CreateProjectRequest{
		Name:       usedProjectName,
		Visibility: visibility,
	})
	if err != nil {
		return newCLIError("api_error", "failed to resolve project", 2, err)
	}
	usedProjectName = proj.Name
	logf("✅ Project ready: %s\n", proj.ProjectID)

	logf("📦 Packaging source code from: %s\n", absPath)
	zipPath, err := packageSource(absPath)
	if err != nil {
		return newCLIError("package_failed", "failed to package source", 1, err)
	}
	defer os.Remove(zipPath)

	if stat, statErr := os.Stat(zipPath); statErr == nil {
		sizeMB := float64(stat.Size()) / (1024.0 * 1024.0)
		logf("📏 Source archive size: %.2f MB\n", sizeMB)
	}
	logf("✅ Source packaged: %s\n", zipPath)

	logf("⬆️  Uploading source code...\n")
	commit, build, err := c.UploadSource(proj.ProjectID, zipPath, version)
	if err != nil {
		return newCLIError("api_error", "failed to upload source", 2, err)
	}
	if commit != nil && commit.CommitID != "" {
		logf("✅ Source uploaded: %s\n", commit.CommitID)
	}
	if build != nil && build.BuildID != "" {
		logf("✅ Build created: %s\n", build.BuildID)
	}

	if build == nil || build.BuildID == "" {
		return newCLIError("local_build_unsupported", "server did not return a build ID; local build upload is not supported by this server", 2, nil)
	}
	plan := (*client.BuildPlan)(nil)
	if commit != nil && commit.ScannerResult != nil {
		plan = commit.ScannerResult.BuildPlan
	}
	if err := runLocalBuild(absPath, plan); err != nil {
		return newCLIError("build_failed", "local build failed", 3, err)
	}
	artifactDir := outputDir
	if artifactDir == "" && plan != nil && strings.TrimSpace(plan.OutputDir) != "" {
		artifactDir = strings.TrimSpace(plan.OutputDir)
	}
	if artifactDir == "" {
		artifactDir = "dist"
	}
	artifactPath := filepath.Join(absPath, artifactDir)
	if stat, err := os.Stat(artifactPath); err != nil || !stat.IsDir() {
		return newCLIError("build_failed", fmt.Sprintf("output directory missing: %s", artifactPath), 3, nil)
	}
	logf("📦 Packaging build output from: %s\n", artifactPath)
	artifactZip, err := packageDirectory(artifactPath)
	if err != nil {
		return newCLIError("build_failed", "failed to package build output", 3, err)
	}
	defer os.Remove(artifactZip)
	logf("✅ Build output packaged: %s\n", artifactZip)

	logf("⬆️  Uploading build artifacts...\n")
	build, err = c.UploadBuildArtifacts(build.BuildID, artifactZip)
	if err != nil {
		return newCLIError("api_error", "failed to upload build artifacts", 2, err)
	}
	logf("✅ Build artifacts uploaded\n")

	if wait {
		if build == nil || build.BuildID == "" {
			return newCLIError("build_failed", "no build ID available to wait for completion", 3, nil)
		}
		if build.Status != "success" {
			logf("⏳ Waiting for build to complete (timeout: %ds)...\n", timeout)
			build, err = waitForBuild(c, proj.ProjectID, build.BuildID, timeout)
			if err != nil {
				return newCLIError("build_failed", "build failed", 3, err)
			}
		}

		if build.Status == "success" {
			logf("✅ Local build completed successfully!\n")
			previewURL = resolvePreviewURL(baseURL, proj, build)
			if previewURL != "" {
				logf("🌐 Preview URL: %s\n", previewURL)
			}
		} else {
			logf("❌ Build failed with status: %s\n", build.Status)
			return newCLIError("build_failed", fmt.Sprintf("build failed with status: %s", build.Status), 3, nil)
		}
	} else if build != nil && build.Status == "success" {
		logf("✅ Local build completed successfully!\n")
		previewURL = resolvePreviewURL(baseURL, proj, build)
		if previewURL != "" {
			logf("🌐 Preview URL: %s\n", previewURL)
		}
	}

	if publish && build != nil && build.Status == "success" {
		logf("🚀 Publishing to production...\n")
		publicPath, err := c.PublishBuild(proj.ProjectID, build.BuildID)
		if err != nil {
			return newCLIError("publish_failed", "failed to publish", 4, err)
		}
		logf("✅ Published successfully!\n")

		productionURL = strings.TrimSpace(publicPath)
		if productionURL == "" {
			productionURL = resolvePublishURL(baseURL, proj)
		}
		if productionURL != "" {
			logf("🌐 Production URL: %s\n", productionURL)
		}
	}

	if previewURL == "" && build != nil && build.Status == "success" {
		previewURL = resolvePreviewURL(baseURL, proj, build)
	}
	if productionURL == "" && publish && build != nil && build.Status == "success" {
		productionURL = resolvePublishURL(baseURL, proj)
	}

	if err := emitSuccess(cmd.Name(), deployResponse{
		ProjectID:     proj.ProjectID,
		ProjectName:   usedProjectName,
		CommitID:      safeCommitID(commit),
		BuildID:       safeBuildID(build),
		VersionSeq:    safeBuildVersionSeq(build),
		VersionLabel:  safeBuildVersionLabel(build),
		SourceRef:     safeBuildSourceRef(build, version),
		BuildStatus:   safeBuildStatus(build),
		PreviewURL:    previewURL,
		ProductionURL: productionURL,
		Published:     publish && productionURL != "",
		Waited:        wait,
		LocalBuild:    localBuild,
	}); err != nil {
		return newCLIError("output_error", "failed to render JSON output", 1, err)
	}

	return nil
}

func safeCommitID(commit *client.SourceCommit) string {
	if commit == nil {
		return ""
	}
	return commit.CommitID
}

func safeBuildID(build *client.Build) string {
	if build == nil {
		return ""
	}
	return build.BuildID
}

func safeBuildStatus(build *client.Build) string {
	if build == nil {
		return ""
	}
	return build.Status
}

func safeBuildVersionSeq(build *client.Build) int64 {
	if build == nil {
		return 0
	}
	return build.VersionSeq
}

func safeBuildVersionLabel(build *client.Build) string {
	if build == nil {
		return ""
	}
	return strings.TrimSpace(build.VersionLabel)
}

func safeBuildSourceRef(build *client.Build, requested *client.BuildVersionInput) string {
	if build != nil && strings.TrimSpace(build.SourceRef) != "" {
		return strings.TrimSpace(build.SourceRef)
	}
	if requested == nil {
		return ""
	}
	return strings.TrimSpace(requested.SourceRef)
}

func validateProjectName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("project name is required")
	}
	if !projectNamePattern.MatchString(trimmed) {
		return fmt.Errorf("project name must be 4-63 chars of lowercase letters, digits, or hyphens")
	}
	return nil
}

func resolveBuildVersionInput() *client.BuildVersionInput {
	label := strings.TrimSpace(versionLabel)
	ref := strings.TrimSpace(sourceRef)
	if label == "" && ref == "" {
		return nil
	}
	return &client.BuildVersionInput{
		VersionLabel: label,
		SourceRef:    ref,
	}
}

func valueOrDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func resolvePreviewURL(baseURL string, project *client.Project, build *client.Build) string {
	if build != nil && strings.TrimSpace(build.PreviewPath) != "" {
		return strings.TrimSpace(build.PreviewPath)
	}
	return projectPreviewURL(project, baseURL)
}

func packageSource(projectPath string) (string, error) {
	tmpFile, err := os.CreateTemp("", "robotx-source-*.zip")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	zipWriter := zip.NewWriter(tmpFile)
	defer zipWriter.Close()

	err = filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(projectPath, path)
		if err != nil {
			return err
		}

		if shouldSkip(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(zipFile, file)
		return err
	})

	if err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

func packageDirectory(root string) (string, error) {
	tmpFile, err := os.CreateTemp("", "robotx-artifacts-*.zip")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	zipWriter := zip.NewWriter(tmpFile)
	defer zipWriter.Close()

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(zipFile, file)
		return err
	})

	if err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

func shouldSkip(path string) bool {
	skipDirs := []string{
		"node_modules",
		".git",
		".next",
		"dist",
		"build",
		".DS_Store",
		"__pycache__",
		".venv",
		"venv",
	}

	for _, skip := range skipDirs {
		if strings.HasPrefix(path, skip) || strings.Contains(path, string(filepath.Separator)+skip) {
			return true
		}
	}

	return false
}

func runLocalBuild(projectPath string, plan *client.BuildPlan) error {
	install := strings.TrimSpace(installCmd)
	build := strings.TrimSpace(buildCmd)

	if install == "" && plan != nil && strings.TrimSpace(plan.InstallCommand) != "" {
		install = strings.TrimSpace(plan.InstallCommand)
	}
	if build == "" && plan != nil && strings.TrimSpace(plan.BuildCommand) != "" {
		build = strings.TrimSpace(plan.BuildCommand)
	}

	if install == "" && fileExists(filepath.Join(projectPath, "package.json")) {
		install = "npm install"
	}
	if build == "" && fileExists(filepath.Join(projectPath, "package.json")) {
		build = "npm run build"
	}

	if plan != nil && !plan.NeedsBuild && installCmd == "" && buildCmd == "" {
		install = ""
		build = ""
	}

	if install != "" {
		logf("🛠️  Running %s\n", install)
		if err := runShell(projectPath, install); err != nil {
			return fmt.Errorf("install failed: %w", err)
		}
	}
	if build != "" {
		logf("🛠️  Running %s\n", build)
		if err := runShell(projectPath, build); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}
	}
	return nil
}

func runShell(dir, command string) error {
	cmd := exec.Command("sh", "-lc", command)
	cmd.Dir = dir
	if isJSONOutput() {
		cmd.Stdout = os.Stderr
	} else {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func waitForBuild(c *client.Client, projectID, buildID string, timeoutSec int) (*client.Build, error) {
	start := time.Now()
	timeout := time.Duration(timeoutSec) * time.Second

	for {
		if time.Since(start) > timeout {
			return nil, fmt.Errorf("build timeout after %d seconds", timeoutSec)
		}

		build, err := c.GetBuild(projectID, buildID)
		if err != nil {
			return nil, err
		}

		switch build.Status {
		case "success", "failed":
			return build, nil
		case "queued", "running":
			logf("⏳ Build status: %s (elapsed: %ds)\n", build.Status, int(time.Since(start).Seconds()))
			time.Sleep(5 * time.Second)
		default:
			return nil, fmt.Errorf("unknown build status: %s", build.Status)
		}
	}
}
