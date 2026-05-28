package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type APIError struct {
	StatusCode        int
	Code              string
	Message           string
	Name              string
	ExistingProjectID string
	Suggestions       []string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	msg := strings.TrimSpace(e.Message)
	if msg == "" {
		msg = "API error"
	}
	if strings.TrimSpace(e.Code) != "" {
		return fmt.Sprintf("API error (status %d, code %s): %s", e.StatusCode, strings.TrimSpace(e.Code), msg)
	}
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, msg)
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Project represents a RobotX project
type Project struct {
	ProjectID   string              `json:"project_id"`
	Name        string              `json:"name"`
	Visibility  string              `json:"visibility"`
	PreviewURL  string              `json:"preview_url,omitempty"`
	PublishURL  string              `json:"publish_url,omitempty"`
	RuntimeRefs *ProjectRuntimeRefs `json:"runtime_refs,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	DeletedAt   *time.Time          `json:"deleted_at,omitempty"`
}

type AccessPolicy struct {
	RequirePlatformLogin bool              `json:"require_platform_login"`
	Credentials          *CredentialConfig `json:"credentials,omitempty"`
	Version              int               `json:"version,omitempty"`
}

type CredentialConfig struct {
	InviteCode      *InviteCodePublic `json:"invite_code,omitempty"`
	AllowSignedLink bool              `json:"allow_signed_link,omitempty"`
	Allowlist       bool              `json:"allowlist,omitempty"`
}

type InviteCodePublic struct {
	Hint string `json:"hint,omitempty"`
}

type AccessPolicyInput struct {
	RequirePlatformLogin bool             `json:"require_platform_login"`
	Credentials          *CredentialInput `json:"credentials,omitempty"`
}

type CredentialInput struct {
	Allowlist       bool `json:"allowlist,omitempty"`
	AllowSignedLink bool `json:"allow_signed_link,omitempty"`
}

type AccessPolicyVersion struct {
	Version int `json:"version"`
}

type URLCheck struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code,omitempty"`
	OK         bool   `json:"ok"`
}

type RuntimeRefVersion struct {
	Ref          string    `json:"ref"`
	ArtifactID   string    `json:"artifact_id,omitempty"`
	BuildID      string    `json:"build_id,omitempty"`
	CommitID     string    `json:"commit_id,omitempty"`
	VersionSeq   int64     `json:"version_seq,omitempty"`
	VersionLabel string    `json:"version_label,omitempty"`
	SourceRef    string    `json:"source_ref,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
	URL          string    `json:"url,omitempty"`
}

type ProjectRuntimeRefs struct {
	Preview *RuntimeRefVersion `json:"preview,omitempty"`
	Publish *RuntimeRefVersion `json:"publish,omitempty"`
}

// BuildPlan describes detected build instructions from server-side scanning.
type BuildPlan struct {
	Strategy       string   `json:"strategy,omitempty"`
	NeedsBuild     bool     `json:"needs_build"`
	ProjectType    string   `json:"project_type,omitempty"`
	PackageManager string   `json:"package_manager,omitempty"`
	InstallCommand string   `json:"install_command,omitempty"`
	BuildCommand   string   `json:"build_command,omitempty"`
	OutputDir      string   `json:"output_dir,omitempty"`
	NodeVersion    string   `json:"node_version,omitempty"`
	RuntimeImage   string   `json:"runtime_image,omitempty"`
	Notes          []string `json:"notes,omitempty"`
}

// ScannerResult mirrors server-side scanning results attached to commits.
type ScannerResult struct {
	BuildPlan *BuildPlan `json:"build_plan,omitempty"`
}

// SourceCommit represents an uploaded source bundle.
type SourceCommit struct {
	CommitID      string         `json:"commit_id"`
	ProjectID     string         `json:"project_id"`
	ScannerResult *ScannerResult `json:"scanner_result,omitempty"`
}

type BuildVersionInput struct {
	VersionLabel string `json:"version_label,omitempty"`
	SourceRef    string `json:"source_ref,omitempty"`
}

// Build represents a build task
type Build struct {
	BuildID           string     `json:"build_id"`
	ProjectID         string     `json:"project_id"`
	CommitID          string     `json:"commit_id"`
	VersionSeq        int64      `json:"version_seq,omitempty"`
	VersionLabel      string     `json:"version_label,omitempty"`
	SourceRef         string     `json:"source_ref,omitempty"`
	Status            string     `json:"status"`
	RuntimeArtifactID string     `json:"runtime_artifact_id,omitempty"`
	ErrorMsg          string     `json:"error_msg,omitempty"`
	PreviewPath       string     `json:"preview_path,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
}

// CreateProjectRequest represents project creation request
type CreateProjectRequest struct {
	Name           string `json:"name"`
	Visibility     string `json:"visibility,omitempty"`
	ConflictPolicy string `json:"conflict_policy,omitempty"`
}

