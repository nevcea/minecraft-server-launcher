# Minecraft Server Launcher

A fast and reliable Paper Minecraft server launcher written in Go.

## Features

- Automatic JAR download and update management (PaperMC)
- Smart RAM allocation based on available system memory
- Java version validation (Java 17+)
- SHA-256 checksum verification for downloaded JARs
- Automatic world backups before server start
- EULA auto-acceptance
- New launcher version notifications

## Requirements

- Java 17 or higher
- Windows, Linux, or macOS

## Installation

### Download from GitHub Releases

1. Go to the Releases page for this repository
2. Download the binary for your OS and architecture
3. Place it in your server folder and run it

### Building from Source

```bash
git clone https://github.com/nevcea/minecraft-server-launcher.git
cd minecraft-server-launcher
go build -o paper-launcher .
```

## Usage

```bash
./paper-launcher
```

On first run, a `config.yaml` is created with default settings. The launcher will:

1. Check for a newer launcher version and notify you if one is available
2. Validate your Java installation
3. Download the Paper JAR if none is found, or update it if a newer build exists
4. Perform a world backup (if `auto_backup: true`)
5. Start the server

## Configuration

`config.yaml` is generated automatically. The options most users need:

```yaml
# Minecraft version ("latest" or a specific version like "1.21.4")
minecraft_version: "latest"

# Auto-update the server JAR when a new Paper build is released
auto_update: true

# Back up world folders before starting the server
auto_backup: false

backup_worlds:
  - world
  - world_nether
  - world_the_end

# RAM in GB. Set max_ram to 0 to auto-allocate (50% of available RAM)
min_ram: 2
max_ram: 0

# Extra arguments passed to the server process
server_args:
  - nogui
```

### Advanced Options

These are not written to `config.yaml` by default but can be added manually or set via environment variables:

| Option | Env Variable | Description |
|---|---|---|
| `java_path` | `JAVA_PATH` | Path to a custom Java executable |
| `work_dir` | `WORK_DIR` | Override the working directory |
| `use_zgc` | — | Use ZGC instead of G1GC (requires Java 17+) |
| `auto_ram_percentage` | — | Percentage of available RAM to use when `max_ram` is 0 (default: 50) |
| `log_file_enable` | — | Write log output to a file |
| `log_file` | `LOG_FILE` | Log file path (default: `launcher.log`) |
| `github_token` | `LAUNCHER_GITHUB_TOKEN` | GitHub token for API access (needed only for private forks) |

### Environment Variables

| Variable | Description |
|---|---|
| `MINECRAFT_VERSION` | Override Minecraft version |
| `WORK_DIR` | Override working directory |
| `JAVA_PATH` | Override Java executable path |
| `MIN_RAM` | Override minimum RAM (GB) |
| `MAX_RAM` | Override maximum RAM (GB) |
| `LOG_FILE` | Override log file path |
| `LAUNCHER_GITHUB_TOKEN` | GitHub token |

### Command-Line Flags

```
  -c string         Config file path (default "config.yaml")
  -w string         Override working directory
  -v string         Override Minecraft version
  -log-level string Log level: trace, debug, info, warn, error (default "info")
  -verbose          Verbose logging (equivalent to -log-level debug)
  -q                Quiet mode — errors only
  -no-pause         Exit without waiting for Enter
```

## License

GPL-3.0 — see [LICENSE.md](LICENSE.md) for details.
