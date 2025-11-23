# Minecraft Server Launcher

A fast and reliable Minecraft Paper server launcher written in Go. This launcher automatically downloads Paper server JAR files, manages server configuration, and handles EULA acceptance.

## Features

- 🚀 **Automatic JAR Download**: Automatically downloads the latest Paper server JAR for your specified Minecraft version
- 💾 **Smart RAM Management**: Automatically calculates optimal RAM allocation based on system resources
- ☕ **Java Version Detection**: Verifies Java installation and version compatibility
- 📝 **EULA Handling**: Automatically accepts the Minecraft EULA
- 🔒 **File Integrity Verification**: SHA-256 checksum validation for downloaded JAR files
- ⚡ **Lightning Fast**: Compiles in 2-3 seconds with Go

## Requirements

- **Java 17 or higher** (required to run Minecraft servers)
- **Windows, Linux, or macOS**

## Installation

### Download from GitHub Releases (Recommended)

1. Go to the [Releases](https://github.com/nevcea-sub/minecraft-server-launcher/releases) page
2. Download the appropriate binary for your OS:
   - Windows: `paper-launcher-windows-amd64.exe`
   - Linux: `paper-launcher-linux-amd64` or `paper-launcher-linux-arm64`
   - macOS: `paper-launcher-darwin-amd64` or `paper-launcher-darwin-arm64`
3. Run the executable

### Building from Source

1. Install [Go](https://golang.org/dl/) (1.21 or later)

2. Clone the repository:
```bash
git clone https://github.com/nevcea-sub/minecraft-server-launcher.git
cd minecraft-server-launcher
```

3. Build the project:
```bash
go build -o paper-launcher .
```

## Usage

Run the launcher:
```bash
./paper-launcher
```

On first run, a `config.yaml` file will be created with default settings. If no Paper JAR file is found, the launcher will automatically download it.

### Configuration

Edit `config.yaml` to customize your server settings:

```yaml
minecraft_version: "latest"           # Use "latest" for the latest version
auto_update: false                    # Automatically download new Paper builds
auto_backup: true                     # Backup worlds before server start
backup_count: 10                      # Number of backups to keep
backup_worlds:                        # Worlds to backup
  - world
  - world_nether
  - world_the_end
min_ram: 2                            # Minimum RAM in GB
max_ram: 4                            # Maximum RAM in GB (0 = auto)
use_zgc: false                        # Use ZGC if available
auto_ram_percentage: 85               # Used when max_ram == 0
server_args:                          # Server arguments
  - nogui
# work_dir: "./server"                # Optional: working directory
```

### Command-Line Options

```
  -log-level string
        Log level (trace, debug, info, warn, error) (default "info")
  -verbose
        Enable verbose logging
  -q    Suppress all output except errors
  -c string
        Custom config file path (default "config.yaml")
  -w string
        Override working directory
  -v string
        Override Minecraft version
  -no-pause
        Don't pause on exit
```

### Environment Variables

Override configuration via environment variables: `MINECRAFT_VERSION`, `WORK_DIR`

```bash
export MINECRAFT_VERSION="1.21.1"
export WORK_DIR="./server"
./paper-launcher
```

## Performance

- ⚡ **Build Time**: 2-3 seconds (vs 180+ seconds with previous implementation)
- 🚀 **Startup Time**: Near-instant
- 💾 **Binary Size**: ~10MB
- 🎯 **Memory Usage**: Minimal overhead

## Why Go?

This project was rewritten from Rust to Go for the following reasons:

- **Faster compilation**: 2-3 seconds vs 3+ minutes
- **Simpler dependencies**: No heavy async runtime or TLS libraries
- **Easier maintenance**: Straightforward code without lifetime complexity
- **Fast iteration**: Rapid development and testing cycle

## License

GPL-3.0 License - See [LICENSE.md](LICENSE.md) for details

## Acknowledgments

- Uses Aikar's flags for optimal Minecraft server performance
- Built with the Paper API for server downloads
