package archlinux

import "github.com/emad-elsaid/types"

import (
	"fmt"
	"log/slog"
	"strings"
)

// dependency represents a system tool dependency required for the framework to function.
type dependency struct {
	Name        string   // Command name to check for
	PackageName string   // Package that provides it (defaults to Name if empty)
	Required    bool     // If true, fail if missing; if false, warn only
	CheckCmd    []string // Custom command to check if installed (default: which <name>)
}

// pkg returns the package name, defaulting to Name if PackageName is empty.
func (d dependency) pkg() string {
	if d.PackageName != "" {
		return d.PackageName
	}
	return d.Name
}

// dependencies lists all system dependencies required by this framework.
var dependencies = []dependency{
	// Core package management (critical)
	{Name: "pacman", Required: true},
	{Name: "yay", PackageName: "yay", Required: true},
	{Name: "expac", PackageName: "expac", Required: true},

	// Build tools (critical for this program)
	{Name: "go", PackageName: "go", Required: true},

	// Dotfile management (critical)
	{Name: "stow", PackageName: "stow", Required: true},

	// Systemd tools (usually part of base system)
	{Name: "systemctl", Required: true},
	{Name: "timedatectl", Required: true},
	{Name: "localectl", Required: true},

	// System configuration tools
	{Name: "locale-gen", Required: true},
	{Name: "sed", Required: true},
	{Name: "sudo", Required: true},

	// File manipulation tools (usually in base system)
	{Name: "diff", PackageName: "diffutils", Required: true},
	{Name: "tee", PackageName: "coreutils", Required: true},
	{Name: "cat", PackageName: "coreutils", Required: true},
	{Name: "chmod", PackageName: "coreutils", Required: true},
	{Name: "mkdir", PackageName: "coreutils", Required: true},
	{Name: "fd", PackageName: "fd", Required: true},

	// Language runtimes (optional features)
	{Name: "ruby", PackageName: "ruby", Required: false},
	{Name: "gem", PackageName: "rubygems", Required: false},
	{Name: "flatpak", PackageName: "flatpak", Required: false},
}

// checkDependencies verifies all required dependencies are installed
// and attempts to install missing ones where possible.
// Returns an error if required dependencies cannot be installed.
func checkDependencies() error {
	var missing, optionalMissing []dependency

	for _, dep := range dependencies {
		if !isInstalled(dep) {
			if dep.Required {
				missing = append(missing, dep)
			} else {
				optionalMissing = append(optionalMissing, dep)
			}
		}
	}

	if len(optionalMissing) > 0 {
		slog.Warn("Optional dependencies not found (some features may be unavailable)",
			"missing", strings.Join(depNames(optionalMissing), ", "))
	}

	if len(missing) == 0 {
		logSuccess("All required dependencies are installed")
		return nil
	}

	slog.Info("Missing required dependencies", "count", len(missing))

	if err := installDependencies(missing); err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	if stillMissing := filterMissing(missing); len(stillMissing) > 0 {
		return fmt.Errorf("failed to install required dependencies: %s", strings.Join(stillMissing, ", "))
	}

	logSuccess("All dependencies installed successfully")
	return nil
}

// isInstalled checks if a dependency is installed using its check command or "which".
func isInstalled(dep dependency) bool {
	if len(dep.CheckCmd) > 0 {
		return types.Cmd(dep.CheckCmd[0], dep.CheckCmd[1:]...).Error() == nil
	}
	return types.Cmd("which", dep.Name).Error() == nil
}

// depNames extracts the names from a slice of dependencies.
func depNames(deps []dependency) []string {
	names := make([]string, len(deps))
	for i, dep := range deps {
		names[i] = dep.Name
	}
	return names
}

// filterMissing returns the names of dependencies that are still missing.
func filterMissing(deps []dependency) []string {
	var missing []string
	for _, dep := range deps {
		if !isInstalled(dep) {
			missing = append(missing, dep.Name)
		}
	}
	return missing
}

// installDependencies attempts to install the given dependencies using pacman or yay.
func installDependencies(deps []dependency) error {
	var pacmanPkgs, aurPkgs []string

	for _, dep := range deps {
		// Check if package exists in official repos
		if types.Cmd("pacman", "-Si", dep.pkg()).Error() == nil {
			pacmanPkgs = append(pacmanPkgs, dep.pkg())
		} else {
			aurPkgs = append(aurPkgs, dep.pkg())
		}
	}

	if len(pacmanPkgs) > 0 {
		slog.Info("Installing from official repos", "packages", strings.Join(pacmanPkgs, ", "))
		if err := types.Sudo("pacman", append([]string{"-S", "--needed", "--noconfirm"}, pacmanPkgs...)...).Interactive().Error(); err != nil {
			return fmt.Errorf("pacman install failed: %w", err)
		}
	}

	if len(aurPkgs) > 0 {
		if !isInstalled(dependency{Name: "yay"}) {
			return fmt.Errorf("yay required for AUR packages: %s", strings.Join(aurPkgs, ", "))
		}
		slog.Info("Installing from AUR", "packages", strings.Join(aurPkgs, ", "))
		if err := types.Cmd("yay", append([]string{"-S", "--needed", "--noconfirm"}, aurPkgs...)...).Interactive().Error(); err != nil {
			return fmt.Errorf("yay install failed: %w", err)
		}
	}

	return nil
}

// installYay provides instructions for installing yay AUR helper if not present.
// This is a special case since yay itself might need to be bootstrapped.
func installYay() error {
	if isInstalled(dependency{Name: "yay"}) {
		slog.Debug("yay is already installed")
		return nil
	}

	slog.Info("yay not found, installing from AUR...")
	slog.Info("This requires manual bootstrapping. Please install yay manually:")
	slog.Info("  pacman -S --needed git base-devel")
	slog.Info("  git clone https://aur.archlinux.org/yay.git")
	slog.Info("  cd yay")
	slog.Info("  makepkg -si")

	return fmt.Errorf("yay is not installed and must be installed manually before running this tool")
}

// listDependencies prints information about all dependencies and their installation status.
func listDependencies() {
	printCategory := func(title string, filter func(dependency) bool) {
		fmt.Println(title)
		for _, dep := range dependencies {
			if filter(dep) {
				status := "✗"
				if isInstalled(dep) {
					status = "✓"
				}
				fmt.Printf("  [%s] %-15s (package: %s)\n", status, dep.Name, dep.pkg())
			}
		}
	}

	printCategory("Required Dependencies:", func(d dependency) bool { return d.Required })
	printCategory("\nOptional Dependencies:", func(d dependency) bool { return !d.Required })
}
