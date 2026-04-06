package fest

import "github.com/emad-elsaid/types"

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/samber/lo"
)

const ResourceUserGroups ResourceName = "user groups"

var groups []string

// Group declares user groups that the current user should be a member of.
// The user is added to these groups using usermod -aG.
//
// Example:
//
//	fest.Group("docker", "wheel", "audio", "video")
func Group(grps ...string) { addUnique(&groups, grps...) }

type userGroups struct{}

func (u userGroups) Wanted() []string { return groups }

func (u userGroups) ResourceName() string         { return string(ResourceUserGroups) }
func (u userGroups) Match(want, have string) bool { return want == have }

func (u userGroups) getPrimaryGroup() (string, error) {
	stdout, err := types.Cmd("id", "-gn").StdoutErr()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(stdout), nil
}

func (u userGroups) ListInstalled() ([]string, error) {
	username := os.Getenv("USER")
	if username == "" {
		return nil, fmt.Errorf("USER env var not set")
	}

	cmd := types.Cmd("id", "-Gn", username)
	stdout := cmd.Stdout()
	if err := cmd.Error(); err != nil {
		return nil, err
	}

	primaryGroup, err := u.getPrimaryGroup()
	if err != nil {
		return nil, err
	}

	var grps []string
	for grp := range strings.FieldsSeq(strings.TrimSpace(stdout)) {
		if grp != primaryGroup {
			grps = append(grps, grp)
		}
	}

	return grps, nil
}

func (u userGroups) ListExplicit() ([]string, error) {
	return u.ListInstalled()
}

func (u userGroups) Install(grps []string) error {
	username := os.Getenv("USER")
	if username == "" {
		return fmt.Errorf("USER env var not set")
	}

	for _, grp := range grps {
		slog.Info("Adding user to group", "group", grp, "user", username)
		if err := types.Sudo("usermod", "-aG", grp, username).Interactive().Error(); err != nil {
			return err
		}
	}

	return nil
}

func (u userGroups) Uninstall(grps []string) error {
	username := os.Getenv("USER")
	if username == "" {
		return fmt.Errorf("USER env var not set")
	}

	for _, grp := range grps {
		slog.Info("Removing user from group", "group", grp, "user", username)
		if err := types.Sudo("gpasswd", "-d", username, grp).Interactive().Error(); err != nil {
			return err
		}
	}

	return nil
}

func (u userGroups) MarkExplicit([]string) error                   { return nil }
func (u userGroups) GetDependencies() (map[string][]string, error) { return nil, nil }

func (u userGroups) SaveAsGo(wanted []string) error {
	installed, err := u.ListInstalled()
	if err != nil {
		return err
	}

	diff := lo.Without(installed, wanted...)
	if len(diff) == 0 {
		logSuccess("No new user groups to save")
		return nil
	}

	if err := saveAsGoFile("groups.go", "Group", diff); err != nil {
		return err
	}
	logSuccess("user groups saved", "file", "groups.go", "count", len(diff))
	return nil
}
