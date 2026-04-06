package fest

import "github.com/emad-elsaid/types"

import "github.com/samber/lo"

const (
	ResourceSystemServices ResourceName = "system services"
	ResourceSystemTimers   ResourceName = "system timers"
	ResourceSystemSockets  ResourceName = "system sockets"
	ResourceUserServices   ResourceName = "user services"
	ResourceUserTimers     ResourceName = "user timers"
	ResourceUserSockets    ResourceName = "user sockets"
)

var (
	systemServices []string
	systemTimers   []string
	systemSockets  []string
	services       []string
	timers         []string
	sockets        []string
)

// SystemService declares system-level systemd services to enable.
// Services are enabled and started on apply.
//
// Example:
//
//	fest.SystemService("docker", "sshd")
func SystemService(svcs ...string) { addUnique(&systemServices, svcs...) }

// SystemTimer declares system-level systemd timers to enable.
//
// Example:
//
//	fest.SystemTimer("fstrim")
func SystemTimer(tmrs ...string) { addUnique(&systemTimers, tmrs...) }

// SystemSocket declares system-level systemd sockets to enable.
//
// Example:
//
//	fest.SystemSocket("docker")
func SystemSocket(socks ...string) { addUnique(&systemSockets, socks...) }

// Service declares user-level systemd services to enable.
// Services run as the current user.
//
// Example:
//
//	fest.Service("syncthing", "ssh-agent")
func Service(svcs ...string) { addUnique(&services, svcs...) }

// Timer declares user-level systemd timers to enable.
//
// Example:
//
//	fest.Timer("backup")
func Timer(tmrs ...string) { addUnique(&timers, tmrs...) }

// Socket declares user-level systemd sockets to enable.
//
// Example:
//
//	fest.Socket("pipewire")
func Socket(socks ...string) { addUnique(&sockets, socks...) }

type systemdManager struct {
	resource        ResourceName
	unitType        string
	user            bool
	wanted          *[]string
	funcName        string
	filename        string
	successMsg      string
	cachedInstalled []string
	cached          bool
}

func (s systemdManager) ResourceName() string         { return string(s.resource) }
func (s systemdManager) Wanted() []string             { return *s.wanted }
func (s systemdManager) Match(want, have string) bool { return want == have }

func (s *systemdManager) ListInstalled() ([]string, error) {
	if s.cached {
		return s.cachedInstalled, nil
	}
	units, err := listSystemdUnits(s.unitType, s.user)
	if err == nil {
		s.cachedInstalled = units
		s.cached = true
	}
	return units, err
}

func (s *systemdManager) ListExplicit() ([]string, error) { return s.ListInstalled() }

func (s *systemdManager) Install(units []string) error {
	var args []string
	if s.user {
		args = append(args, "--user")
	}

	args = append(args, "enable", "--now")
	for _, unit := range units {
		args = append(args, unit+"."+s.unitType)
	}

	if s.user {
		return types.Cmd("systemctl", args...).Interactive().Error()
	}

	return types.Sudo("systemctl", args...).Interactive().Error()
}

func (s *systemdManager) Uninstall(units []string) error {
	var args []string
	if s.user {
		args = append(args, "--user")
	}

	args = append(args, "disable", "--now")
	for _, unit := range units {
		args = append(args, unit+"."+s.unitType)
	}

	if s.user {
		return types.Cmd("systemctl", args...).Interactive().Error()
	}

	return types.Sudo("systemctl", args...).Interactive().Error()
}

func (s *systemdManager) MarkExplicit([]string) error                   { return nil }
func (s *systemdManager) GetDependencies() (map[string][]string, error) { return nil, nil }

func (s *systemdManager) SaveAsGo(wanted []string) error {
	installed, err := s.ListInstalled()
	if err != nil {
		return err
	}

	diff := lo.Without(installed, wanted...)
	if len(diff) == 0 {
		logSuccess("No new " + s.successMsg + " to save")
		return nil
	}

	if err := saveAsGoFile(s.filename, s.funcName, diff); err != nil {
		return err
	}
	logSuccess(s.successMsg+" saved", "file", s.filename, "count", len(diff))
	return nil
}
