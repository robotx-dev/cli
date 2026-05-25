package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	robotxDirName     = ".robotx"
	targetsFileName   = "targets.json"
	defaultTargetName = "main"
)

type targetsFile struct {
	Version       int                    `json:"version"`
	DefaultTarget string                 `json:"default_target,omitempty"`
	Targets       map[string]targetEntry `json:"targets"`
}

type targetEntry struct {
	Label         string `json:"label,omitempty"`
	ProjectID     string `json:"project_id"`
	Name          string `json:"name"`
	SourcePath    string `json:"source_path,omitempty"`
	OutputDir     string `json:"output_dir,omitempty"`
	ProductionURL string `json:"production_url,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
}

type targetStore struct {
	WorkspaceRoot string
	Path          string
	Data          targetsFile
}

func loadTargetStore(workspaceRoot string) (*targetStore, error) {
	path := filepath.Join(workspaceRoot, robotxDirName, targetsFileName)
	store := &targetStore{
		WorkspaceRoot: workspaceRoot,
		Path:          path,
		Data: targetsFile{
			Version:       1,
			DefaultTarget: defaultTargetName,
			Targets:       map[string]targetEntry{},
		},
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, err
	}
	var data targetsFile
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	store.Data = data
	normalizeTargetsFile(&store.Data)
	return store, nil
}

func (s *targetStore) save() error {
	if s == nil {
		return nil
	}
	if s.Data.Version == 0 {
		s.Data.Version = 1
	}
	if s.Data.Targets == nil {
		s.Data.Targets = map[string]targetEntry{}
	}
	normalizeTargetsFile(&s.Data)
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s.Data, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return writeFileAtomic(s.Path, raw, 0o644)
}

func writeFileAtomic(path string, raw []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	keepTemp := false
	defer func() {
		if !keepTemp {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	keepTemp = true
	syncDirectoryBestEffort(dir)
	return nil
}

func syncDirectoryBestEffort(dir string) {
	f, err := os.Open(dir)
	if err != nil {
		return
	}
	defer f.Close()
	_ = f.Sync()
}

func normalizeTargetsFile(data *targetsFile) {
	if data.Version == 0 {
		data.Version = 1
	}
	if data.Targets == nil {
		data.Targets = map[string]targetEntry{}
	}
	if len(data.Targets) == 0 {
		if strings.TrimSpace(data.DefaultTarget) == "" {
			data.DefaultTarget = defaultTargetName
		}
		return
	}
	if def := strings.TrimSpace(data.DefaultTarget); def != "" {
		if _, ok := data.Targets[def]; ok {
			data.DefaultTarget = def
			return
		}
	}
	if len(data.Targets) == 1 {
		for name := range data.Targets {
			data.DefaultTarget = name
			return
		}
	}
	data.DefaultTarget = ""
}

func (s *targetStore) selected(name string) (string, targetEntry, bool, error) {
	if s == nil || len(s.Data.Targets) == 0 {
		return strings.TrimSpace(name), targetEntry{}, false, nil
	}
	name = strings.TrimSpace(name)
	if name != "" {
		entry, ok := s.Data.Targets[name]
		return name, entry, ok, nil
	}
	if def := strings.TrimSpace(s.Data.DefaultTarget); def != "" {
		entry, ok := s.Data.Targets[def]
		if ok {
			return def, entry, true, nil
		}
	}
	if len(s.Data.Targets) == 1 {
		for key, entry := range s.Data.Targets {
			return key, entry, true, nil
		}
	}
	return "", targetEntry{}, false, fmt.Errorf("multiple RobotX targets found; pass --target")
}

func (s *targetStore) upsert(name string, entry targetEntry) {
	if s.Data.Targets == nil {
		s.Data.Targets = map[string]targetEntry{}
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultTargetName
	}
	now := time.Now().UTC().Format(time.RFC3339)
	existing := s.Data.Targets[name]
	if strings.TrimSpace(entry.Label) == "" {
		if strings.TrimSpace(existing.Label) != "" {
			entry.Label = existing.Label
		} else if name == defaultTargetName {
			entry.Label = "主站"
		} else {
			entry.Label = name
		}
	}
	if strings.TrimSpace(entry.CreatedAt) == "" {
		entry.CreatedAt = existing.CreatedAt
	}
	if strings.TrimSpace(entry.CreatedAt) == "" {
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now
	s.Data.Targets[name] = entry
	if len(s.Data.Targets) == 1 {
		s.Data.DefaultTarget = name
	}
}

func (s *targetStore) remove(name string) (targetEntry, bool) {
	if s == nil || s.Data.Targets == nil {
		return targetEntry{}, false
	}
	name = strings.TrimSpace(name)
	entry, ok := s.Data.Targets[name]
	if !ok {
		return targetEntry{}, false
	}
	delete(s.Data.Targets, name)
	if strings.TrimSpace(s.Data.DefaultTarget) == name {
		s.Data.DefaultTarget = ""
	}
	normalizeTargetsFile(&s.Data)
	return entry, true
}

func (s *targetStore) summaries() []targetSummary {
	if s == nil {
		return nil
	}
	names := make([]string, 0, len(s.Data.Targets))
	for name := range s.Data.Targets {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]targetSummary, 0, len(names))
	for _, name := range names {
		entry := s.Data.Targets[name]
		out = append(out, targetSummary{
			Name:          name,
			Label:         entry.Label,
			ProjectID:     entry.ProjectID,
			ProjectName:   entry.Name,
			SourcePath:    entry.SourcePath,
			OutputDir:     entry.OutputDir,
			ProductionURL: entry.ProductionURL,
			UpdatedAt:     entry.UpdatedAt,
			Default:       name == s.Data.DefaultTarget,
		})
	}
	return out
}

func inferWorkspaceRoot(absPath, override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		return filepath.Abs(override)
	}
	candidate := absPath
	if stat, err := os.Stat(candidate); err == nil && !stat.IsDir() {
		candidate = filepath.Dir(candidate)
	}
	if isBuildOutputDir(filepath.Base(candidate)) {
		candidate = filepath.Dir(candidate)
	}
	if root := nearestProjectRoot(candidate); root != "" {
		return root, nil
	}
	return candidate, nil
}

func nearestProjectRoot(start string) string {
	dir := start
	for {
		if fileExists(filepath.Join(dir, ".git")) ||
			fileExists(filepath.Join(dir, robotxDirName)) ||
			fileExists(filepath.Join(dir, "package.json")) ||
			fileExists(filepath.Join(dir, "robotx.yaml")) ||
			fileExists(filepath.Join(dir, "go.mod")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func isBuildOutputDir(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "dist", "build", "out":
		return true
	default:
		return false
	}
}

func persistentSourcePath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "" {
		return "."
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return ""
	}
	return filepath.ToSlash(rel)
}
