package sync

import (
	"context"

	"github.com/emad-elsaid/fest/yay/pkg/completion"
	"github.com/emad-elsaid/fest/yay/pkg/db"
	"github.com/emad-elsaid/fest/yay/pkg/dep"
	"github.com/emad-elsaid/fest/yay/pkg/multierror"
	"github.com/emad-elsaid/fest/yay/pkg/runtime"
	"github.com/emad-elsaid/fest/yay/pkg/settings"
	"github.com/emad-elsaid/fest/yay/pkg/settings/parser"
	"github.com/emad-elsaid/fest/yay/pkg/sync/build"
	"github.com/emad-elsaid/fest/yay/pkg/sync/srcinfo"
	"github.com/emad-elsaid/fest/yay/pkg/sync/workdir"
	"github.com/emad-elsaid/fest/yay/pkg/text"

	"github.com/leonelquinteros/gotext"
)

type OperationService struct {
	ctx        context.Context
	cfg        *settings.Configuration
	dbExecutor db.Executor
	logger     *text.Logger
}

func NewOperationService(ctx context.Context,
	dbExecutor db.Executor,
	run *runtime.Runtime,
) *OperationService {
	return &OperationService{
		ctx:        ctx,
		cfg:        run.Cfg,
		dbExecutor: dbExecutor,
		logger:     run.Logger.Child("operation"),
	}
}

func (o *OperationService) Run(ctx context.Context, run *runtime.Runtime,
	cmdArgs *parser.Arguments,
	targets []map[string]*dep.InstallInfo, excluded []string,
) error {
	if len(targets) == 0 {
		o.logger.Println("", gotext.Get("there is nothing to do"))
		return nil
	}
	preparer := workdir.NewPreparer(o.dbExecutor, run.CmdBuilder, o.cfg, o.logger.Child("workdir"))
	installer := build.NewInstaller(o.dbExecutor, run.CmdBuilder,
		run.VCSStore, o.cfg.Mode, o.cfg.ReBuild,
		cmdArgs.ExistsArg("w", "downloadonly"), run.Logger.Child("installer"))

	pkgBuildDirs, errInstall := preparer.Run(ctx, run, targets)
	if errInstall != nil {
		return errInstall
	}

	if cleanFunc := preparer.ShouldCleanMakeDeps(run, cmdArgs); cleanFunc != nil {
		installer.AddPostInstallHook(cleanFunc)
	}

	if cleanAURDirsFunc := preparer.ShouldCleanAURDirs(run, pkgBuildDirs); cleanAURDirsFunc != nil {
		installer.AddPostInstallHook(cleanAURDirsFunc)
	}

	if completion.NeedsUpdate(o.cfg.CompletionPath, o.cfg.CompletionInterval, false) {
		go func() {
			errComp := completion.UpdateCache(ctx, run.HTTPClient, o.dbExecutor,
				o.cfg.AURURL, o.cfg.CompletionPath, o.logger)
			if errComp != nil {
				o.logger.Warnln(errComp)
			}
		}()
	}

	srcInfo, errInstall := srcinfo.NewService(o.dbExecutor, o.cfg,
		o.logger.Child("srcinfo"), run.CmdBuilder, run.VCSStore, pkgBuildDirs)
	if errInstall != nil {
		return errInstall
	}

	incompatible, errInstall := srcInfo.IncompatiblePkgs(ctx)
	if errInstall != nil {
		return errInstall
	}

	if errIncompatible := confirmIncompatible(o.logger, incompatible); errIncompatible != nil {
		return errIncompatible
	}

	if errPGP := srcInfo.CheckPGPKeys(ctx); errPGP != nil {
		return errPGP
	}

	if errInstall := installer.Install(ctx, cmdArgs, targets, pkgBuildDirs,
		excluded, o.manualConfirmRequired(cmdArgs)); errInstall != nil {
		return errInstall
	}

	var multiErr multierror.MultiError

	failedAndIgnored, err := installer.CompileFailedAndIgnored()
	if err != nil {
		multiErr.Add(err)
	}

	if !cmdArgs.ExistsArg("w", "downloadonly") {
		if err := srcInfo.UpdateVCSStore(ctx, targets, failedAndIgnored); err != nil {
			o.logger.Warnln(err)
		}
	}

	if err := installer.RunPostInstallHooks(ctx); err != nil {
		multiErr.Add(err)
	}

	return multiErr.Return()
}

func (o *OperationService) manualConfirmRequired(cmdArgs *parser.Arguments) bool {
	return (!cmdArgs.ExistsArg("u", "sysupgrade") && cmdArgs.Op != "Y") || o.cfg.DoubleConfirm
}

func confirmIncompatible(logger *text.Logger, incompatible []string) error {
	if len(incompatible) > 0 {
		logger.Warnln(gotext.Get("The following packages are not compatible with your architecture:"))

		for _, pkg := range incompatible {
			logger.Print("  " + text.Cyan(pkg))
		}

		logger.Println()

		if !logger.ContinueTask(gotext.Get("Try to build them anyway?"), true, settings.NoConfirm) {
			return &settings.ErrUserAbort{}
		}
	}

	return nil
}
