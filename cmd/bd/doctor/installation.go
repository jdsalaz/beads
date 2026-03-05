package doctor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/steveyegge/beads/cmd/bd/doctor/fix"
	"github.com/steveyegge/beads/internal/configfile"
	"github.com/steveyegge/beads/internal/storage/dolt"
)

// CheckInstallation verifies that .beads directory exists
func CheckInstallation(path string) DoctorCheck {
	beadsDir := filepath.Join(path, ".beads")
	if _, err := os.Stat(beadsDir); os.IsNotExist(err) {
		// Auto-detect prefix from directory name
		prefix := filepath.Base(path)
		prefix = strings.TrimRight(prefix, "-")

		return DoctorCheck{
			Name:    "Installation",
			Status:  StatusError,
			Message: "No .beads/ directory found",
			Fix:     fmt.Sprintf("Run 'bd init --prefix %s' to initialize beads", prefix),
		}
	}

	return DoctorCheck{
		Name:    "Installation",
		Status:  StatusOK,
		Message: ".beads/ directory found",
	}
}

// CheckPermissions verifies that .beads directory and database are readable/writable
func CheckPermissions(path string) DoctorCheck {
	// Follow redirect to resolve actual beads directory (bd-tvus fix)
	beadsDir := resolveBeadsDir(filepath.Join(path, ".beads"))

	// Check if .beads/ is writable
	testFile := filepath.Join(beadsDir, ".doctor-test-write")
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		return DoctorCheck{
			Name:    "Permissions",
			Status:  StatusError,
			Message: ".beads/ directory is not writable",
			Fix:     "Run 'bd doctor --fix' to fix permissions",
		}
	}
	_ = os.Remove(testFile) // Clean up test file (intentionally ignore error)

	// Check Dolt database directory permissions
	cfg, err := configfile.Load(beadsDir)
	if err == nil && cfg != nil && cfg.GetBackend() == configfile.BackendDolt {
		doltPath := getDatabasePath(beadsDir)
		if info, err := os.Stat(doltPath); err == nil {
			if !info.IsDir() {
				return DoctorCheck{
					Name:    "Permissions",
					Status:  StatusError,
					Message: "dolt/ is not a directory",
					Fix:     "Run 'bd doctor --fix' to fix permissions",
				}
			}
			// Try to open Dolt store read-only to verify accessibility
			ctx := context.Background()
			store, err := dolt.NewFromConfigWithOptions(ctx, beadsDir, &dolt.Config{ReadOnly: true})
			if err != nil {
				return DoctorCheck{
					Name:    "Permissions",
					Status:  StatusError,
					Message: "Dolt database exists but cannot be opened",
					Detail:  err.Error(),
					Fix:     "Run 'bd doctor --fix' to fix permissions",
				}
			}
			_ = store.Close()
		}
	}

	return DoctorCheck{
		Name:    "Permissions",
		Status:  StatusOK,
		Message: "All permissions OK",
	}
}

// FixPermissions fixes file permission issues in the .beads directory
func FixPermissions(path string) error {
	return fix.Permissions(path)
}
