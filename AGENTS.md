# Agent Guide: archlinux

A declarative system configuration management framework for Arch Linux written in Go.
This library allows declaring system state (packages, services, files) and syncs the actual system to match.

## Build & Run Commands

### Standard Operations
```bash
# Run the CLI (primary usage pattern)
go run . diff        # Preview changes without applying
go run . apply       # Synchronize system to match declared configuration
go run . save        # Save current system state as Go code

# Build binary
go build             # Outputs ./archlinux binary
go build -o mysystem # Custom binary name

# Dependency management
go mod tidy          # Clean and update dependencies
go mod download      # Download dependencies without installing
go get -u ./...      # Update all dependencies
```

### Testing
**No tests currently exist** - the project has no *_test.go files.
If adding tests, follow Go conventions with testify/require (available as indirect dependency):
```bash
go test ./...                    # Run all tests
go test -v ./...                 # Verbose output
go test -run TestFuncName        # Run single test function
go test -run TestFuncName/SubTest # Run specific subtest
go test -race ./...              # Run with race detector
go test -cover ./...             # Show coverage
```

### Code Quality
```bash
gofmt -w .           # Format all Go files
go vet ./...         # Run Go's built-in linter
go mod verify        # Verify dependencies
```

## Project Structure

```
/home/emad/code/archlinux/
├── interface.go           # Core packageManager interface and callbacks
├── main.go                # Entry point, CLI commands (apply/save/diff)
├── common.go              # Shared utilities (addUnique, askYesNo, subtract, etc.)
├── dependencies.go        # Dependency checking and installation
├── logger.go              # Custom slog pretty handler
├── pacman.go              # Pacman package manager implementation
├── flatpak.go             # Flatpak application manager
├── go_packages.go         # Go package manager (go install)
├── npm_packages.go        # npm global package manager
├── ruby_gems.go           # Ruby gem manager
├── systemd.go             # Systemd service/timer/socket manager
├── system_config.go       # System configuration (timezone, locale, keyboard)
├── system_files.go        # Deploy and track system configuration files
├── system_files_state.go  # State tracking for system files
├── user_groups.go         # User group membership management
├── user_stow.go           # GNU Stow dotfile manager
├── user_symlinks.go       # Broken symlink cleanup
├── go.mod                 # Module definition (go 1.25.4)
└── example/               # Example usage directory
```

## Code Style Guidelines

### Package & Imports
- Package: `package archlinux`
- Group imports: stdlib, external, then local (with blank lines between groups)
- Use `github.com/emad-elsaid/types` for command execution (`types.Cmd()`)
- Use `github.com/samber/lo` for functional utilities (ContainsBy, Filter, Reject, Without)

### Naming Conventions
- **Variables**: Short, descriptive names (e.g., `pm`, `rn`, `wg`, `mgr`, `deps`)
- **Functions**: Descriptive camelCase (e.g., `syncPackages`, `diffPackages`, `listSystemdUnits`)
- **Public API**: PascalCase exported functions (e.g., `Package()`, `Service()`, `Main()`)
- **Constants**: PascalCase with prefix (e.g., `ResourcePackages`, `PhaseBeforeApply`)
- **Types**: Descriptive types with clear purpose (e.g., `packageManager`, `ResourceName`, `CommandPhase`)

### Code Organization
- **Keep functions short**: Break complex logic into smaller functions
- **Interface-driven design**: All resource managers implement `packageManager` interface
- **Callback system**: Use `Before()`, `After()`, `OnCommand()` for hooks
- **Two-phase execution**: Separate diff (preview) from apply (sync)

### Error Handling
- Use `checkFatal()` for unrecoverable errors during critical operations
- Use `checkWarn()` for non-critical errors that shouldn't stop execution
- Return errors from functions, let caller decide severity
- Log errors with structured logging: `slog.Error()`, `slog.Warn()`

### Logging
- Use structured logging with `log/slog`
- Custom pretty handler (`newPrettyHandler`) for user-facing output
- Log levels: `slog.Debug()`, `slog.Info()`, `slog.Warn()`, `slog.Error()`
- Include context: `slog.Info("msg", "key", value, "count", len(items))`
- Success messages: Use `logSuccess()` helper

### Comments & Documentation
- Every exported function/type must have a godoc comment
- Comments start with the name of the thing being documented
- Examples in godoc format when helpful (see `pacman.go`)
- Inline comments for complex logic, not obvious statements

### Types
- Use `any` instead of `interface{}`
- Define type aliases for clarity (e.g., `type ResourceName string`)
- Prefer explicit types over generic interfaces

### Concurrency
- Use `sync.WaitGroup` for parallel operations
- Prefer `wg.Go()` over `wg.Add(1)` + `go func()`
- Run independent package managers in parallel during diff/save
- Run package managers sequentially during apply (for Before/After callbacks)

### Testing
If adding tests:
- Use table-driven tests
- Use `testify/require` for assertions
- Test files: `*_test.go` alongside source files
- Test function naming: `func TestFunctionName(t *testing.T)`

### Dependencies
- Minimize external dependencies
- Prefer stdlib when possible
- Current deps: types, color, promptui, lo, go-version

## Architecture Patterns

### packageManager Interface
All resource managers implement this interface:
- `ResourceName()` - Human-readable identifier
- `Wanted()` - User-declared resources
- `ListInstalled()` - Currently installed resources
- `ListExplicit()` - Explicitly installed (vs dependencies)
- `Install()`, `Uninstall()`, `MarkExplicit()` - State mutations
- `Match()` - Fuzzy matching for version flexibility
- `GetDependencies()` - Dependency graph (nil if unsupported)
- `SaveAsGo()` - Generate declarative Go code

### Callback System
- `Before(ResourceName, Callback)` - Execute before resource sync
- `After(ResourceName, Callback)` - Execute after resource sync
- `OnCommand(CommandPhase, Callback)` - Execute at lifecycle phases

### Commands Flow
**diff**: BeforeDiff → parallel diff all managers → AfterDiff
**save**: BeforeSave → parallel save all managers → AfterSave
**apply**: BeforeApply → sequential sync (with Before/After per manager) → AfterApply

## Common Operations

### Adding New Resource Type
1. Define ResourceName constant
2. Define global slice for wanted items
3. Create exported function to add items (e.g., `Package()`)
4. Implement packageManager interface
5. Add to `allManagers()` in main.go

### Command Execution
Use `github.com/emad-elsaid/types`:
```go
stdout, err := types.Cmd("command", "arg1", "arg2").StdoutErr()
```

### Version Handling
For versioned resources (npm, go packages):
- Use `splitVer()` or `splitNpmVer()` to parse package@version
- Use `matchWithVersion()` for flexible matching

## Notes for Agents
- This is a library + CLI tool meant to be imported by user's Go program
- Users define configuration in `init()` functions
- Users call `fest.Main()` from their `main()` function
- The framework operates in phases: diff (preview) → apply (sync)
- State is tracked to enable cleanup of unwanted resources
- Dependencies are respected - won't remove packages that others depend on
