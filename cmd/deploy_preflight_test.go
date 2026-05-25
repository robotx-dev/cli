package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/haibingtown/robotx_cli/pkg/client"
)

func TestEnsureProjectNameAvailableRejectsExistingName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `{"projects":[{"project_id":"proj-existing","name":"my-app"}]}`)
	}))
	defer server.Close()

	err := ensureProjectNameAvailable(client.NewClient(server.URL, "test-key"), "my-app")
	if err == nil {
		t.Fatal("expected name conflict")
	}
	var cliErr *cliError
	if !errors.As(err, &cliErr) || cliErr.Code != "name_conflict" {
		t.Fatalf("error = %#v, want name_conflict", err)
	}
}

func TestSuggestProjectNamesKeepsDNSLength(t *testing.T) {
	name := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-"
	suggestions := suggestProjectNames(name)
	if len(suggestions) != 2 {
		t.Fatalf("suggestions = %#v, want 2", suggestions)
	}
	for _, suggestion := range suggestions {
		if len(suggestion) > 63 {
			t.Fatalf("suggestion %q length = %d, want <= 63", suggestion, len(suggestion))
		}
		if suggestion[len(suggestion)-1] == '-' {
			t.Fatalf("suggestion should not end with hyphen: %q", suggestion)
		}
	}
}

func TestProjectNameForExplicitProjectIDAllowsNonDNSPathName(t *testing.T) {
	oldProjectName := projectName
	projectName = ""
	t.Cleanup(func() {
		projectName = oldProjectName
	})

	_, err := projectNameForIntent("/tmp/我的网站", &deployIntent{Kind: deployIntentUpdateExplicitProject})
	if err != nil {
		t.Fatalf("project-id update should not validate local path name: %v", err)
	}
}

func TestProjectNameForCreateValidatesInferredPathName(t *testing.T) {
	oldProjectName := projectName
	projectName = ""
	t.Cleanup(func() {
		projectName = oldProjectName
	})

	_, err := projectNameForIntent("/tmp/我的网站", &deployIntent{Kind: deployIntentCreateNewProject})
	if err == nil {
		t.Fatal("expected create path name to be validated")
	}
	var cliErr *cliError
	if !errors.As(err, &cliErr) || cliErr.Code != "invalid_project_name" {
		t.Fatalf("error = %#v, want invalid_project_name", err)
	}
}

func TestWriteDeployTargetRecordPersistsMinimalTargetForCreate(t *testing.T) {
	root := t.TempDir()
	store, err := loadTargetStore(root)
	if err != nil {
		t.Fatal(err)
	}
	intent := &deployIntent{
		Kind:          deployIntentCreateNewProject,
		WorkspaceRoot: root,
		Targets:       store,
		WriteTarget:   true,
	}

	err = writeDeployTargetRecord(intent, &client.Project{ProjectID: "proj-new", Name: "my-app"}, root, "", "")
	if err != nil {
		t.Fatalf("write target: %v", err)
	}
	reloaded, err := loadTargetStore(root)
	if err != nil {
		t.Fatal(err)
	}
	name, entry, ok, err := reloaded.selected("")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || name != defaultTargetName {
		t.Fatalf("selected target = %q/%v, want default", name, ok)
	}
	if entry.ProjectID != "proj-new" || entry.Name != "my-app" || entry.SourcePath != "." {
		t.Fatalf("entry = %+v, want minimal project binding", entry)
	}
	if entry.OutputDir != "" || entry.ProductionURL != "" {
		t.Fatalf("minimal target should not contain deploy outputs: %+v", entry)
	}
}

func TestEnsureProjectNameAvailableAllowsNewName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `{"projects":[{"project_id":"proj-existing","name":"other-app"}]}`)
	}))
	defer server.Close()

	if err := ensureProjectNameAvailable(client.NewClient(server.URL, "test-key"), "my-app"); err != nil {
		t.Fatalf("expected name to be available, got %v", err)
	}
}

func TestResolveDeployProjectStopsBeforeCreateWhenPreflightFindsName(t *testing.T) {
	var createCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/projects":
			_, _ = fmt.Fprint(w, `{"projects":[{"project_id":"proj-existing","name":"my-app"}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/projects":
			atomic.AddInt32(&createCalls, 1)
			http.Error(w, "unexpected create", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	oldProjectID, oldUpsertMode, oldUpdateMode, oldCreateMode := projectID, upsertMode, updateMode, createMode
	projectID = ""
	upsertMode = false
	updateMode = false
	createMode = false
	t.Cleanup(func() {
		projectID = oldProjectID
		upsertMode = oldUpsertMode
		updateMode = oldUpdateMode
		createMode = oldCreateMode
	})

	_, _, err := resolveDeployProject(client.NewClient(server.URL, "test-key"), "my-app", &deployIntent{Kind: deployIntentCreateNewProject})
	if err == nil {
		t.Fatal("expected name conflict")
	}
	var cliErr *cliError
	if !errors.As(err, &cliErr) || cliErr.Code != "name_conflict" {
		t.Fatalf("error = %#v, want name_conflict", err)
	}
	if got := atomic.LoadInt32(&createCalls); got != 0 {
		t.Fatalf("create calls = %d, want 0", got)
	}
}

func TestResolveDeployProjectLegacyUpsertUsesReusePolicy(t *testing.T) {
	var sawCreate bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/projects" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		sawCreate = true
		var body struct {
			Name           string `json:"name"`
			ConflictPolicy string `json:"conflict_policy"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.Name != "my-app" || body.ConflictPolicy != "reuse_owned" {
			t.Fatalf("request body = %#v, want reuse_owned upsert", body)
		}
		_, _ = fmt.Fprint(w, `{"project_id":"proj-existing","name":"my-app"}`)
	}))
	defer server.Close()

	proj, resolution, err := resolveDeployProject(client.NewClient(server.URL, "test-key"), "my-app", &deployIntent{Kind: deployIntentLegacyUpsertByName})
	if err != nil {
		t.Fatalf("resolve deploy project: %v", err)
	}
	if !sawCreate {
		t.Fatal("expected create endpoint to be called")
	}
	if resolution != "upsert" || proj == nil || proj.ProjectID != "proj-existing" {
		t.Fatalf("resolution/project = %q/%#v", resolution, proj)
	}
}
