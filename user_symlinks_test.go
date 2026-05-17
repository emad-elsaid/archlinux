package fest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFilterBrokenSymlinks_IgnoreFilesMatchesPartialNames tests the bug where
// filterBrokenSymlinks incorrectly filters symlinks whose names END with an ignore
// pattern, even if they're not exactly the ignored filename.
func TestFilterBrokenSymlinks_IgnoreFilesMatchesPartialNames(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create broken symlinks with names similar to ignored files
	// "MySingletonCookie" ends with "SingletonCookie" which is in ignoreFiles
	brokenPrefixedSymlink := filepath.Join(tmpDir, "MySingletonCookie")
	require.NoError(t, os.Symlink("/nonexistent/target1", brokenPrefixedSymlink))
	
	brokenNormalSymlink := filepath.Join(tmpDir, "normal_link")
	require.NoError(t, os.Symlink("/nonexistent/target2", brokenNormalSymlink))
	
	u := symlinks{}
	stdin := strings.Join([]string{
		brokenPrefixedSymlink,
		brokenNormalSymlink,
	}, "\n")
	
	stdout, stderr, err := u.filterBrokenSymlinks(stdin)
	require.NoError(t, err)
	require.Empty(t, stderr)
	
	broken := strings.Split(strings.TrimSpace(stdout), "\n")
	
	// BUG: Line 76 checks: strings.HasSuffix(link, ignore)
	// This matches "/path/MySingletonCookie" because it ends with "SingletonCookie"
	// But we probably only want to ignore files literally named "SingletonCookie"
	
	// Currently, MySingletonCookie gets incorrectly filtered out
	// This test will FAIL because the function incorrectly matches partial names
	require.Len(t, broken, 2, "Both broken symlinks should be included")
	require.Contains(t, broken, brokenPrefixedSymlink, 
		"MySingletonCookie should not be ignored just because it ends with 'SingletonCookie'")
	require.Contains(t, broken, brokenNormalSymlink)
}
