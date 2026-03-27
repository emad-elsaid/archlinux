package archlinux

import "github.com/emad-elsaid/types"

import (
	"fmt"
	"log/slog"
	"strings"
)

// systemConfigType holds system configuration settings
type systemConfigType struct {
	// Timedate
	Timezone string
	NTP      bool

	// Locale
	Locale string

	// Keyboard
	Keymap  string
	Layout  string
	Model   string
	Variant string
	Options string
}

var systemConfig = systemConfigType{}

// Timedate sets timezone and NTP configuration.
// This configures the system timezone and enables/disables NTP time synchronization.
//
// Example:
//
//	archlinux.Timedate("America/New_York", true)  // Set timezone and enable NTP
func Timedate(timezone string, ntp bool) {
	systemConfig.Timezone = timezone
	systemConfig.NTP = ntp
}

// Locale sets the system locale.
// The locale string should match an entry in /etc/locale.gen.
//
// Example:
//
//	archlinux.Locale("en_US.UTF-8 UTF-8")
func Locale(locale string) {
	systemConfig.Locale = locale
}

// Keyboard sets keyboard configuration for both console and X11.
// Parameters:
//   - keymap: Console keymap (e.g., "us")
//   - layout: X11 keyboard layout (e.g., "us,ara" for multiple layouts)
//   - model: Keyboard model (e.g., "pc105")
//   - variant: Layout variant (e.g., "dvorak")
//   - options: Keyboard options (e.g., "ctrl:nocaps,grp:alt_shift_toggle")
//
// Example:
//
//	archlinux.Keyboard("us", "us", "pc105", "", "ctrl:nocaps")
func Keyboard(keymap, layout, model, variant, options string) {
	systemConfig.Keymap = keymap
	systemConfig.Layout = layout
	systemConfig.Model = model
	systemConfig.Variant = variant
	systemConfig.Options = options
}

type systemConfigManager struct{}

func (s systemConfigManager) ResourceName() string { return "system-config" }
func (s systemConfigManager) Wanted() []string {
	var wanted []string
	if systemConfig.Timezone != "" {
		wanted = append(wanted, "timezone:"+systemConfig.Timezone)
	}
	if systemConfig.NTP {
		wanted = append(wanted, "ntp:enabled")
	}
	if systemConfig.Locale != "" {
		wanted = append(wanted, "locale:"+systemConfig.Locale)
	}
	if systemConfig.Keymap != "" || systemConfig.Layout != "" {
		wanted = append(wanted, "keyboard:configured")
	}
	return wanted
}
func (s systemConfigManager) Match(want, have string) bool { return want == have }
func (s systemConfigManager) ListInstalled() ([]string, error) {
	var installed []string

	// Check timezone
	if systemConfig.Timezone != "" {
		output, err := types.Cmd("timedatectl", "show", "-p", "Timezone", "--value").StdoutErr()
		if err == nil && strings.TrimSpace(output) == systemConfig.Timezone {
			installed = append(installed, "timezone:"+systemConfig.Timezone)
		}
	}

	// Check NTP
	if systemConfig.NTP {
		output, err := types.Cmd("timedatectl", "show", "-p", "NTP", "--value").StdoutErr()
		if err == nil && strings.TrimSpace(output) == "yes" {
			installed = append(installed, "ntp:enabled")
		}
	}

	// Check locale
	if systemConfig.Locale != "" {
		output, err := types.Cmd("localectl", "status").StdoutErr()
		if err == nil && strings.Contains(output, "LANG="+systemConfig.Locale) {
			installed = append(installed, "locale:"+systemConfig.Locale)
		}
	}

	// Check keyboard (simplified check)
	if systemConfig.Keymap != "" || systemConfig.Layout != "" {
		// Just mark as installed if any keyboard config exists
		output, err := types.Cmd("localectl", "status").StdoutErr()
		if err == nil && strings.Contains(output, "Keymap:") {
			installed = append(installed, "keyboard:configured")
		}
	}

	return installed, nil
}
func (s systemConfigManager) ListExplicit() ([]string, error) { return s.ListInstalled() }
func (s systemConfigManager) Install(configs []string) error {
	for _, cfg := range configs {
		parts := strings.SplitN(cfg, ":", 2)
		if len(parts) != 2 {
			continue
		}

		switch parts[0] {
		case "timezone":
			slog.Info("Setting timezone", "timezone", parts[1])
			if err := types.Sudo("timedatectl", "set-timezone", parts[1]).Interactive().Error(); err != nil {
				return err
			}
		case "ntp":
			slog.Info("Enabling NTP")
			if err := types.Sudo("timedatectl", "set-ntp", "true").Interactive().Error(); err != nil {
				return err
			}
		case "locale":
			slog.Info("Setting locale", "locale", parts[1])
			// Ensure locale is generated
			if err := types.Sudo("sed", "-i", fmt.Sprintf("s/^#%s/%s/", parts[1], parts[1]), "/etc/locale.gen").Interactive().Error(); err != nil {
				slog.Warn("Failed to uncomment locale in /etc/locale.gen", "error", err)
			}
			if err := types.Sudo("locale-gen").Interactive().Error(); err != nil {
				return err
			}
			if err := types.Sudo("localectl", "set-locale", "LANG="+parts[1]).Interactive().Error(); err != nil {
				return err
			}
		case "keyboard":
			slog.Info("Setting keyboard configuration")
			keymapArgs := []string{"set-keymap"}
			if systemConfig.Keymap != "" {
				keymapArgs = append(keymapArgs, systemConfig.Keymap)
			} else {
				keymapArgs = append(keymapArgs, "us")
			}
			if err := types.Sudo("localectl", keymapArgs...).Interactive().Error(); err != nil {
				return err
			}

			// Set X11 keyboard layout if specified
			if systemConfig.Layout != "" {
				x11Args := []string{"set-x11-keymap", systemConfig.Layout}
				if systemConfig.Model != "" {
					x11Args = append(x11Args, systemConfig.Model)
				} else {
					x11Args = append(x11Args, "")
				}
				if systemConfig.Variant != "" {
					x11Args = append(x11Args, systemConfig.Variant)
				} else {
					x11Args = append(x11Args, "")
				}
				if systemConfig.Options != "" {
					x11Args = append(x11Args, systemConfig.Options)
				}
				if err := types.Sudo("localectl", x11Args...).Interactive().Error(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
func (s systemConfigManager) Uninstall(configs []string) error {
	slog.Warn("Skipping uninstall for system config", "count", len(configs))
	return nil
}
func (s systemConfigManager) MarkExplicit(configs []string) error { return nil }
func (s systemConfigManager) GetDependencies() (map[string][]string, error) {
	return make(map[string][]string), nil
}
func (s systemConfigManager) SaveAsGo(wanted []string) error { return nil }
