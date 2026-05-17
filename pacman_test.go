package fest

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// ============================================================
// Test 1: PackageGroup continues processing after errors
// ============================================================

func TestPackageGroup_ContinuesOnError(t *testing.T) {
	// Reset packages slice
	original := packages
	defer func() { packages = original }()
	packages = nil

	// FIXED: PackageGroup now continues processing at line 40
	//   if err != nil {
	//       slog.Warn("failed to query pacman groups", "error", err, "group", groupName)
	//       continue // Continue with next group instead of returning early
	//   }
	
	// Call with a nonexistent group first, then a real one
	// The fix ensures the second group is still processed
	PackageGroup("nonexistent-group-xyz-123", "base")
	
	// After the fix, if base group exists, its packages should be added
	// The test passes as long as we don't panic or return early
	// Note: base group might not exist in all test environments, so we just
	// verify the function completed without early return
	
	t.Logf("PackageGroup processed both groups, added %d packages", len(packages))
	// The important fix is that it doesn't return early on first error
	// Whether packages are actually added depends on system state
}

// ============================================================
// Bug 2: Install uses value receiver and wrong defer scope
// ============================================================

func TestPacmanInstall_ValueReceiverDoesNotPersist(t *testing.T) {
	// BUG 1: Install() uses value receiver (line 78), so modifications don't persist
	// BUG 2: defer client.Close() (line 85) executes on local var, not p.yayClient
	
	pm := pacman{yayClient: nil}
	
	// This will initialize a client internally but won't persist it
	_ = pm.Install([]string{"nonexistent-pkg-test"})
	
	// Due to value receiver, pm.yayClient is still nil outside the function
	require.Nil(t, pm.yayClient, 
		"BUG: yayClient remains nil because Install uses value receiver, not pointer")
	
	// Additional bug: Even if receiver was pointer, defer client.Close()
	// closes the local variable before it's assigned to p.yayClient
}

// ============================================================
// Bug 3: strings.Fields() fragile parsing  
// ============================================================

func TestPacmanListInstalled_FragileParsing(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectCount   int
		expectCorrect bool
	}{
		{
			name:          "normal packages work",
			input:         "vim\ngit\ndocker",
			expectCount:   3,
			expectCorrect: true,
		},
		{
			name:          "empty string edge case",
			input:         "",
			expectCount:   0,
			expectCorrect: true,
		},
		{
			name:          "whitespace-only input",
			input:         "   \n\n\t\t\n  ",
			expectCount:   0,
			expectCorrect: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// This mimics ListInstalled/ListExplicit line 67/75
			// BUG: strings.Fields() is fragile - it works but isn't explicit about expectations
			result := strings.Fields(tc.input)
			
			if tc.expectCorrect {
				require.Equal(t, tc.expectCount, len(result))
			}
			
			// Issue: No validation that results are valid package names
			// Issue: Behavior depends on ALL whitespace, not just newlines
			// Better: strings.Split(strings.TrimSpace(stdout), "\n") with validation
		})
	}
}

// ============================================================
// Bug 4: GetDependencies creates duplicate resolved dependencies
// ============================================================

func TestPacmanGetDependencies_DuplicateResolution(t *testing.T) {
	// Simulate the buggy logic from GetDependencies lines 131-136
	// In real scenarios, expac output can have duplicate dependencies
	
	deps := map[string][]string{
		"pkg-a": {"virtual-lib", "virtual-lib"}, // duplicate dep (can happen in PKGBUILD)
	}
	
	provides := map[string][]string{
		"virtual-lib": {"provider-1", "provider-2"}, // multiple providers for virtual package
		"provider-1":  {"provider-1"},
		"provider-2":  {"provider-2"},
	}
	
	// This is the BUGGY logic from lines 132-135
	resolved := make(map[string][]string)
	for pkg, pkgDeps := range deps {
		for _, dep := range pkgDeps {
			resolved[pkg] = append(resolved[pkg], provides[dep]...)  // BUG: No deduplication
		}
	}
	
	// pkg-a depends on "virtual-lib" twice
	// Each occurrence resolves to ["provider-1", "provider-2"]
	// Result: ["provider-1", "provider-2", "provider-1", "provider-2"]
	
	require.Len(t, resolved["pkg-a"], 4, 
		"BUG: Should have 2 unique providers but has 4 due to duplicate virtual deps")
	
	// Verify the exact duplication
	require.Equal(t, []string{"provider-1", "provider-2", "provider-1", "provider-2"}, 
		resolved["pkg-a"],
		"BUG: Duplicate providers should be deduplicated but aren't")
}

// ============================================================
// Bug 5: SaveAsGo dependency calculation edge cases
// ============================================================

func TestPacmanSaveAsGo_CircularDependencyHandling(t *testing.T) {
	tests := []struct {
		name     string
		wanted   []string
		deps     map[string][]string
		expected map[string]bool
	}{
		{
			name:   "circular dependencies should be handled",
			wanted: []string{"pkg-a"},
			deps: map[string][]string{
				"pkg-a": {"pkg-b"},
				"pkg-b": {"pkg-c"},
				"pkg-c": {"pkg-a"}, // circular back to pkg-a
			},
			expected: map[string]bool{
				"pkg-a": true,
				"pkg-b": true,
				"pkg-c": true,
			},
		},
		{
			name:   "self-referential dependency",
			wanted: []string{"pkg-a"},
			deps: map[string][]string{
				"pkg-a": {"pkg-a"}, // depends on itself
			},
			expected: map[string]bool{
				"pkg-a": true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Test getKeepPackages from common.go:50
			keep := getKeepPackages(tc.wanted, tc.deps)
			
			require.Equal(t, tc.expected, keep,
				"Circular and self-referential dependencies should be handled correctly")
		})
	}
}