// CreateProject creates a new project
func (c *Client) CreateProject(req CreateProjectRequest) (*Project, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest("POST", "/api/projects", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var project Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &project, nil
}

// GetProject retrieves project information
func (c *Client) GetProject(projectID string) (*Project, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/projects/%s", projectID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var project Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &project, nil
}

// DeleteProject deletes a RobotX project.
func (c *Client) DeleteProject(projectID string) error {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return fmt.Errorf("project ID is required")
	}

	resp, err := c.doRequest(http.MethodDelete, fmt.Sprintf("/api/projects/%s", url.PathEscape(projectID)), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}
	return nil
}

// ListProjects lists projects for current account.
func (c *Client) ListProjects(limit int) ([]*Project, error) {
	path := "/api/projects"
	if limit > 0 {
		path = fmt.Sprintf("%s?limit=%d", path, limit)
	}
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	projects, err := decodeProjectListResponse(rawBody)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(projects) > limit {
		return projects[:limit], nil
	}
	return projects, nil
}

func decodeProjectListResponse(raw []byte) ([]*Project, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return []*Project{}, nil
	}

	projects, parsed, err := extractProjectsFromJSON(trimmed)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if !parsed {
		return nil, fmt.Errorf("failed to decode response: unsupported project list payload")
	}
	return projects, nil
}

func extractProjectsFromJSON(raw []byte) ([]*Project, bool, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return []*Project{}, true, nil
	}

	var projects []*Project
	if err := json.Unmarshal(trimmed, &projects); err == nil {
		if projects == nil {
			return []*Project{}, true, nil
		}
		return projects, true, nil
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &payload); err != nil {
		return nil, false, nil
	}

	for _, key := range []string{"projects", "items", "list", "results", "data"} {
		child, ok := payload[key]
		if !ok {
			continue
		}
		nested, parsed, err := extractProjectsFromJSON(child)
		if err != nil {
			return nil, true, err
		}
		if parsed {
			return nested, true, nil
		}
	}

	return nil, false, nil
}

