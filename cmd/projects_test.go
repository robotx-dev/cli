package cmd

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestResolveProjectDeleteID(t *testing.T) {
	projectID, err := resolveProjectDeleteID([]string{"proj-1"}, "")
	if err != nil {
		t.Fatalf("resolve positional: %v", err)
	}
	if projectID != "proj-1" {
		t.Fatalf("projectID = %q, want proj-1", projectID)
	}

	projectID, err = resolveProjectDeleteID(nil, "proj-2")
	if err != nil {
		t.Fatalf("resolve flag: %v", err)
	}
	if projectID != "proj-2" {
		t.Fatalf("projectID = %q, want proj-2", projectID)
	}

	_, err = resolveProjectDeleteID([]string{"proj-1"}, "proj-2")
	var cliErr *cliError
	if !errors.As(err, &cliErr) || cliErr.Code != "invalid_project_id" {
		t.Fatalf("mismatch error = %#v, want invalid_project_id", err)
	}
}

func TestRunProjectsDeleteRequiresYes(t *testing.T) {
	oldYes, oldProjectID := projectsDeleteYes, projectsDeleteProjectID
	projectsDeleteYes = false
	projectsDeleteProjectID = ""
	t.Cleanup(func() {
		projectsDeleteYes = oldYes
		projectsDeleteProjectID = oldProjectID
	})

	err := runProjectsDelete(&cobra.Command{Use: "delete"}, []string{"proj-1"})
	var cliErr *cliError
	if !errors.As(err, &cliErr) || cliErr.Code != "confirmation_required" {
		t.Fatalf("error = %#v, want confirmation_required", err)
	}
}

func TestRunProjectsDeleteCallsAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/projects/proj-1" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	oldYes, oldProjectID := projectsDeleteYes, projectsDeleteProjectID
	oldBaseURL, oldAPIKey := viper.GetString("base_url"), viper.GetString("api_key")
	projectsDeleteYes = true
	projectsDeleteProjectID = ""
	viper.Set("base_url", server.URL)
	viper.Set("api_key", "test-key")
	t.Cleanup(func() {
		projectsDeleteYes = oldYes
		projectsDeleteProjectID = oldProjectID
		viper.Set("base_url", oldBaseURL)
		viper.Set("api_key", oldAPIKey)
	})

	if err := runProjectsDelete(&cobra.Command{Use: "delete"}, []string{"proj-1"}); err != nil {
		t.Fatalf("delete project: %v", err)
	}
}
