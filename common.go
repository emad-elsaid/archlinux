// Package archlinux provides a declarative system configuration management framework for Arch Linux.
//
// It allows you to declare your system state (packages, services, files) and synchronizes the actual
// system state to match. The package supports various resource types including pacman packages,
// flatpak apps, npm packages, Go packages, Ruby gems, systemd services, user groups, system files,
// and dotfile management via GNU Stow.
package archlinux

import "github.com/emad-elsaid/types"

import (
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/samber/lo"
)

// addUnique appends items to target slice, skipping duplicates already present in the slice.
func addUnique(target *[]string, items ...string) {
	*target = append(*target, lo.Without(items, *target...)...)
}

// askYesNo prompts the user with a yes/no question and returns the boolean result.
func askYesNo(question string) (bool, error) {
	prompt := promptui.Select{
		Label: question,
		Items: []string{"No", "Yes"},
	}
	idx, _, err := prompt.Run()
	return idx == 1, err
}

// subtract returns elements in slice1 that are not in slice2, using the package manager's Match function.
func subtract(pm packageManager, slice1, slice2 []string) []string {
	return lo.Reject(slice1, func(want string, _ int) bool {
		return lo.ContainsBy(slice2, func(have string) bool { return pm.Match(want, have) })
	})
}

// intersect returns elements that appear in both slice1 and slice2, using the package manager's Match function.
func intersect(pm packageManager, slice1, slice2 []string) []string {
	return lo.Filter(slice1, func(want string, _ int) bool {
		return lo.ContainsBy(slice2, func(have string) bool { return pm.Match(want, have) })
	})
}

// getKeepPackages returns a map of packages to keep based on wanted packages and their dependencies.
// This recursively marks all dependencies of wanted packages as packages to keep.
func getKeepPackages(wanted []string, deps map[string][]string) map[string]bool {
	keep := make(map[string]bool)
	var mark func(string)
	mark = func(pkg string) {
		if keep[pkg] {
			return
		}
		keep[pkg] = true
		for _, dep := range deps[pkg] {
			mark(dep)
		}
	}
	for _, pkg := range wanted {
		mark(pkg)
	}
	return keep
}

// requireDir checks if a directory exists, fatally exiting if it doesn't.
func requireDir(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fatal("Directory does not exist", "path", path)
	}
}

// splitVer splits a package@version string into package and version parts.
// Returns empty version string if no @ is found.
func splitVer(pkg string) (name, ver string) {
	if idx := strings.LastIndex(pkg, "@"); idx != -1 {
		return pkg[:idx], pkg[idx+1:]
	}
	return pkg, ""
}

// splitNpmVer splits an npm package@version string, handling scoped packages (@scope/name@version).
// For scoped packages like @scope/name@1.0.0, it correctly identifies the version separator.
func splitNpmVer(pkg string) (name, ver string) {
	searchStart := 0
	if strings.HasPrefix(pkg, "@") {
		searchStart = 1
	}
	if idx := strings.Index(pkg[searchStart:], "@"); idx != -1 {
		actualIdx := searchStart + idx
		return pkg[:actualIdx], pkg[actualIdx+1:]
	}
	return pkg, ""
}

// matchWithVersion compares two versioned packages using the provided split function.
// Returns true if names match and either version is empty, "latest", or versions match.
func matchWithVersion(want, have string, splitFn func(string) (string, string)) bool {
	wantName, wantVer := splitFn(want)
	haveName, haveVer := splitFn(have)
	if wantName != haveName {
		return false
	}
	wantVer, haveVer = strings.TrimSpace(wantVer), strings.TrimSpace(haveVer)
	return wantVer == "" || wantVer == "latest" || haveVer == "" || haveVer == "latest" || wantVer == haveVer
}

// saveAsGoFile generates a Go source file that calls the given function with the provided items.
// This is used during the "save" operation to persist system state as declarative Go code.
func saveAsGoFile(filename, funcName string, items []string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "package main\n\nimport \"github.com/emad-elsaid/archlinux\"\n\n")
	fmt.Fprintf(&b, "func init() {\n\tarchlinux.%s(\n", funcName)
	for _, item := range items {
		fmt.Fprintf(&b, "\t\t%q,\n", item)
	}
	b.WriteString("\t)\n}\n")
	return os.WriteFile(filename, []byte(b.String()), 0644)
}

// listSystemdUnits returns enabled systemd units of the given type.
// If user is true, lists user units; otherwise lists system units.
func listSystemdUnits(unitType string, user bool) ([]string, error) {
	args := []string{"list-unit-files", "--type=" + unitType, "--state=enabled", "--no-legend"}
	if user {
		args = append([]string{"--user"}, args...)
	}

	stdout, err := types.Cmd("systemctl", args...).StdoutErr()
	if strings.TrimSpace(stdout) == "" {
		return []string{}, nil
	}

	if err != nil {
		return nil, err
	}

	suffix := "." + unitType
	var units []string
	for line := range strings.SplitSeq(strings.TrimSpace(stdout), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] != "static" {
			units = append(units, strings.TrimSuffix(fields[0], suffix))
		}
	}
	return units, nil
}
