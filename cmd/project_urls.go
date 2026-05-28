package cmd

import (
	"fmt"
	"strings"

	"github.com/robotx-dev/cli/pkg/client"
)

func projectPreviewURL(project *client.Project, fallbackBaseURL string) string {
	if project == nil {
		return ""
	}
	if previewURL := strings.TrimSpace(project.PreviewURL); previewURL != "" {
		return previewURL
	}
	if project.RuntimeRefs != nil && project.RuntimeRefs.Preview != nil {
		if previewURL := strings.TrimSpace(project.RuntimeRefs.Preview.URL); previewURL != "" {
			return previewURL
		}
	}
	projectID := strings.TrimSpace(project.ProjectID)
	baseURL := strings.TrimSuffix(strings.TrimSpace(fallbackBaseURL), "/")
	if projectID == "" || baseURL == "" {
		return ""
	}
	return fmt.Sprintf("%s/preview/%s", baseURL, projectID)
}

func resolvePublishURL(fallbackBaseURL string, project *client.Project) string {
	if project == nil {
		return ""
	}
	if publishURL := strings.TrimSpace(project.PublishURL); publishURL != "" {
		return publishURL
	}
	if project.RuntimeRefs != nil && project.RuntimeRefs.Publish != nil {
		if publishURL := strings.TrimSpace(project.RuntimeRefs.Publish.URL); publishURL != "" {
			return publishURL
		}
	}
	projectID := strings.TrimSpace(project.ProjectID)
	baseURL := strings.TrimSuffix(strings.TrimSpace(fallbackBaseURL), "/")
	if projectID == "" || baseURL == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", baseURL, projectID)
}
