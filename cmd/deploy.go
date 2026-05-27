package cmd

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
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
	1. Resolve the target project safely
	2. Package and upload source code
	3. Build locally in your current workspace
	4. Upload build artifacts to the created build
	5. Wait for build completion if needed
	6. Publish to production by default (use --publish=false to disable)`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDeploy,
}

var (
	projectName   string
	targetName    string
	projectID     string
	visibility    string
	publish       bool
	wait          bool
	timeout       int
	localBuild    bool
	createMode    bool
	updateMode    bool
	upsertMode    bool
	noWriteTarget bool
	installCmd    string
	buildCmd      string
	outputDir     string
	workspaceRoot string
	versionLabel  string
	sourceRef     string
	deployAccess  string
	verifyURL     bool
)

var projectNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,61}[a-z0-9]$`)

type deployResponse struct {
	ProjectID     string            `json:"project_id"`
	ProjectName   string            `json:"project_name,omitempty"`
	Target        string            `json:"target,omitempty"`
	WorkspaceRoot string            `json:"workspace_root,omitempty"`
	CommitID      string            `json:"commit_id,omitempty"`
	BuildID       string            `json:"build_id,omitempty"`
	VersionSeq    int64             `json:"version_seq,omitempty"`
	VersionLabel  string            `json:"version_label,omitempty"`
	SourceRef     string            `json:"source_ref,omitempty"`
	BuildStatus   string            `json:"build_status,omitempty"`
	PreviewURL    string            `json:"preview_url,omitempty"`
	ProductionURL string            `json:"production_url,omitempty"`
	AccessMode    string            `json:"access_mode,omitempty"`
	AccessVersion int               `json:"access_version,omitempty"`
	AccessCheck   *urlCheckResponse `json:"access_check,omitempty"`
	Published     bool              `json:"published"`
	Waited        bool              `json:"waited"`
	LocalBuild    bool              `json:"local_build"`
	Resolution    string            `json:"resolution,omitempty"`
}