// UploadSource uploads source code and creates a commit/build.
func (c *Client) UploadSource(projectID, sourcePath string, version *BuildVersionInput) (*SourceCommit, *Build, error) {
	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	file, err := os.Open(sourcePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open source file: %w", err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(sourcePath))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, nil, fmt.Errorf("failed to copy file: %w", err)
	}
	if version != nil {
		if versionLabel := strings.TrimSpace(version.VersionLabel); versionLabel != "" {
			if err := writer.WriteField("version_label", versionLabel); err != nil {
				return nil, nil, fmt.Errorf("failed to write version_label: %w", err)
			}
		}
		if sourceRef := strings.TrimSpace(version.SourceRef); sourceRef != "" {
			if err := writer.WriteField("source_ref", sourceRef); err != nil {
				return nil, nil, fmt.Errorf("failed to write source_ref: %w", err)
			}
		}
	}

	if err := writer.Close(); err != nil {
		return nil, nil, fmt.Errorf("failed to close writer: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/projects/%s/commits", c.baseURL, projectID), body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to upload source: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		if resp.StatusCode == http.StatusAccepted {
			// Continue parsing for APIs that accept upload asynchronously.
		} else {
			return nil, nil, c.parseError(resp)
		}
	}

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Commit   *SourceCommit `json:"commit"`
		Build    *Build        `json:"build"`
		CommitID string        `json:"commit_id"`
	}
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &result); err != nil {
			// Try common wrapped payload structure: {"data": {...}}
			var wrapped struct {
				Data struct {
					Commit   *SourceCommit `json:"commit"`
					Build    *Build        `json:"build"`
					CommitID string        `json:"commit_id"`
				} `json:"data"`
			}
			if err2 := json.Unmarshal(rawBody, &wrapped); err2 == nil {
				result.Commit = wrapped.Data.Commit
				result.Build = wrapped.Data.Build
				result.CommitID = wrapped.Data.CommitID
			} else {
				return nil, nil, fmt.Errorf("failed to decode response: %w", err)
			}
		}
	}

	if result.Commit == nil && result.CommitID != "" {
		result.Commit = &SourceCommit{CommitID: result.CommitID, ProjectID: projectID}
	}

	// Some APIs return a top-level build_id without build object.
	if result.Build == nil && len(rawBody) > 0 {
		var fallback struct {
			BuildID string `json:"build_id"`
			Data    struct {
				BuildID string `json:"build_id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rawBody, &fallback); err == nil {
			buildID := strings.TrimSpace(fallback.BuildID)
			if buildID == "" {
				buildID = strings.TrimSpace(fallback.Data.BuildID)
			}
			if buildID != "" {
				result.Build = &Build{BuildID: buildID, ProjectID: projectID}
			}
		}
	}

	return result.Commit, result.Build, nil
}

// GetBuild retrieves build information.
func (c *Client) GetBuild(projectID, buildID string) (*Build, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/builds/%s", buildID), nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound && projectID != "" {
		resp.Body.Close()
		resp, err = c.doRequest("GET", fmt.Sprintf("/api/projects/%s/builds/%s", projectID, buildID), nil)
		if err != nil {
			return nil, err
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var build Build
	if err := json.NewDecoder(resp.Body).Decode(&build); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &build, nil
}

// ListBuildsForProject lists recent builds for a project.
func (c *Client) ListBuildsForProject(projectID string, limit int) ([]*Build, error) {
	path := fmt.Sprintf("/api/projects/%s/builds", projectID)
	if limit > 0 {
		path = fmt.Sprintf("%s?limit=%d", path, limit)
	}
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var builds []*Build
	if err := json.NewDecoder(resp.Body).Decode(&builds); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return builds, nil
}

// PublishBuild publishes a build to production
func (c *Client) PublishBuild(projectID, buildID string) (string, error) {
	body, err := json.Marshal(map[string]string{
		"build_id": buildID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest("POST", fmt.Sprintf("/api/projects/%s/publish", projectID), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", c.parseError(resp)
	}

	var result struct {
		PublicPath string `json:"public_path"`
		Publish    *struct {
			URL string `json:"url"`
		} `json:"publish,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
		publicPath := strings.TrimSpace(result.PublicPath)
		if publicPath != "" {
			return publicPath, nil
		}
		if result.Publish != nil {
			return strings.TrimSpace(result.Publish.URL), nil
		}
		return "", nil
	}
	return "", nil
}

func (c *Client) GetAccessPolicy(projectID string) (*AccessPolicy, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/projects/%s/access-policy", projectID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var policy AccessPolicy
	if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &policy, nil
}

func (c *Client) UpdateAccessPolicy(projectID string, input AccessPolicyInput) (*AccessPolicyVersion, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest("PUT", fmt.Sprintf("/api/projects/%s/access-policy", projectID), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var version AccessPolicyVersion
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &version, nil
}

func (c *Client) CheckURL(rawURL string) (*URLCheck, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("url is required")
	}
	statusCode, err := c.checkURLWithMethod(http.MethodHead, rawURL)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusMethodNotAllowed {
		statusCode, err = c.checkURLWithMethod(http.MethodGet, rawURL)
		if err != nil {
			return nil, err
		}
	}
	return &URLCheck{
		URL:        rawURL,
		StatusCode: statusCode,
		OK:         statusCode >= 200 && statusCode < 400,
	}, nil
}

func (c *Client) checkURLWithMethod(method, rawURL string) (int, error) {
	req, err := http.NewRequest(method, rawURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

// UploadBuildArtifacts uploads a zip of build outputs for a given build.
func (c *Client) UploadBuildArtifacts(buildID, zipPath string) (*Build, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	file, err := os.Open(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open artifact file: %w", err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(zipPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/builds/%s/artifacts", c.baseURL, buildID), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to upload artifacts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return nil, c.parseError(resp)
	}

	var build Build
	if err := json.NewDecoder(resp.Body).Decode(&build); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &build, nil
}

func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

func (c *Client) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Error   interface{} `json:"error"`
		Message string      `json:"message"`
		Detail  string      `json:"detail"`
		Code    string      `json:"code"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil {
		msg := strings.TrimSpace(errResp.Message)
		if msg == "" {
			msg = strings.TrimSpace(errResp.Detail)
		}
		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Code:       strings.TrimSpace(errResp.Code),
		}
		if values, ok := errResp.Error.(map[string]interface{}); ok {
			if apiErr.Code == "" {
				apiErr.Code = firstString(values, "code", "error_code")
			}
			apiErr.Name = firstString(values, "name")
			apiErr.ExistingProjectID = firstString(values, "existing_project_id", "project_id")
			apiErr.Suggestions = stringList(values["suggestions"])
		}
		if msg == "" {
			switch v := errResp.Error.(type) {
			case string:
				msg = strings.TrimSpace(v)
			case map[string]interface{}:
				for _, key := range []string{"message", "detail", "error", "msg"} {
					if raw, ok := v[key]; ok {
						if s, ok := raw.(string); ok && strings.TrimSpace(s) != "" {
							msg = strings.TrimSpace(s)
							break
						}
					}
				}
			}
		}

		if msg != "" {
			apiErr.Message = msg
			return apiErr
		}
	}

	trimmedBody := strings.TrimSpace(string(body))
	if trimmedBody == "" {
		return fmt.Errorf("API error: status %d", resp.StatusCode)
	}
	return fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, trimmedBody)
}

func firstString(values map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		raw, ok := values[key]
		if !ok {
			continue
		}
		if s, ok := raw.(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func stringList(raw interface{}) []string {
	values, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}
