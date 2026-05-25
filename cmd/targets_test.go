package cmd

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInferWorkspaceRootUsesParentForBuildOutput(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "site")
	if err := os.MkdirAll(filepath.Join(project, "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "package.json"), []byte(`{"scripts":{"build":"vite build"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := inferWorkspaceRoot(filepath.Join(project, "dist"), "")
	if err != nil {
		t.Fatal(err)
	}
	if got != project {
		t.Fatalf("workspace root = %q, want %q", got, project)
	}
}

func TestInferWorkspaceRootUsesRobotXMarkerFromStaticSubdir(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "static-site")
	assets := filepath.Join(project, "assets")
	if err := os.MkdirAll(filepath.Join(project, robotxDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := inferWorkspaceRoot(assets, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != project {
		t.Fatalf("workspace root = %q, want %q", got, project)
	}
}

func TestTargetStoreRoundTripAndSelection(t *testing.T) {
	root := t.TempDir()
	store, err := loadTargetStore(root)
	if err != nil {
		t.Fatal(err)
	}
	store.upsert("preview", targetEntry{
		ProjectID:  "proj_123",
		Name:       "demo-preview",
		SourcePath: ".",
		OutputDir:  "dist",
	})
	if err := store.save(); err != nil {
		t.Fatal(err)
	}

	reloaded, err := loadTargetStore(root)
	if err != nil {
		t.Fatal(err)
	}
	name, entry, ok, err := reloaded.selected("")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected selected target")
	}
	if name != "preview" {
		t.Fatalf("selected target = %q, want preview", name)
	}
	if entry.ProjectID != "proj_123" || entry.OutputDir != "dist" {
		t.Fatalf("selected entry = %+v", entry)
	}
}

func TestWriteFileAtomicCleansTempFileOnRenameFailure(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, targetsFileName)
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "keep"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := writeFileAtomic(path, []byte(`{"version":1}`), 0o644)
	if err == nil {
		t.Fatal("expected rename over non-empty directory to fail")
	}
	entries, readErr := os.ReadDir(root)
	if readErr != nil {
		t.Fatal(readErr)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "."+targetsFileName+".tmp-") {
			t.Fatalf("temporary file was not cleaned up: %s", entry.Name())
		}
	}
	if _, err := os.Stat(filepath.Join(path, "keep")); err != nil {
		t.Fatalf("existing directory content should remain: %v", err)
	}
}

func TestTargetStoreRemovePromotesOnlyRemainingTarget(t *testing.T) {
	root := t.TempDir()
	store, err := loadTargetStore(root)
	if err != nil {
		t.Fatal(err)
	}
	store.upsert("main", targetEntry{ProjectID: "proj-main", Name: "main-site"})
	store.upsert("live", targetEntry{ProjectID: "proj-live", Name: "live-site"})
	store.Data.DefaultTarget = "main"

	removed, ok := store.remove("main")
	if !ok {
		t.Fatal("expected main target to be removed")
	}
	if removed.ProjectID != "proj-main" {
		t.Fatalf("removed target = %+v", removed)
	}
	if store.Data.DefaultTarget != "live" {
		t.Fatalf("default target = %q, want live", store.Data.DefaultTarget)
	}
	if _, ok := store.Data.Targets["main"]; ok {
		t.Fatal("main target still exists")
	}
}

func TestTargetStoreRemoveClearsDefaultWhenMultipleRemain(t *testing.T) {
	root := t.TempDir()
	store, err := loadTargetStore(root)
	if err != nil {
		t.Fatal(err)
	}
	store.upsert("main", targetEntry{ProjectID: "proj-main", Name: "main-site"})
	store.upsert("live", targetEntry{ProjectID: "proj-live", Name: "live-site"})
	store.upsert("preview", targetEntry{ProjectID: "proj-preview", Name: "preview-site"})
	store.Data.DefaultTarget = "main"

	if _, ok := store.remove("main"); !ok {
		t.Fatal("expected main target to be removed")
	}
	if store.Data.DefaultTarget != "" {
		t.Fatalf("default target = %q, want empty", store.Data.DefaultTarget)
	}
	if _, _, _, err := store.selected(""); err == nil {
		t.Fatal("expected selected target to require --target")
	}
}

func TestTargetStoreUpsertPreservesMissingDefaultWhenMultipleTargetsRemain(t *testing.T) {
	root := t.TempDir()
	store, err := loadTargetStore(root)
	if err != nil {
		t.Fatal(err)
	}
	store.upsert("live", targetEntry{ProjectID: "proj-live", Name: "live-site"})
	store.upsert("preview", targetEntry{ProjectID: "proj-preview", Name: "preview-site"})
	store.Data.DefaultTarget = ""

	store.upsert("live", targetEntry{ProjectID: "proj-live", Name: "live-site-v2"})
	if store.Data.DefaultTarget != "" {
		t.Fatalf("default target = %q, want empty", store.Data.DefaultTarget)
	}
	if _, _, _, err := store.selected(""); err == nil {
		t.Fatal("expected selected target to require --target")
	}
}

