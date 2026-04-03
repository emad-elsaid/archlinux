package fest

import "github.com/emad-elsaid/types"

import (
	"errors"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/samber/lo"
)

const ResourceBrokenSymlinks ResourceName = "broken symlinks"

var ignoreFiles = []string{
	"SingletonCookie",
	"SingletonLock",
	"SingletonSocket",
}

var ignoredDirs = []string{
	".git",
	".svn",
	".hg",
	"node_modules",
	".cache",
	".npm",
	".cargo",
	".rustup",
	".go",
	".mozilla",
	".steam",
	".wine",
	"vendor",
	"__pycache__",
	".local/share",
	".local/state",
	".var",
	".thunderbird",
	".dropbox",
	".vscode",
	".cursor",
}

var systemDirs = []string{
	"/etc",
	"/usr/local",
	"/opt",
	"/usr/share",
}

// Cache for ListInstalled to avoid expensive operations being called twice
var symlinkCache struct {
	sync.Mutex
	result []string
	cached bool
}

type symlinks struct{}

func (u symlinks) ResourceName() string                          { return string(ResourceBrokenSymlinks) }
func (u symlinks) Wanted() []string                              { return []string{} }
func (u symlinks) Match(want, have string) bool                  { return want == have }
func (u symlinks) GetDependencies() (map[string][]string, error) { return nil, nil }
func (u symlinks) SaveAsGo([]string) error                       { return nil }

// filterBrokenSymlinks is a PipeFn that filters symlink paths from stdin,
// keeping only broken symlinks (where the target doesn't exist).
func (u symlinks) filterBrokenSymlinks(stdin string) (stdout, stderr string, err error) {
	var broken []string

	for link := range strings.Lines(stdin) {
		link = strings.TrimSpace(link)
		// Skip ignored files
		if lo.ContainsBy(ignoreFiles, func(ignore string) bool {
			return strings.HasSuffix(link, "/"+ignore) || strings.HasSuffix(link, ignore)
		}) {
			continue
		}

		// Check if symlink target exists (broken = target doesn't exist)
		if _, statErr := os.Stat(link); os.IsNotExist(statErr) {
			broken = append(broken, link)
		}
	}

	return strings.Join(broken, "\n"), "", nil
}

// findBrokenSymlinks uses fd piped to a Go filter function to efficiently find broken symlinks.
// Requires: fd (installed via pacman/yay)
func (u symlinks) findBrokenSymlinks(searchPath string, useSudo bool) ([]string, error) {
	args := []string{"--type", "l", "--absolute-path", "--no-ignore"}

	// Add excluded directories
	for _, dir := range ignoredDirs {
		args = append(args, "--exclude", dir)
	}

	// Add search path at the end
	args = append(args, ".", searchPath)

	var cmd *types.Command
	if useSudo {
		cmd = types.Sudo("fd", args...).PipeFn(u.filterBrokenSymlinks)
	} else {
		cmd = types.Cmd("fd", args...).PipeFn(u.filterBrokenSymlinks)
	}

	out := cmd.Stdout()
	if err := cmd.Error(); err != nil {
		return nil, err
	}

	result := strings.TrimSpace(out)
	if result == "" {
		return []string{}, nil
	}

	return strings.Split(result, "\n"), nil
}

func (u symlinks) ListInstalled() ([]string, error) {
	// Return cached result if available
	symlinkCache.Lock()
	if symlinkCache.cached {
		result := symlinkCache.result
		symlinkCache.Unlock()
		return result, nil
	}
	symlinkCache.Unlock()

	// Get home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// Collect all directories to scan
	var dirsToScan []struct {
		path    string
		useSudo bool
	}
	dirsToScan = append(dirsToScan, struct {
		path    string
		useSudo bool
	}{home, false})

	var needsSudo bool
	for _, sysDir := range systemDirs {
		if _, err := os.Stat(sysDir); os.IsNotExist(err) {
			continue // Skip directories that don't exist
		}
		dirsToScan = append(dirsToScan, struct {
			path    string
			useSudo bool
		}{sysDir, true})
		needsSudo = true
	}

	// Pre-authenticate sudo once if needed, to avoid parallel sudo conflicts
	if needsSudo {
		if err := types.Cmd("sudo", "-v").Run().Error(); err != nil {
			slog.Debug("Failed to pre-authenticate sudo", "error", err)
		}
	}

	// Scan all directories in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allSymlinks []string

	for _, dir := range dirsToScan {
		wg.Go(func() {
			symlinks, err := u.findBrokenSymlinks(dir.path, dir.useSudo)
			if err != nil {
				slog.Debug("Failed to check "+dir.path, "error", err)
				return
			}
			mu.Lock()
			allSymlinks = append(allSymlinks, symlinks...)
			mu.Unlock()
		})
	}

	wg.Wait()

	// Cache the result
	symlinkCache.Lock()
	symlinkCache.result = allSymlinks
	symlinkCache.cached = true
	symlinkCache.Unlock()

	return allSymlinks, nil
}

func (u symlinks) ListExplicit() ([]string, error) {
	return u.ListInstalled()
}

func (u symlinks) Install([]string) error      { return nil }
func (u symlinks) MarkExplicit([]string) error { return nil }

func (u symlinks) isSystemPath(path string) bool {
	for _, sysDir := range systemDirs {
		if strings.HasPrefix(path, sysDir+"/") || path == sysDir {
			return true
		}
	}
	return false
}

func (u symlinks) Uninstall(symlinks []string) error {
	// Clear cache since we're modifying symlinks
	symlinkCache.Lock()
	symlinkCache.cached = false
	symlinkCache.result = nil
	symlinkCache.Unlock()

	errs := []error{}

	for _, link := range symlinks {
		slog.Debug("Removing " + link)

		if u.isSystemPath(link) {
			// Use sudo for system paths
			if err := types.Sudo("rm", link).Interactive().Error(); err != nil {
				errs = append(errs, err)
			}
		} else {
			// Regular removal for user paths
			if err := os.Remove(link); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errors.Join(errs...)
}
