package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/fs"
)

func TestGetAddHooks(t *testing.T) {
	t.Run("returns hooks from TOML config", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .grove.toml with hooks
		tomlContent := `[hooks]
add = ["pnpm i", "pnpm build"]
`
		tomlPath := filepath.Join(tmpDir, ".grove.toml")
		if err := os.WriteFile(tomlPath, []byte(tomlContent), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		hooks := GetAddHooks(tmpDir)

		if len(hooks) != 2 {
			t.Fatalf("Expected 2 hooks, got %d", len(hooks))
		}
		if hooks[0] != "pnpm i" || hooks[1] != "pnpm build" {
			t.Errorf("Unexpected hooks: %v", hooks)
		}
	})

	t.Run("returns empty when no config", func(t *testing.T) {
		tmpDir := t.TempDir()
		// No .grove.toml

		hooks := GetAddHooks(tmpDir)

		if len(hooks) != 0 {
			t.Errorf("Expected empty hooks, got %v", hooks)
		}
	})

	t.Run("returns empty when no add hooks defined", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .grove.toml without hooks section
		tomlContent := `[preserve]
patterns = [".env"]
`
		tomlPath := filepath.Join(tmpDir, ".grove.toml")
		if err := os.WriteFile(tomlPath, []byte(tomlContent), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		hooks := GetAddHooks(tmpDir)

		if len(hooks) != 0 {
			t.Errorf("Expected empty hooks, got %v", hooks)
		}
	})
}