func TestValidateDeployTargetSelectionRejectsUnknownTargetWithoutCreate(t *testing.T) {
	root := t.TempDir()
	store, err := loadTargetStore(root)
	if err != nil {
		t.Fatal(err)
	}
	store.upsert("live", targetEntry{ProjectID: "proj-live", Name: "live-site"})

	err = validateDeployTargetSelection(store, "lvie", targetEntry{}, false, true, false, false, "")
	if err == nil {
		t.Fatal("expected unknown target to be rejected")
	}
	var cliErr *cliError
	if !errors.As(err, &cliErr) || cliErr.Code != "target_not_found" {
		t.Fatalf("error = %#v, want target_not_found", err)
	}
}

func TestValidateDeployTargetSelectionAllowsUnknownTargetWhenCreating(t *testing.T) {
	root := t.TempDir()
	store, err := loadTargetStore(root)
	if err != nil {
		t.Fatal(err)
	}
	store.upsert("live", targetEntry{ProjectID: "proj-live", Name: "live-site"})

	if err := validateDeployTargetSelection(store, "preview", targetEntry{}, false, true, true, false, ""); err != nil {
		t.Fatalf("expected create with a new target to be allowed, got %v", err)
	}
}

func TestValidateDeployTargetSelectionRejectsProjectIDMismatch(t *testing.T) {
	root := t.TempDir()
	store, err := loadTargetStore(root)
	if err != nil {
		t.Fatal(err)
	}
	store.upsert("live", targetEntry{ProjectID: "proj-live", Name: "live-site"})
	_, entry, ok, err := store.selected("live")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected live target")
	}

	err = validateDeployTargetSelection(store, "live", entry, true, true, false, false, "proj-other")
	if err == nil {
		t.Fatal("expected project id mismatch to be rejected")
	}
	var cliErr *cliError
	if !errors.As(err, &cliErr) || cliErr.Code != "target_project_mismatch" {
		t.Fatalf("error = %#v, want target_project_mismatch", err)
	}
}

func TestShouldWriteTargetRecordSkipsOneOffProjectIDUpdate(t *testing.T) {
	if shouldWriteTargetRecord(true, false, false) {
		t.Fatal("project-id without target should not write a target record")
	}
	if !shouldWriteTargetRecord(true, true, false) {
		t.Fatal("project-id with an explicit target should write a target record")
	}
	if shouldWriteTargetRecord(false, true, true) {
		t.Fatal("--no-write-target should always skip target writes")
	}
}

func TestResolveDeployIntentUpsertIgnoresExistingTarget(t *testing.T) {
	root := t.TempDir()
	store, err := loadTargetStore(root)
	if err != nil {
		t.Fatal(err)
	}
	store.upsert("live", targetEntry{ProjectID: "proj-live", Name: "live-site", SourcePath: "."})

	intent, err := resolveDeployIntent(deployIntentInput{
		AbsPath:       root,
		WorkspaceRoot: root,
		UpsertMode:    true,
		Targets:       store,
	})
	if err != nil {
		t.Fatalf("resolve deploy intent: %v", err)
	}
	if intent.Kind != deployIntentLegacyUpsertByName {
		t.Fatalf("intent kind = %q, want legacy upsert", intent.Kind)
	}
	if intent.HasTarget || intent.Target.ProjectID != "" {
		t.Fatalf("upsert should not select a target: %#v", intent)
	}
	if intent.WriteTarget {
		t.Fatalf("upsert should not write local target records")
	}
}

func TestResolveDeployIntentRejectsUpsertWithTarget(t *testing.T) {
	root := t.TempDir()
	store, err := loadTargetStore(root)
	if err != nil {
		t.Fatal(err)
	}
	store.upsert("live", targetEntry{ProjectID: "proj-live", Name: "live-site", SourcePath: "."})

	_, err = resolveDeployIntent(deployIntentInput{
		AbsPath:        root,
		WorkspaceRoot:  root,
		TargetProvided: true,
		TargetName:     "live",
		UpsertMode:     true,
		Targets:        store,
	})
	if err == nil {
		t.Fatal("expected --upsert with --target to be rejected")
	}
	var cliErr *cliError
	if !errors.As(err, &cliErr) || cliErr.Code != "invalid_deploy_mode" {
		t.Fatalf("error = %#v, want invalid_deploy_mode", err)
	}
}

func TestResolveDeploySourcePathRejectsTargetWithoutSavedSourcePath(t *testing.T) {
	root := t.TempDir()
	store, err := loadTargetStore(root)
	if err != nil {
		t.Fatal(err)
	}

	_, err = resolveDeploySourcePath(root, root, false, "live", targetEntry{ProjectID: "proj-live", Name: "live-site"}, true, store)
	if err == nil {
		t.Fatal("expected missing source path")
	}
	var cliErr *cliError
	if !errors.As(err, &cliErr) || cliErr.Code != "missing_source_path" {
		t.Fatalf("error = %#v, want missing_source_path", err)
	}
}

