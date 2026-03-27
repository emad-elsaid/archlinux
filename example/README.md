# Example Usage

This directory contains a minimal example of using the archlinux configuration framework.

## Running

```bash
# Preview what would change
go run . diff

# Apply configuration (requires sudo)
go run . apply

# Save current state as Go code
go run . save
```

## Configuration

Edit `main.go` to declare your desired system configuration. See the main README for all available functions.
