package fest

import "github.com/emad-elsaid/types"

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

const ResourceNpmPackages ResourceName = "npm packages"

var wantedNpmPackages []string

// NpmPackage declares global npm packages to install.
// Supports version pinning using @version syntax.
// Handles scoped packages correctly (e.g., @vue/cli).
//
// Example:
//
//	fest.NpmPackage(
//	    "typescript",
//	    "@vue/cli",
//	    "eslint@8.50.0",  // Pin to specific version
//	)
func NpmPackage(pkgs ...string) { addUnique(&wantedNpmPackages, pkgs...) }

type npmPackages struct{}

func (n npmPackages) Wanted() []string     { return wantedNpmPackages }
func (n npmPackages) ResourceName() string { return string(ResourceNpmPackages) }

func (n npmPackages) Match(want, have string) bool {
	return matchWithVersion(want, have, splitNpmVer)
}

type npmListOutput struct {
	Dependencies map[string]struct {
		Version string `json:"version"`
	} `json:"dependencies"`
}

func (n npmPackages) ListInstalled() ([]string, error) {
	if _, err := types.Cmd("npm", "--version").StdoutErr(); err != nil {
		slog.Debug("npm is not installed or not available")
		return []string{}, nil
	}

	stdout, err := types.Cmd("npm", "list", "-g", "--depth=0", "--json").StdoutErr()
	if err != nil {
		// npm list returns non-zero if there are issues, but still outputs JSON
		// So we continue if we got output
		if stdout == "" {
			return nil, err
		}
	}

	var output npmListOutput
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		return nil, fmt.Errorf("failed to parse npm list output: %w", err)
	}

	var packages []string
	for name, info := range output.Dependencies {
		packages = append(packages, name+"@"+info.Version)
	}
	return packages, nil
}

func (n npmPackages) ListExplicit() ([]string, error) {
	return n.ListInstalled()
}

func (n npmPackages) Install(pkgs []string) error {
	if _, err := types.Cmd("npm", "--version").StdoutErr(); err != nil {
		slog.Warn("npm is not installed, skipping npm package installation")
		return nil
	}

	for _, pkg := range pkgs {
		installPkg := pkg
		if _, ver := splitNpmVer(pkg); ver == "" {
			installPkg = pkg + "@latest"
		}

		slog.Info("Installing npm package", "package", installPkg)
		if err := types.Cmd("npm", "install", "-g", installPkg).Interactive().Error(); err != nil {
			return err
		}
	}
	return nil
}

func (n npmPackages) Uninstall(pkgs []string) error {
	if _, err := types.Cmd("npm", "--version").StdoutErr(); err != nil {
		return nil
	}

	for _, pkg := range pkgs {
		pkgName, _ := splitNpmVer(pkg)

		slog.Info("Uninstalling npm package", "package", pkgName)
		if err := types.Cmd("npm", "uninstall", "-g", pkgName).Interactive().Error(); err != nil {
			return err
		}
	}
	return nil
}

func (n npmPackages) MarkExplicit([]string) error                   { return nil }
func (n npmPackages) GetDependencies() (map[string][]string, error) { return nil, nil }

func (n npmPackages) SaveAsGo(wanted []string) error {
	installed, err := n.ListExplicit()
	if err != nil {
		return err
	}

	diff := subtract(n, installed, wanted)
	if len(diff) == 0 {
		logSuccess("No new npm packages to save")
		return nil
	}

	if err := saveAsGoFile("npm_packages.go", "NpmPackage", diff); err != nil {
		return err
	}
	logSuccess("npm packages saved", "file", "npm_packages.go", "count", len(diff))
	return nil
}