type urlCheckResponse struct {
	URL        string `json:"url,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
	OK         bool   `json:"ok"`
	Error      string `json:"error,omitempty"`
}

type deployIntentKind string

const (
	deployIntentCreateNewProject      deployIntentKind = "create_new_project"
	deployIntentUpdateTargetProject   deployIntentKind = "update_target_project"
	deployIntentUpdateExplicitProject deployIntentKind = "update_explicit_project"
	deployIntentLegacyUpsertByName    deployIntentKind = "legacy_upsert_by_name"
)

type deployIntent struct {
	Kind          deployIntentKind
	WorkspaceRoot string
	Targets       *targetStore
	TargetName    string
	Target        targetEntry
	HasTarget     bool
	ProjectID     string
	SourcePath    string
	WriteTarget   bool
}

type deployIntentInput struct {
	AbsPath        string
	WorkspaceRoot  string
	PathProvided   bool
	TargetProvided bool
	TargetName     string
	ProjectID      string
	CreateMode     bool
	UpdateMode     bool
	UpsertMode     bool
	NoWriteTarget  bool
	Targets        *targetStore
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVarP(&projectName, "name", "n", "", "Project name for new projects")
	deployCmd.Flags().StringVar(&targetName, "target", "", "Local RobotX target name")
	deployCmd.Flags().StringVar(&projectID, "project-id", "", "Update an explicit RobotX project ID")
	deployCmd.Flags().StringVarP(&visibility, "visibility", "v", "private", "Project visibility (public/private)")
	deployCmd.Flags().BoolVar(&createMode, "create", false, "Create a new project and fail if the name already exists")
	deployCmd.Flags().BoolVar(&updateMode, "update", false, "Update an existing target or project")
	deployCmd.Flags().BoolVar(&upsertMode, "upsert", false, "Use legacy create-or-update behavior by project name")
	deployCmd.Flags().BoolVar(&noWriteTarget, "no-write-target", false, "Do not write .robotx/targets.json")
	deployCmd.Flags().BoolVar(&publish, "publish", true, "Publish to production after successful build")
	deployCmd.Flags().BoolVar(&wait, "wait", true, "Wait for build completion")
	deployCmd.Flags().IntVar(&timeout, "timeout", 600, "Build timeout in seconds")
	deployCmd.Flags().BoolVar(&localBuild, "local-build", true, "Build locally and upload artifacts (must remain true; RobotX cloud build is no longer supported)")
	deployCmd.Flags().StringVar(&installCmd, "install-command", "", "Override install command for local build")
	deployCmd.Flags().StringVar(&buildCmd, "build-command", "", "Override build command for local build")
	deployCmd.Flags().StringVar(&outputDir, "output-dir", "", "Override output directory for local build")
	deployCmd.Flags().StringVar(&workspaceRoot, "workspace-root", "", "Override local RobotX target record root")
	deployCmd.Flags().StringVar(&versionLabel, "version-label", "", "Optional build version label (e.g. v1.2.3)")
	deployCmd.Flags().StringVar(&sourceRef, "source-ref", "", "Optional source reference (e.g. tag:v1.2.3, branch:main@<sha>)")
	deployCmd.Flags().StringVar(&deployAccess, "access", "unchanged", "Access policy after deploy (unchanged|open|login|private)")
	deployCmd.Flags().BoolVar(&verifyURL, "verify-url", false, "Check whether the production URL is anonymously reachable")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	projectPath := "."
	pathProvided := len(args) > 0
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
	if err := validateDeployMode(); err != nil {
		return err
	}
	accessMode, err := normalizeDeployAccess(deployAccess)
	if err != nil {
		return err
	}
	targetProvided := cmd.Flags().Changed("target")
	projectIDValue := strings.TrimSpace(projectID)

	resolvedWorkspaceRoot, err := inferWorkspaceRoot(absPath, workspaceRoot)
	if err != nil {
		return newCLIError("invalid_workspace_root", "invalid workspace root", 1, err)
	}
	targets, err := loadTargetStore(resolvedWorkspaceRoot)
	if err != nil {
		return newCLIError("target_config_error", "failed to read RobotX targets", 1, err)
	}
	intent, err := resolveDeployIntent(deployIntentInput{
		AbsPath:        absPath,
		WorkspaceRoot:  resolvedWorkspaceRoot,
		PathProvided:   pathProvided,
		TargetProvided: targetProvided,
		TargetName:     targetName,
		ProjectID:      projectIDValue,
		CreateMode:     createMode,
		UpdateMode:     updateMode,
		UpsertMode:     upsertMode,
		NoWriteTarget:  noWriteTarget,
		Targets:        targets,
	})
	if err != nil {
		return err
	}
	absPath = intent.SourcePath
	if stat, err := os.Stat(absPath); err != nil || !stat.IsDir() {
		return newCLIError("invalid_project_path", fmt.Sprintf("project path is not a directory: %s", absPath), 1, err)
	}

	c := client.NewClient(baseURL, apiKey)
	usedProjectName, err := projectNameForIntent(absPath, intent)
	if err != nil {
		return err
	}
	var previewURL string
	var productionURL string

	version := resolveBuildVersionInput()
	if version != nil {
		logf("🏷️  Build version label: %s\n", valueOrDash(version.VersionLabel))
		logf("🔖 Source ref: %s\n", valueOrDash(version.SourceRef))
	}

	proj, resolution, err := resolveDeployProject(c, usedProjectName, intent)
	if err != nil {
		return err
	}
	usedProjectName = proj.Name
	logf("✅ Project ready: %s\n", proj.ProjectID)
	if intent.Kind == deployIntentCreateNewProject && intent.WriteTarget {
		if err := writeDeployTargetRecord(intent, proj, absPath, "", ""); err != nil {
			return newCLIError("target_write_failed", "failed to write RobotX target record", 1, err)
		}
	}

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
	if artifactDir == "" && intent.HasTarget && intent.Kind != deployIntentCreateNewProject && strings.TrimSpace(intent.Target.OutputDir) != "" {
		artifactDir = strings.TrimSpace(intent.Target.OutputDir)
	}
	if artifactDir == "" && plan != nil && strings.TrimSpace(plan.OutputDir) != "" {
		artifactDir = strings.TrimSpace(plan.OutputDir)
	}
	if artifactDir == "" && isStaticRootProject(absPath) {
		artifactDir = "."
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

	accessVersion := 0
	if accessMode != "unchanged" && build != nil && build.Status == "success" {
		input, err := accessPolicyInputForMode(accessMode)
		if err != nil {
			return err
		}
		logf("🔐 Updating access policy to %s...\n", accessMode)
		version, err := c.UpdateAccessPolicy(proj.ProjectID, input)
		if err != nil {
			return newCLIError("access_policy_failed", "failed to update access policy", 2, err)
		}
		if version != nil {
			accessVersion = version.Version
		}
		logf("✅ Access policy updated: %s\n", accessModeLabel(accessMode))
	}

	var accessCheck *urlCheckResponse
	if verifyURL {
		accessCheck = verifyProductionURL(c, productionURL)
		logURLCheck(accessCheck)
	}

	if intent.WriteTarget {
		if err := writeDeployTargetRecord(intent, proj, absPath, artifactDir, productionURL); err != nil {
			return newCLIError("target_write_failed", "failed to write RobotX target record", 1, err)
		}
	}

	if err := emitSuccess(cmd.Name(), deployResponse{
		ProjectID:     proj.ProjectID,
		ProjectName:   usedProjectName,
		Target:        intent.TargetName,
		WorkspaceRoot: intent.WorkspaceRoot,
		CommitID:      safeCommitID(commit),
		BuildID:       safeBuildID(build),
		VersionSeq:    safeBuildVersionSeq(build),
		VersionLabel:  safeBuildVersionLabel(build),
		SourceRef:     safeBuildSourceRef(build, version),
		BuildStatus:   safeBuildStatus(build),
		PreviewURL:    previewURL,
		ProductionURL: productionURL,
		AccessMode:    accessMode,
		AccessVersion: accessVersion,
		AccessCheck:   accessCheck,
		Published:     publish && productionURL != "",
		Waited:        wait,
		LocalBuild:    localBuild,
		Resolution:    resolution,
	}); err != nil {
		return newCLIError("output_error", "failed to render JSON output", 1, err)
	}

	return nil
}

func normalizeDeployAccess(value string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(value))
	if mode == "" {
		mode = "unchanged"
	}
	switch mode {
	case "unchanged", "open", "login", "private":
		return mode, nil
	default:
		return "", newCLIError("invalid_access_mode", "invalid --access value (expected unchanged, open, login, or private)", 1, nil)
	}
}

func verifyProductionURL(c *client.Client, productionURL string) *urlCheckResponse {
	productionURL = strings.TrimSpace(productionURL)
	if productionURL == "" {
		return &urlCheckResponse{
			OK:    false,
			Error: "production URL is unavailable",
		}
	}
	check, err := c.CheckURL(productionURL)
	if err != nil {
		return &urlCheckResponse{
			URL:   productionURL,
			OK:    false,
			Error: err.Error(),
		}
	}
	return &urlCheckResponse{
		URL:        check.URL,
		StatusCode: check.StatusCode,
		OK:         check.OK,
	}
}

func logURLCheck(check *urlCheckResponse) {
	if check == nil {
		return
	}
	if check.OK {
		logf("✅ Production URL is anonymously reachable (HTTP %d)\n", check.StatusCode)
		return
	}
	if check.StatusCode > 0 {
		logf("⚠️  Production URL is not anonymously reachable (HTTP %d)\n", check.StatusCode)
		return
	}
	logf("⚠️  Production URL check did not complete: %s\n", valueOrDash(check.Error))
}

func validateDeployMode() error {
	if createMode && updateMode {
		return newCLIError("invalid_deploy_mode", "--create and --update cannot be used together", 1, nil)
	}
	if upsertMode && (createMode || updateMode || strings.TrimSpace(projectID) != "") {
		return newCLIError("invalid_deploy_mode", "--upsert cannot be combined with --create, --update, or --project-id", 1, nil)
	}
	if upsertMode && strings.TrimSpace(projectName) == "" {
		return newCLIError("invalid_deploy_mode", "--upsert requires --name because it resolves projects by name", 1, nil)
	}
	if createMode && strings.TrimSpace(projectID) != "" {
		return newCLIError("invalid_deploy_mode", "--create cannot be combined with --project-id", 1, nil)
	}
	return nil
}

func projectNameForIntent(absPath string, intent *deployIntent) (string, error) {
	name := strings.TrimSpace(projectName)
	if name == "" {
		if intent != nil && intent.Kind == deployIntentUpdateTargetProject && strings.TrimSpace(intent.Target.Name) != "" {
			name = strings.TrimSpace(intent.Target.Name)
		} else {
			name = filepath.Base(absPath)
		}
	}
	name = strings.ToLower(strings.TrimSpace(name))
	if intent != nil && deployIntentRequiresProjectName(intent.Kind) {
		if err := validateProjectName(name); err != nil {
			return "", newCLIError("invalid_project_name", err.Error(), 1, nil)
		}
	}
	return name, nil
}

func deployIntentRequiresProjectName(kind deployIntentKind) bool {
	return kind == deployIntentCreateNewProject || kind == deployIntentLegacyUpsertByName
}

func resolveDeployIntent(input deployIntentInput) (*deployIntent, error) {
	projectIDValue := strings.TrimSpace(input.ProjectID)
	projectIDProvided := projectIDValue != ""

	if input.UpsertMode {
		if input.TargetProvided {
			return nil, newCLIError("invalid_deploy_mode", "--upsert cannot be combined with --target because it resolves projects by name", 1, nil)
		}
		return &deployIntent{
			Kind:          deployIntentLegacyUpsertByName,
			WorkspaceRoot: input.WorkspaceRoot,
			Targets:       input.Targets,
			SourcePath:    input.AbsPath,
			WriteTarget:   false,
		}, nil
	}

	var selectedTargetName string
	var selectedTarget targetEntry
	var hasSelectedTarget bool
	var err error
	if projectIDProvided && !input.TargetProvided {
		// An explicit project ID is a one-off update unless the caller also
		// chooses a local target record to bind it to.
		selectedTargetName = strings.TrimSpace(input.TargetName)
	} else {
		selectedTargetName, selectedTarget, hasSelectedTarget, err = input.Targets.selected(input.TargetName)
		if err != nil {
			return nil, newCLIErrorWithDetails("target_required", err.Error(), 1, map[string]interface{}{
				"targets_file": input.Targets.Path,
				"targets":      targetNames(input.Targets.Data.Targets),
			}, nil)
		}
	}
	if err := validateDeployTargetSelection(input.Targets, selectedTargetName, selectedTarget, hasSelectedTarget, input.TargetProvided, input.CreateMode, input.UpdateMode, projectIDValue); err != nil {
		return nil, err
	}

	sourcePath, err := resolveDeploySourcePath(input.AbsPath, input.WorkspaceRoot, input.PathProvided, selectedTargetName, selectedTarget, hasSelectedTarget, input.Targets)
	if err != nil {
		return nil, err
	}

	kind := deployIntentCreateNewProject
	switch {
	case projectIDProvided:
		kind = deployIntentUpdateExplicitProject
	case hasSelectedTarget && !input.CreateMode:
		kind = deployIntentUpdateTargetProject
	case input.UpdateMode:
		return nil, newCLIError("missing_project_target", "--update requires an existing --target or --project-id", 1, nil)
	}

	return &deployIntent{
		Kind:          kind,
		WorkspaceRoot: input.WorkspaceRoot,
		Targets:       input.Targets,
		TargetName:    selectedTargetName,
		Target:        selectedTarget,
		HasTarget:     hasSelectedTarget,
		ProjectID:     projectIDValue,
		SourcePath:    sourcePath,
		WriteTarget:   shouldWriteTargetRecord(projectIDProvided, input.TargetProvided, input.NoWriteTarget),
	}, nil
}

func validateDeployTargetSelection(targets *targetStore, selectedTargetName string, selectedTarget targetEntry, hasSelectedTarget bool, targetProvided bool, createModeValue bool, updateModeValue bool, projectIDValue string) error {
	projectIDValue = strings.TrimSpace(projectIDValue)
	projectIDProvided := projectIDValue != ""
	if updateModeValue && !hasSelectedTarget && !projectIDProvided {
		return newCLIError("missing_project_target", "--update requires an existing --target or --project-id", 1, nil)
	}
	if createModeValue && hasSelectedTarget {
		return newCLIError("target_exists", fmt.Sprintf("target %q already exists; use --update or choose a new --target", selectedTargetName), 1, nil)
	}
	if targetProvided && !hasSelectedTarget && targets != nil && len(targets.Data.Targets) > 0 && !createModeValue && !projectIDProvided {
		return newCLIErrorWithDetails("target_not_found", fmt.Sprintf("target %q was not found; use --create to create a new target", selectedTargetName), 1, map[string]interface{}{
			"targets_file": targets.Path,
			"targets":      targetNames(targets.Data.Targets),
		}, nil)
	}
	if projectIDProvided && targetProvided && hasSelectedTarget {
		existingProjectID := strings.TrimSpace(selectedTarget.ProjectID)
		if existingProjectID != "" && existingProjectID != projectIDValue {
			return newCLIErrorWithDetails("target_project_mismatch", fmt.Sprintf("target %q points to a different project", selectedTargetName), 1, map[string]interface{}{
				"target":               selectedTargetName,
				"target_project_id":    existingProjectID,
				"requested_project_id": projectIDValue,
			}, nil)
		}
	}
	return nil
}

func shouldWriteTargetRecord(projectIDProvided, targetProvided, noWriteTarget bool) bool {
	if noWriteTarget {
		return false
	}
	if projectIDProvided && !targetProvided {
		return false
	}
	return true
}

func resolveDeploySourcePath(absPath string, workspaceRoot string, pathProvided bool, selectedTargetName string, selectedTarget targetEntry, hasSelectedTarget bool, targets *targetStore) (string, error) {
	if pathProvided || !hasSelectedTarget {
		return absPath, nil
	}
	sourcePath := strings.TrimSpace(selectedTarget.SourcePath)
	if sourcePath == "" {
		details := map[string]interface{}{
			"target": selectedTargetName,
		}
		if targets != nil {
			details["targets_file"] = targets.Path
		}
		return "", newCLIErrorWithDetails("missing_source_path", fmt.Sprintf("target %q does not have a saved source path; pass the project path explicitly", selectedTargetName), 1, details, nil)
	}
	resolved, err := resolveSavedSourcePath(workspaceRoot, sourcePath)
	if err != nil {
		details := map[string]interface{}{
			"target":      selectedTargetName,
			"source_path": sourcePath,
		}
		if targets != nil {
			details["targets_file"] = targets.Path
		}
		return "", newCLIErrorWithDetails("invalid_source_path", fmt.Sprintf("target %q has an invalid saved source path; pass the project path explicitly", selectedTargetName), 1, details, err)
	}
	return resolved, nil
}

func resolveSavedSourcePath(workspaceRoot, sourcePath string) (string, error) {
	sourcePath = strings.TrimSpace(sourcePath)
	localPath := filepath.FromSlash(sourcePath)
	if sourcePath == "" || filepath.IsAbs(localPath) {
		return "", fmt.Errorf("source_path must be relative to the workspace")
	}
	cleanPath := filepath.Clean(localPath)
	if cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("source_path must stay inside the workspace")
	}
	resolved := filepath.Join(workspaceRoot, cleanPath)
	rel, err := filepath.Rel(workspaceRoot, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("source_path must stay inside the workspace")
	}
	return resolved, nil
}

func writeDeployTargetRecord(intent *deployIntent, proj *client.Project, sourcePath, outputDir, productionURL string) error {
	if intent == nil || !intent.WriteTarget {
		return nil
	}
	if intent.Targets == nil {
		return fmt.Errorf("target store is required")
	}
	if proj == nil || strings.TrimSpace(proj.ProjectID) == "" {
		return fmt.Errorf("project is required")
	}
	writeTargetName := strings.TrimSpace(intent.TargetName)
	if writeTargetName == "" {
		writeTargetName = defaultTargetName
	}
	projectName := strings.TrimSpace(proj.Name)
	intent.Targets.upsert(writeTargetName, targetEntry{
		ProjectID:     strings.TrimSpace(proj.ProjectID),
		Name:          projectName,
		SourcePath:    persistentSourcePath(intent.WorkspaceRoot, sourcePath),
		OutputDir:     strings.TrimSpace(outputDir),
		ProductionURL: strings.TrimSpace(productionURL),
	})
	if err := intent.Targets.save(); err != nil {
		return err
	}
	intent.TargetName = writeTargetName
	return nil
}

func resolveDeployProject(c *client.Client, usedProjectName string, intent *deployIntent) (*client.Project, string, error) {
	if intent == nil {
		return nil, "", newCLIError("invalid_deploy_intent", "deploy intent is required", 1, nil)
	}
	switch intent.Kind {
	case deployIntentUpdateExplicitProject:
		logf("📦 Updating project by ID: %s\n", strings.TrimSpace(intent.ProjectID))
		proj, err := c.GetProject(strings.TrimSpace(intent.ProjectID))
		if err != nil {
			return nil, "", newCLIError("api_error", "failed to get project", 2, err)
		}
		return proj, "updated_explicit", nil
	case deployIntentUpdateTargetProject:
		if strings.TrimSpace(intent.Target.ProjectID) == "" {
			return nil, "", newCLIError("invalid_target", "selected target does not have a project_id", 1, nil)
		}
		logf("📦 Updating project from local target: %s\n", strings.TrimSpace(intent.Target.ProjectID))
		proj, err := c.GetProject(strings.TrimSpace(intent.Target.ProjectID))
		if err != nil {
			return nil, "", newCLIError("api_error", "failed to get target project", 2, err)
		}
		return proj, "updated_target", nil
	case deployIntentLegacyUpsertByName:
		logf("📦 Resolving project by name (explicit upsert): %s\n", usedProjectName)
		proj, err := c.CreateProject(client.CreateProjectRequest{
			Name:           usedProjectName,
			Visibility:     visibility,
			ConflictPolicy: "reuse_owned",
		})
		if err != nil {
			return nil, "", newCLIError("api_error", "failed to upsert project", 2, err)
		}
		return proj, "upsert", nil
	case deployIntentCreateNewProject:
		logf("📦 Creating project by name: %s\n", usedProjectName)
		if err := ensureProjectNameAvailable(c, usedProjectName); err != nil {
			return nil, "", err
		}
		proj, err := c.CreateProject(client.CreateProjectRequest{
			Name:           usedProjectName,
			Visibility:     visibility,
			ConflictPolicy: "error",
		})
		if err != nil {
			var apiErr *client.APIError
			if errors.As(err, &apiErr) && apiErr.Code == "name_conflict" {
				return nil, "", newCLIErrorWithDetails("name_conflict", "project name already exists", 2, map[string]interface{}{
					"name":                apiErr.Name,
					"existing_project_id": apiErr.ExistingProjectID,
					"suggestions":         apiErr.Suggestions,
				}, err)
			}
			return nil, "", newCLIError("api_error", "failed to create project", 2, err)
		}
		return proj, "created", nil
	default:
		return nil, "", newCLIError("invalid_deploy_intent", fmt.Sprintf("unknown deploy intent: %s", intent.Kind), 1, nil)
	}
}

func ensureProjectNameAvailable(c *client.Client, name string) error {
	projects, err := c.ListProjects(0)
	if err != nil {
		return newCLIError("api_error", "failed to check existing projects", 2, err)
	}
	for _, project := range projects {
		if project == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(project.Name), strings.TrimSpace(name)) {
			return newCLIErrorWithDetails("name_conflict", "project name already exists", 2, map[string]interface{}{
				"name":                strings.TrimSpace(project.Name),
				"existing_project_id": strings.TrimSpace(project.ProjectID),
				"suggestions":         suggestProjectNames(strings.TrimSpace(name)),
			}, nil)
		}
	}
	return nil
}

func suggestProjectNames(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	return []string{
		suggestProjectName(name, "-2"),
		suggestProjectName(name, "-"+time.Now().UTC().Format("0102")),
	}
}

func suggestProjectName(name, suffix string) string {
	maxBaseLen := 63 - len(suffix)
	if maxBaseLen <= 0 {
		return name
	}
	if len(name) > maxBaseLen {
		name = strings.TrimRight(name[:maxBaseLen], "-")
	}
	if name == "" {
		return strings.TrimPrefix(suffix, "-")
	}
	return name + suffix
}

func targetNames(targets map[string]targetEntry) []string {
	names := make([]string, 0, len(targets))
	for name := range targets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func isStaticRootProject(projectPath string) bool {
	if !fileExists(filepath.Join(projectPath, "index.html")) {
		return false
	}
	if fileExists(filepath.Join(projectPath, "package.json")) ||
		fileExists(filepath.Join(projectPath, "src")) ||
		fileExists(filepath.Join(projectPath, "vite.config.js")) ||
		fileExists(filepath.Join(projectPath, "vite.config.ts")) ||
		fileExists(filepath.Join(projectPath, "next.config.js")) ||
		fileExists(filepath.Join(projectPath, "next.config.mjs")) ||
		fileExists(filepath.Join(projectPath, "astro.config.mjs")) ||
		fileExists(filepath.Join(projectPath, "nuxt.config.ts")) {
		return false
	}
	return true
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
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if shouldSkipArtifact(relPath) {
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

func shouldSkip(path string) bool {
	skipDirs := []string{
		"node_modules",
		".git",
		".agents",
		".robotx",
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

func shouldSkipArtifact(path string) bool {
	skipDirs := []string{
		".git",
		".agents",
		".robotx",
		".DS_Store",
		"__pycache__",
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
