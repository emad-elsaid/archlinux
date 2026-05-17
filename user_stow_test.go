package fest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Bug 1: parseApplyConflicts regex stops at newlines (uses .+ which doesn't match \n)
func TestUserStow_ParseApplyConflicts_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []string
	}{
		{
			name:     "file path with spaces",
			output:   "cannot stow something over existing target /home/user/My Documents/.bashrc since target exists",
			expected: []string{"/home/user/My Documents/.bashrc"},
		},
		{
			name:     "file path with parentheses",
			output:   "cannot stow something over existing target /home/user/.config/app (x86)/.conf since target exists",
			expected: []string{"/home/user/.config/app (x86)/.conf"},
		},
		{
			name: "file path with embedded newline",
			// Fixed: (?s:.+) matches newlines
			output:   "cannot stow something over existing target /home/user/.file\nwith\nnewline since target exists",
			expected: []string{"/home/user/.file\nwith\nnewline"}, // Fixed: now gets full path
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := newUserStow()
			result := u.parseApplyConflicts(tc.output)
			require.Equal(t, tc.expected, result, "Should handle special characters in paths")
		})
	}
}

// Bug 2: getDirs resolves symlinks via os.Getwd()
func TestUserStow_GetDirs_SymlinkWorkingDir(t *testing.T) {
	tmpDir := t.TempDir()
	symlinkPath := filepath.Join(tmpDir, "dotfiles-link")
	actualPath := filepath.Join(tmpDir, "dotfiles-actual")

	require.NoError(t, os.Mkdir(actualPath, 0755))
	require.NoError(t, os.Symlink(actualPath, symlinkPath))

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	originalPWD := os.Getenv("PWD")
	defer os.Setenv("PWD", originalPWD)

	require.NoError(t, os.Chdir(symlinkPath))
	require.NoError(t, os.Setenv("PWD", symlinkPath))

	u := newUserStow()
	dotfilesDir, stowDir, _, err := u.getDirs()
	require.NoError(t, err)

	// Fixed: os.Getenv("PWD") preserves symlinks
	require.Equal(t, symlinkPath, dotfilesDir, "getDirs should preserve symlink path")
	
	expectedStow := filepath.Join(symlinkPath, "user")
	require.Equal(t, expectedStow, stowDir, "stowDir should be relative to unresolved path")
}

// Bug 3: parseApplyConflicts requires "since" suffix, truncated output is missed
func TestUserStow_ParseApplyConflicts_TruncatedOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{
			name:   "output truncated mid-path",
			output: "cannot stow something over existing target /home/user/.bash",
		},
		{
			name:   "missing 'since' suffix",
			output: "cannot stow something over existing target /home/user/.bashrc",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := newUserStow()
			result := u.parseApplyConflicts(tc.output)
			// Bug: regex requires "since" at end, so truncated output returns empty
			// Real conflicts are missed, Apply proceeds without user interaction
			require.Empty(t, result, "Truncated output causes missed conflicts")
		})
	}
}

// Fixed: parseConflicts now handles newlines in paths
func TestUserStow_ParseConflicts_NewlineInPath(t *testing.T) {
	// Fixed: (?s:.+) matches newlines
	output := "existing target is not owned by stow: /home/user/.file\nwith\nnewline"
	
	u := newUserStow()
	result := u.parseConflicts(output)
	
	// Fixed: full path is captured including newlines
	expected := []string{"/home/user/.file\nwith\nnewline"}
	require.Equal(t, expected, result, "Path should include newlines")
}

// Bug 5: showDiff uses os.Stat which follows symlinks
func TestUserStow_ShowDiff_FollowsSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	stowDir := filepath.Join(tmpDir, "stow")
	targetDir := filepath.Join(tmpDir, "target")

	require.NoError(t, os.MkdirAll(stowDir, 0755))
	require.NoError(t, os.MkdirAll(targetDir, 0755))

	// Create stow file
	stowFile := filepath.Join(stowDir, "config.txt")
	require.NoError(t, os.WriteFile(stowFile, []byte("stow content"), 0644))

	// Target is a symlink to another file
	actualTarget := filepath.Join(tmpDir, "other.txt")
	require.NoError(t, os.WriteFile(actualTarget, []byte("other content"), 0644))
	
	targetFile := filepath.Join(targetDir, "config.txt")
	require.NoError(t, os.Symlink(actualTarget, targetFile))

	// Verify symlink exists
	info, err := os.Lstat(targetFile)
	require.NoError(t, err)
	require.NotEqual(t, 0, info.Mode()&os.ModeSymlink, "target should be a symlink")

	u := newUserStow()
	err = u.showDiff(targetFile, stowFile)
	require.NoError(t, err)

	// Bug: os.Stat follows symlinks, so we can't detect that target is a symlink
	// The diff compares stowFile with actualTarget content, not the symlink itself
	// Expected: should use Lstat to detect and warn about symlinks
	// Actual: silently follows symlink and diffs wrong file
}

// Bug 6: parseApplyConflicts doesn't validate that capture group exists
func TestUserStow_ParseApplyConflicts_MissingCaptureGroup(t *testing.T) {
	// While the regex pattern has a capture group (.+), if stow changes its output
	// format, we could match the outer pattern but not have a valid capture
	
	// This test demonstrates the fragility - any change to stow output breaks silently
	tests := []struct {
		name   string
		output string
	}{
		{
			name:   "stow v3.0 hypothetical format change",
			output: "CONFLICT: /home/user/.bashrc (target exists)",
		},
		{
			name:   "stow error message change",
			output: "ERROR: cannot install to /home/user/.bashrc: file exists",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := newUserStow()
			conflicts := u.parseApplyConflicts(tc.output)
			
			// Bug: returns empty for unrecognized format
			// Apply() proceeds without conflict resolution, using --override=.*
			// which might overwrite user files unexpectedly
			require.Empty(t, conflicts, "Unrecognized format returns empty, no error raised")
		})
	}
}
