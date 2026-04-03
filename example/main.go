package main

import "github.com/emad-elsaid/fest"

func init() {
	// System configuration
	fest.Timedate("America/New_York", true)
	fest.Locale("en_US.UTF-8 UTF-8")
	fest.Keyboard("us", "us", "pc105", "", "ctrl:nocaps")

	// Essential packages
	fest.Package(
		"base",
		"base-devel",
		"linux",
		"linux-firmware",
		"vim",
		"git",
	)

	// Development tools
	fest.Package(
		"docker",
		"go",
	)

	// System services
	fest.SystemService("docker")

	// User groups
	fest.Group("docker", "wheel")
}

func main() {
	fest.Main()
}
