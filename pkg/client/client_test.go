package client

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestParseErrorReadsNestedProjectNameConflict(t *testing.T) {
	c := NewClient("https://robotx.example", "test-key")
	resp := &http.Response{
		StatusCode: http.StatusConflict,
		Body: io.NopCloser(strings.NewReader(`{
			"error": {
				"code": "name_conflict",
				"message": "project name already exists",
				"name": "my-app",
				"existing_project_id": "proj-existing",
				"suggestions": ["my-app-2", "my-app-0523"]
			}
		}`)),
	}

	err := c.parseError(resp)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %#v", err)
	}
	if apiErr.StatusCode != http.StatusConflict || apiErr.Code != "name_conflict" {
		t.Fatalf("unexpected status/code: %#v", apiErr)
	}
	if apiErr.Name != "my-app" || apiErr.ExistingProjectID != "proj-existing" {
		t.Fatalf("unexpected conflict fields: %#v", apiErr)
	}
	if len(apiErr.Suggestions) != 2 || apiErr.Suggestions[0] != "my-app-2" {
		t.Fatalf("unexpected suggestions: %#v", apiErr.Suggestions)
	}
}
