//go:build !integration
// +build !integration

package yay

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/Jguer/aur"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emad-elsaid/fest/yay/pkg/db/mock"
	mockaur "github.com/emad-elsaid/fest/yay/pkg/dep/mock"
	"github.com/emad-elsaid/fest/yay/pkg/query"
	"github.com/emad-elsaid/fest/yay/pkg/runtime"
	"github.com/emad-elsaid/fest/yay/pkg/settings"
	"github.com/emad-elsaid/fest/yay/pkg/settings/exe"
	"github.com/emad-elsaid/fest/yay/pkg/settings/parser"
	"github.com/emad-elsaid/fest/yay/pkg/text"
	"github.com/emad-elsaid/fest/yay/pkg/vcs"
)

func TestYogurtMenuAURDB(t *testing.T) {
	t.Skip("skip until Operation service is an interface")
	t.Parallel()
	makepkgBin := t.TempDir() + "/makepkg"
	pacmanBin := t.TempDir() + "/pacman"
	gitBin := t.TempDir() + "/git"
	f, err := os.OpenFile(makepkgBin, os.O_RDONLY|os.O_CREATE, 0o755)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	f, err = os.OpenFile(pacmanBin, os.O_RDONLY|os.O_CREATE, 0o755)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	f, err = os.OpenFile(gitBin, os.O_RDONLY|os.O_CREATE, 0o755)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	captureOverride := func(cmd *exec.Cmd) (stdout string, stderr string, err error) {
		return "", "", nil
	}

	showOverride := func(cmd *exec.Cmd) error {
		return nil
	}

	mockRunner := &exe.MockRunner{CaptureFn: captureOverride, ShowFn: showOverride}
	cmdBuilder := &exe.CmdBuilder{
		MakepkgBin:       makepkgBin,
		SudoBin:          "su",
		PacmanBin:        pacmanBin,
		PacmanConfigPath: "/etc/pacman.conf",
		GitBin:           "git",
		Runner:           mockRunner,
		SudoLoopEnabled:  false,
	}

	cmdArgs := parser.MakeArguments()
	cmdArgs.AddArg("Y")
	cmdArgs.AddTarget("yay")

	db := &mock.DBExecutor{
		AlpmArchitecturesFn: func() ([]string, error) {
			return []string{"x86_64"}, nil
		},
		RefreshHandleFn: func() error {
			return nil
		},
		ReposFn: func() []string {
			return []string{"aur"}
		},
		SyncPackagesFn: func(s ...string) []mock.IPackage {
			return []mock.IPackage{
				&mock.Package{
					PName:    "yay",
					PBase:    "yay",
					PVersion: "10.0.0",
					PDB:      mock.NewDB("aur"),
				},
			}
		},
		LocalPackageFn: func(s string) mock.IPackage {
			return nil
		},
	}
	aurCache := &mockaur.MockAUR{
		GetFn: func(ctx context.Context, query *aur.Query) ([]aur.Pkg, error) {
			return []aur.Pkg{
				{
					Name:        "yay",
					PackageBase: "yay",
					Version:     "10.0.0",
				},
			}, nil
		},
	}
	logger := text.NewLogger(io.Discard, os.Stderr, strings.NewReader("1\n"), true, "test")

	run := &runtime.Runtime{
		Cfg: &settings.Configuration{
			RemoveMake: "no",
		},
		Logger:     logger,
		CmdBuilder: cmdBuilder,
		VCSStore:   &vcs.Mock{},
		QueryBuilder: query.NewSourceQueryBuilder(aurCache, logger, "votes", parser.ModeAny, "name",
			true, false, true),
		AURClient: aurCache,
	}
	err = handleCmd(context.Background(), run, cmdArgs, db)
	require.NoError(t, err)

	wantCapture := []string{}
	wantShow := []string{
		"pacman -S -y --config /etc/pacman.conf --",
		"pacman -S -y -u --config /etc/pacman.conf --",
	}

	require.Len(t, mockRunner.ShowCalls, len(wantShow))
	require.Len(t, mockRunner.CaptureCalls, len(wantCapture))

	for i, call := range mockRunner.ShowCalls {
		show := call.Args[0].(*exec.Cmd).String()
		show = strings.ReplaceAll(show, makepkgBin, "makepkg")
		show = strings.ReplaceAll(show, pacmanBin, "pacman")
		show = strings.ReplaceAll(show, gitBin, "pacman")

		// options are in a different order on different systems and on CI root user is used
		assert.Subset(t, strings.Split(show, " "), strings.Split(wantShow[i], " "), fmt.Sprintf("%d - %s", i, show))
	}
}
