package client

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
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

func TestListProjectsEnforcesClientSideLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects" || r.URL.Query().Get("limit") != "1" {
			t.Fatalf("unexpected request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, `[
			{"project_id":"proj-1","name":"one"},
			{"project_id":"proj-2","name":"two"}
		]`)
	}))
	defer server.Close()

	projects, err := NewClient(server.URL, "test-key").ListProjects(1)
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if len(projects) != 1 || projects[0].ProjectID != "proj-1" {
		t.Fatalf("projects = %#v, want only first project", projects)
	}
}

func TestUpdateAccessPolicySendsOpenPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/projects/proj-1/access-policy" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization header = %q", got)
		}
		var body AccessPolicyInput
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.RequirePlatformLogin || body.Credentials != nil {
			t.Fatalf("body = %#v, want anonymous open policy", body)
		}
		_, _ = io.WriteString(w, `{"version":2}`)
	}))
	defer server.Close()

	version, err := NewClient(server.URL, "test-key").UpdateAccessPolicy("proj-1", AccessPolicyInput{RequirePlatformLogin: false})
	if err != nil {
		t.Fatalf("update access policy: %v", err)
	}
	if version == nil || version.Version != 2 {
		t.Fatalf("version = %#v, want 2", version)
	}
}

func TestCheckURLUsesHead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Fatalf("method = %s, want HEAD", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	check, err := NewClient("https://robotx.example", "test-key").CheckURL(server.URL)
	if err != nil {
		t.Fatalf("check url: %v", err)
	}
	if !check.OK || check.StatusCode != http.StatusOK {
		t.Fatalf("check = %#v, want HTTP 200 ok", check)
	}
}

func TestCheckURLFallsBackToGetWhenHeadIsNotAllowed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.WriteHeader(http.StatusMethodNotAllowed)
		case http.MethodGet:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("method = %s, want HEAD or GET", r.Method)
		}
	}))
	defer server.Close()

	check, err := NewClient("https://robotx.example", "test-key").CheckURL(server.URL)
	if err != nil {
		t.Fatalf("check url: %v", err)
	}
	if !check.OK || check.StatusCode != http.StatusNoContent {
		t.Fatalf("check = %#v, want fallback HTTP 204 ok", check)
	}
}
