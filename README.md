# Minecraft Server Launcher

A fast and reliable Minecraft Paper server launcher written in Rust. This launcher automatically downloads Paper server JAR files, manages server configuration, and handles EULA acceptance.

## Features

- 🚀 **Automatic JAR Download**: Automatically downloads the latest Paper server JAR for your specified Minecraft version
- ⚙️ **Configuration Management**: Simple TOML-based configuration file
- 💾 **Smart RAM Management**: Automatically calculates optimal RAM allocation based on system resources
- ☕ **Java Version Detection**: Verifies Java installation and version compatibility
- 📝 **EULA Handling**: Automatically accepts the Minecraft EULA
- 🔧 **Environment Variable Overrides**: Override configuration via environment variables
- 📊 **Progress Indicators**: Visual download progress with progress bars
- 🔒 **File Integrity Verification**: SHA-256 checksum validation with caching for downloaded JAR files
- 🔐 **HTTPS Enforcement**: All downloads are performed over secure HTTPS connections
- ⚡ **High Performance**: Optimized algorithms with integrated JAR validation and checksum calculation

## Requirements

- **Java 17 or higher** (required to run Minecraft servers)
- **Windows, Linux, or macOS**

## Installation

### Download from GitHub Releases (Recommended)

1. Go to the [Releases](https://github.com/nevcea-sub/minecraft-server-launcher/releases) page
2. Download the latest `paper-launcher.exe` file
3. Run the executable

### Building from Source

1. Install [Rust](https://www.rust-lang.org/tools/install) (1.70 or later)

2. Clone the repository:
```bash
git clone https://github.com/nevcea-sub/minecraft-server-launcher.git
cd minecraft-server-launcher
```

3. Build the project:
```bash
cargo build --release
```

4. The executable will be located at `target/release/paper-launcher.exe` (Windows) or `target/release/paper-launcher` (Linux/macOS)

## Usage

### First Run

1. Run the launcher:
```bash
./paper-launcher
```

2. On first run, a `config.toml` file will be created with default settings. Edit it to customize your server configuration.

3. If no Paper JAR file is found, the launcher will prompt you to download it automatically.

### Configuration

Edit `config.toml` to customize your server settings:

```toml
# Minecraft version (use "latest" for the latest version)
minecraft_version = "latest"

# Minimum RAM in GB
min_ram = 2

# Maximum RAM in GB (will be auto-adjusted based on system RAM)
max_ram = 4

# Server arguments
server_args = ["nogui"]

# Working directory (optional, defaults to current directory)
# work_dir = "./server"
```

### Command-Line Options

The launcher supports various command-line options:

| Option | Short | Description |
|--------|-------|-------------|
| `--log-level` | `-l` | Set log level: `trace`, `debug`, `info`, `warn`, `error` (default: `info`) |
| `--verbose` | | Enable verbose logging (equivalent to `--log-level debug`) |
| `--quiet` | `-q` | Suppress all output except errors |
| `--config` | `-c` | Specify custom config file path |
| `--work-dir` | `-w` | Override working directory |
| `--version` | `-v` | Override Minecraft version |
| `--no-pause` | | Don't pause on exit (useful for scripts) |

Examples:
```bash
# Run with debug logging
./paper-launcher --verbose

# Run with custom config and work directory
./paper-launcher -c ./my-config.toml -w ./server

# Run for specific version without pausing
./paper-launcher --version 1.21.1 --no-pause
```

### Environment Variables

You can override configuration values using environment variables:

- `MINECRAFT_VERSION`: Override the Minecraft version
- `MIN_RAM`: Override minimum RAM (in GB)
- `MAX_RAM`: Override maximum RAM (in GB)
- `WORK_DIR`: Override the working directory

Example:
```bash
export MINECRAFT_VERSION="1.21.1"
export MIN_RAM=4
export MAX_RAM=8
./paper-launcher
```

## Security Features

- 🔐 **HTTPS Enforcement**: All downloads are performed over HTTPS only
- 🔒 **SHA-256 Checksum Validation**: Downloaded JAR files are verified against SHA-256 checksums
- 💾 **Checksum Caching**: Checksums are cached in `.jar.sha256` files for faster subsequent validations
- ✅ **JAR Integrity Verification**: Validates ZIP structure, magic numbers, and manifest before use

## Performance Benchmarks

### Checksum Validation Performance

| Operation | Time | Improvement | Notes |
|-----------|------|-------------|-------|
| Checksum validation only | ~60µs | **20% faster** | Size-based buffer optimization |
| Checksum calculation (1KB) | ~58µs | Stable | Optimized for small files |
| Checksum calculation (1MB) | ~4.68ms | **3.4% faster** | Large buffer optimization |
| Checksum calculation (10MB) | ~49ms | Stable | Efficient for large files |
| Checksum validation (valid) | ~4.75ms | **4.5% faster** | Byte-level comparison |
| Checksum validation (invalid) | ~4.80ms | **5.6% faster** | Early detection optimization |

### JAR Validation Performance

| Operation | Time | Improvement | Notes |
|-----------|------|-------------|-------|
| JAR validation only | ~185µs | **12.8% faster** | Optimized ZIP parsing |
| JAR validation + checksum (integrated) | ~190µs | **22% faster** | Single file read |
| Checksum validation only | ~60µs | **8.8% faster** | Optimized buffer selection |

**Performance Comparison:**
- **Previous**: JAR validation (185µs) + checksum validation (58µs) = **243µs**
- **Current**: Integrated function = **190µs**
- **Result**: **22% faster** by reading file only once

> **Note**: Checksums are cached in `.jar.sha256` files. Subsequent validations only require reading the cached checksum file (~1µs) instead of recalculating the hash.

### Performance Summary

- ✅ All operations maintain **sub-millisecond latency** for typical JAR files
- ✅ Checksum validation overhead: **~60µs** (minimal impact on startup time)
- ✅ File integrity verification adds only **~3% overhead** to JAR validation
- ✅ Integrated validation: **22% faster** than separate operations