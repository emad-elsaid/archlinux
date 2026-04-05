package yay

import (
	"os"
	"strings"

	"github.com/leonelquinteros/gotext"
)

var (
	// YayVersion is the version string for yay
	YayVersion = "12.5.7"
	localePath = "/usr/share/locale"
)

// InitGotext initializes the localization system
func InitGotext() {
	envLocalePath := os.Getenv("LOCALE_PATH")
	if envLocalePath != "" {
		localePath = envLocalePath
	}

	if lc := os.Getenv("LANGUAGE"); lc != "" {
		locales := strings.Split(lc, ":")
		if len(locales) > 0 && locales[0] != "" {
			gotext.Configure(localePath, locales[0], "yay")
			return
		}
	}

	if lc := os.Getenv("LC_ALL"); lc != "" {
		gotext.Configure(localePath, lc, "yay")
		return
	}

	if lc := os.Getenv("LC_MESSAGES"); lc != "" {
		gotext.Configure(localePath, lc, "yay")
		return
	}

	gotext.Configure(localePath, os.Getenv("LANG"), "yay")
}

// For internal use by other files
var yayVersion = YayVersion