func TestResolveDeploySourcePathAllowsExplicitPathWithoutSavedSourcePath(t *testing.T) {
	root := t.TempDir()
	explicit := filepath.Join(root, "site")

	got, err := resolveDeploySourcePath(explicit, root, true, "live", targetEntry{ProjectID: "proj-live", Name: "live-site"}, true, nil)
	if err != nil {
		t.Fatalf("expected explicit path to be allowed, got %v", err)
	}
	if got != explicit {
		t.Fatalf("source path = %q, want %q", got, explicit)
	}
}

func TestResolveDeploySourcePathRejectsSavedPathOutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	store, err := loadTargetStore(root)
	if err != nil {
		t.Fatal(err)
	}

	_, err = resolveDeploySourcePath(root, root, false, "live", targetEntry{
		ProjectID:  "proj-live",
		Name:       "live-site",
		SourcePath: "../outside",
	}, true, store)
	if err == nil {
		t.Fatal("expected escaped source path to be rejected")
	}
	var cliErr *cliError
	if !errors.As(err, &cliErr) || cliErr.Code != "invalid_source_path" {
		t.Fatalf("error = %#v, want invalid_source_path", err)
	}
}

func TestResolveDeploySourcePathRejectsSavedAbsolutePath(t *testing.T) {
	root := t.TempDir()
	store, err := loadTargetStore(root)
	if err != nil {
		t.Fatal(err)
	}

	_, err = resolveDeploySourcePath(root, root, false, "live", targetEntry{
		ProjectID:  "proj-live",
		Name:       "live-site",
		SourcePath: filepath.ToSlash(filepath.Join(root, "site")),
	}, true, store)
	if err == nil {
		t.Fatal("expected absolute source path to be rejected")
	}
	var cliErr *cliError
	if !errors.As(err, &cliErr) || cliErr.Code != "invalid_source_path" {
		t.Fatalf("error = %#v, want invalid_source_path", err)
	}
}

func TestPersistentSourcePathKeepsWorkspaceRelativePath(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "site")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}

	got := persistentSourcePath(root, project)
	if got != "site" {
		t.Fatalf("persistent source path = %q, want site", got)
	}
}

func TestPersistentSourcePathSkipsOutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	project := filepath.Join(outside, "site")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}

	got := persistentSourcePath(root, project)
	if got != "" {
		t.Fatalf("persistent source path = %q, want empty for outside workspace", got)
	}
}

func TestPackageDirectorySkipsRobotXButKeepsBuildFolders(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "index.html"), "<h1>Hello</h1>")
	mustWriteFile(t, filepath.Join(root, "build", "chunk.js"), "console.log('ok')")
	mustWriteFile(t, filepath.Join(root, robotxDirName, targetsFileName), "{}")
	mustWriteFile(t, filepath.Join(root, ".agents", "skills", "robotx", "SKILL.md"), "secret")

	zipPath, err := packageDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(zipPath)

	names := zipNameSet(t, zipPath)
	if !names["index.html"] {
		t.Fatalf("zip names missing index.html: %#v", names)
	}
	if !names[filepath.Join("build", "chunk.js")] {
		t.Fatalf("zip names missing build/chunk.js: %#v", names)
	}
	if names[filepath.Join(robotxDirName, targetsFileName)] {
		t.Fatalf("zip names included RobotX metadata: %#v", names)
	}
	if names[filepath.Join(".agents", "skills", "robotx", "SKILL.md")] {
		t.Fatalf("zip names included agent metadata: %#v", names)
	}
}

func TestPackageSourceSkipsGeneratedAndRobotXFolders(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "package.json"), "{}")
	mustWriteFile(t, filepath.Join(root, "src", "main.js"), "console.log('src')")
	mustWriteFile(t, filepath.Join(root, "dist", "index.html"), "<h1>built</h1>")
	mustWriteFile(t, filepath.Join(root, robotxDirName, targetsFileName), "{}")
	mustWriteFile(t, filepath.Join(root, ".agents", "skills", "robotx", "SKILL.md"), "secret")

	zipPath, err := packageSource(root)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(zipPath)

	names := zipNameSet(t, zipPath)
	if !names["package.json"] || !names[filepath.Join("src", "main.js")] {
		t.Fatalf("zip names missing source files: %#v", names)
	}
	if names[filepath.Join("dist", "index.html")] {
		t.Fatalf("zip names included generated dist file: %#v", names)
	}
	if names[filepath.Join(robotxDirName, targetsFileName)] {
		t.Fatalf("zip names included RobotX metadata: %#v", names)
	}
	if names[filepath.Join(".agents", "skills", "robotx", "SKILL.md")] {
		t.Fatalf("zip names included agent metadata: %#v", names)
	}
}

func mustWriteFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func zipNameSet(t *testing.T, path string) map[string]bool {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	names := map[string]bool{}
	for _, file := range reader.File {
		names[file.Name] = true
	}
	return names
}
