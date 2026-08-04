package pool

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAbs(t *testing.T) {
	absDir := t.TempDir()
	if got := abs(absDir); got != absDir {
		t.Errorf("abs(%q) = %q, want %q", absDir, got, absDir)
	}
	rel := "some/relative/path"
	want, _ := filepath.Abs(rel)
	if got := abs(rel); got != want {
		t.Errorf("abs(%q) = %q, want %q", rel, got, want)
	}
}

func TestResolveSymlinks(t *testing.T) {
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	canonicalReal, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatal(err)
	}
	// a symlink resolves to its target
	if got := resolveSymlinks(link); got != canonicalReal {
		t.Errorf("resolveSymlinks(%q) = %q, want %q", link, got, canonicalReal)
	}
	// deepest existing ancestor resolved, remainder rejoined
	deep := filepath.Join(real, "a", "b")
	wantDeep := filepath.Join(canonicalReal, "a", "b")
	if got := resolveSymlinks(deep); got != wantDeep {
		t.Errorf("resolveSymlinks(%q) = %q, want %q", deep, got, wantDeep)
	}
	deepLink := filepath.Join(link, "a", "b")
	if got := resolveSymlinks(deepLink); got != wantDeep {
		t.Errorf("resolveSymlinks(%q) = %q, want %q", deepLink, got, wantDeep)
	}
}

// rootWant makes p exist so EvalSymlinks canonicalizes it (macOS /var → /private/var).
func rootWant(t *testing.T, p string) string {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

func TestRootPriority(t *testing.T) {
	emptyCfg := filepath.Join(t.TempDir(), "empty")
	if err := os.WriteFile(emptyCfg, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("env override", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("GIT_WORKPOOL_HOME", dir)
		t.Setenv("GIT_CONFIG_GLOBAL", emptyCfg)
		if got, want := Root(), rootWant(t, dir); got != want {
			t.Errorf("Root() = %q, want %q", got, want)
		}
	})

	t.Run("git config fallback", func(t *testing.T) {
		dir := t.TempDir()
		cfg := filepath.Join(t.TempDir(), "config")
		if err := os.WriteFile(cfg, []byte("[workpool]\n\thome = "+dir+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GIT_WORKPOOL_HOME", "")
		t.Setenv("GIT_CONFIG_GLOBAL", cfg)
		t.Setenv("XDG_DATA_HOME", "")
		if got, want := Root(), rootWant(t, dir); got != want {
			t.Errorf("Root() = %q, want %q", got, want)
		}
	})

	t.Run("xdg fallback", func(t *testing.T) {
		xdg := t.TempDir()
		t.Setenv("GIT_WORKPOOL_HOME", "")
		t.Setenv("GIT_CONFIG_GLOBAL", emptyCfg)
		t.Setenv("XDG_DATA_HOME", xdg)
		if got, want := Root(), rootWant(t, filepath.Join(xdg, "git-workpool")); got != want {
			t.Errorf("Root() = %q, want %q", got, want)
		}
	})

	t.Run("home fallback", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("GIT_WORKPOOL_HOME", "")
		t.Setenv("GIT_CONFIG_GLOBAL", emptyCfg)
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("HOME", home)
		if got, want := Root(), rootWant(t, filepath.Join(home, ".local", "share", "git-workpool")); got != want {
			t.Errorf("Root() = %q, want %q", got, want)
		}
	})
}

func TestProjectDirHubDir(t *testing.T) {
	if got, want := ProjectDir("/pool", "proj"), filepath.Join("/pool", "proj"); got != want {
		t.Errorf("ProjectDir = %q, want %q", got, want)
	}
	if got, want := HubDir("/pool", "proj"), filepath.Join("/pool", "proj", "hub.git"); got != want {
		t.Errorf("HubDir = %q, want %q", got, want)
	}
}
