package main

import "github.com/emad-elsaid/archlinux"

func init() {
	// System configuration
	archlinux.Timedate("America/New_York", true)
	archlinux.Locale("en_US.UTF-8 UTF-8")
	archlinux.Keyboard("us", "us", "pc105", "", "ctrl:nocaps")

	// Essential packages
	archlinux.Package(
		"base",
		"base-devel",
		"linux",
		"linux-firmware",
		"vim",
		"git",
	)

	// Development tools
	archlinux.Package(
		"docker",
		"go",
	)

	// System services
	archlinux.SystemService("docker")

	// User groups
	archlinux.Group("docker", "wheel")
}

func main() {
	archlinux.Main()
}
