package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloud-ru/evolution-devservices-cli/internal/config"
)

// withTempConfig points EDS_CONFIG at a fresh, non-existent file inside a
// per-test temp dir so tests never touch the developer's real config.
func withTempConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("EDS_CONFIG", path)
	return path
}

// clearEnv resets every env var Load() consults so tests aren't affected by
// whatever happens to be exported in the shell running `go test`.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"EDS_PROJECT_ID", "EDS_REPO_API_URL", "EDS_API_KEY",
		"EDS_REPO_GIT_HOST", "EDS_WF_API_URL",
	} {
		t.Setenv(k, "")
	}
}

func TestLoad_Defaults(t *testing.T) {
	withTempConfig(t)
	clearEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.APIURL != config.DefaultAPIURL {
		t.Errorf("APIURL = %q, want %q", cfg.APIURL, config.DefaultAPIURL)
	}
	if cfg.GitHost != config.DefaultGitHost {
		t.Errorf("GitHost = %q, want %q", cfg.GitHost, config.DefaultGitHost)
	}
	if cfg.WorkflowAPIURL != config.DefaultWorkflowAPIURL {
		t.Errorf("WorkflowAPIURL = %q, want %q", cfg.WorkflowAPIURL, config.DefaultWorkflowAPIURL)
	}
	if cfg.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", cfg.APIKey)
	}
}

func TestLoad_FileOverridesDefaults(t *testing.T) {
	path := withTempConfig(t)
	clearEnv(t)

	writeConfig(t, path, config.Config{
		APIURL:    "https://file.example/api",
		ProjectID: "proj-from-file",
		APIKey:    "key-from-file",
	})

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.APIURL != "https://file.example/api" {
		t.Errorf("APIURL = %q, want file value", cfg.APIURL)
	}
	if cfg.ProjectID != "proj-from-file" {
		t.Errorf("ProjectID = %q, want file value", cfg.ProjectID)
	}
	if cfg.APIKey != "key-from-file" {
		t.Errorf("APIKey = %q, want file value", cfg.APIKey)
	}
	// GitHost was left empty in the file, so it should fall back to the default.
	if cfg.GitHost != config.DefaultGitHost {
		t.Errorf("GitHost = %q, want default %q when unset in file", cfg.GitHost, config.DefaultGitHost)
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	path := withTempConfig(t)
	clearEnv(t)

	writeConfig(t, path, config.Config{
		APIURL:    "https://file.example/api",
		ProjectID: "proj-from-file",
		APIKey:    "key-from-file",
	})

	t.Setenv("EDS_REPO_API_URL", "https://env.example/api")
	t.Setenv("EDS_API_KEY", "key-from-env")
	t.Setenv("EDS_PROJECT_ID", "proj-from-env")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.APIURL != "https://env.example/api" {
		t.Errorf("APIURL = %q, want env value", cfg.APIURL)
	}
	if cfg.APIKey != "key-from-env" {
		t.Errorf("APIKey = %q, want env value", cfg.APIKey)
	}
	if cfg.ProjectID != "proj-from-env" {
		t.Errorf("ProjectID = %q, want env value", cfg.ProjectID)
	}
}

func TestLoad_MissingFileIsNotAnError(t *testing.T) {
	withTempConfig(t) // path deliberately left non-existent
	clearEnv(t)

	if _, err := config.Load(); err != nil {
		t.Fatalf("Load() with no config file on disk should not error, got %v", err)
	}
}

func TestSaveThenLoad_RoundTrips(t *testing.T) {
	path := withTempConfig(t)
	clearEnv(t)

	cfg := &config.Config{
		APIURL:    "https://saved.example/api",
		ProjectID: "saved-project",
		APIKey:    "saved-key",
		GitHost:   "https://saved.example/git",
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat saved config: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config file mode = %o, want 0600", perm)
	}

	got, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.APIURL != cfg.APIURL || got.ProjectID != cfg.ProjectID ||
		got.APIKey != cfg.APIKey || got.GitHost != cfg.GitHost {
		t.Errorf("round-tripped config = %+v, want %+v", got, cfg)
	}
}

func TestLoad_TildeInEDSConfigExpandsToHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)        // os.UserHomeDir() honors $HOME on Unix
	t.Setenv("USERPROFILE", home) // ... and on Windows
	t.Setenv("EDS_CONFIG", "~/eds-test/config.json")
	clearEnv(t)

	cfg := &config.Config{ProjectID: "tilde-project"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	wantPath := filepath.Join(home, "eds-test", "config.json")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("expected config at %s, stat error = %v", wantPath, err)
	}

	got, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.ProjectID != "tilde-project" {
		t.Errorf("ProjectID = %q, want %q", got.ProjectID, "tilde-project")
	}
}

func writeConfig(t *testing.T, path string, cfg config.Config) {
	t.Helper()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
