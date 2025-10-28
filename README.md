# Nexus Util

A unified command-line tool for managing files and directories in Nexus OSS Raw Repository. This tool combines the functionality of three separate Python scripts (`nexus_push.py`, `nexus_pull.py`, `nexus_delete.py`) into a single, cross-platform Go application.

## Features

- **Push**: Upload files and directories to Nexus repository
- **Pull**: Download files and directories from Nexus repository  
- **Delete**: Remove files and directories from Nexus repository
- **Configuration file**: Store connection details in YAML config file
- **Cross-platform**: Builds for Linux, Windows, macOS, FreeBSD, OpenBSD, NetBSD
- **Multiple architectures**: AMD64, ARM64, ARM, 386
- **Dry run mode**: Preview operations without making changes
- **Quiet mode**: Minimal output for scripting
- **Relative paths**: Support for relative directory uploads

## Installation

### Pre-built binaries

Download the latest release for your platform from the [Releases](https://github.com/your-username/nexus-util/releases) page.

### From source

```bash
git clone https://github.com/your-username/nexus-util.git
cd nexus-util
make build
```

## Usage

### Global Flags

- `-a, --address`: Nexus OSS host address (overrides config file)
- `-r, --repository`: Nexus OSS raw repository name (overrides config file)
- `-u, --user`: User authentication login (overrides config file)
- `-p, --password`: User authentication password (overrides config file)
- `-c, --config`: Path to configuration file (default: ~/.nexus-util.yaml)
- `-q, --quiet`: Quiet mode - minimal output
- `--dry`: Dry run - show what would be done without actually doing it

### Configuration

The tool supports configuration via a YAML file to avoid specifying connection details on every command. By default, it looks for `~/.nexus-util.yaml`, but you can specify a custom path with `--config`.

**Example configuration file:**
```yaml
nexus:
  address: http://nexus.example.com
repository: myrepo
user: myuser
password: mypassword
```

**Initialize configuration:**
```bash
# Create default config file
nexus-util init --address http://nexus.example.com --user myuser --password mypass

# Create custom config file
nexus-util init --config ./my-config.yaml --address http://nexus.example.com --user myuser --password mypass

# Initialize without password (will be prompted)
nexus-util init --address http://nexus.example.com --user myuser
```

Command line flags always override configuration file values.

### Push Command

Upload files or directories to Nexus repository.

```bash
# Upload a single file
nexus-util push -a http://nexus.example.com -r myrepo -u user -p pass file.txt

# Upload a directory
nexus-util push -a http://nexus.example.com -r myrepo -u user -p pass ./localdir/

# Upload with custom destination path
nexus-util push -a http://nexus.example.com -r myrepo -u user -p pass -d custom/path file.txt

# Upload directory with relative paths
nexus-util push -a http://nexus.example.com -r myrepo -u user -p pass --relative ./localdir/

# Dry run to see what would be uploaded
nexus-util push --dry -a http://nexus.example.com -r myrepo -u user -p pass file.txt
```

**Push-specific flags:**
- `-d, --destination`: Destination path in Nexus repository
- `--relative`: Use relative paths when uploading directories

### Pull Command

Download files or directories from Nexus repository.

```bash
# Download a single file
nexus-util pull -a http://nexus.example.com -r myrepo -u user -p pass -d ./downloads file.txt

# Download a directory
nexus-util pull -a http://nexus.example.com -r myrepo -u user -p pass -d ./downloads dir/

# Download with custom root path
nexus-util pull -a http://nexus.example.com -r myrepo -u user -p pass -d ./downloads --root custom/path file.txt

# Dry run to see what would be downloaded
nexus-util pull --dry -a http://nexus.example.com -r myrepo -u user -p pass -d ./downloads file.txt
```

**Pull-specific flags:**
- `-d, --destination`: Local destination path (required)
- `--root`: Root path in Nexus repository

### Delete Command

Delete files or directories from Nexus repository.

```bash
# Delete a single file
nexus-util delete -a http://nexus.example.com -r myrepo -u user -p pass file.txt

# Delete a directory
nexus-util delete -a http://nexus.example.com -r myrepo -u user -p pass dir/

# Dry run to see what would be deleted
nexus-util delete --dry -a http://nexus.example.com -r myrepo -u user -p pass file.txt
```

### Init Command

Initialize configuration file with default values.

```bash
# Initialize with default config file location (~/.nexus-util.yaml)
nexus-util init --address http://nexus.example.com --repository myrepo --user myuser --password mypass

# Initialize with custom config file location
nexus-util init --config ./my-config.yaml --address http://nexus.example.com --repository myrepo --user myuser --password mypass

# Initialize without password (will be prompted)
nexus-util init --address http://nexus.example.com --repository myrepo --user myuser
```

**Init-specific flags:**
- `-a, --address`: Nexus OSS host address (required)
- `-r, --repository`: Nexus OSS raw repository name (required)
- `-u, --user`: User authentication login (required)
- `-p, --password`: User authentication password
- `-c, --config`: Path to configuration file (default: ~/.nexus-util.yaml)

## Examples

### Setup Configuration

```bash
# Initialize configuration file
nexus-util init --address http://nexus.example.com --repository releases --user deploy --password secret

# Now you can use commands without specifying connection details
nexus-util push file.txt
nexus-util pull -d ./downloads file.txt
nexus-util delete file.txt
```

### Upload a project to Nexus

```bash
# Using configuration file
nexus-util push -d myproject/v1.0.0 ./build/

# Override repository from config
nexus-util push -r staging -d myproject/v1.0.0 ./build/

# Upload with relative paths (only files, not directory structure)
nexus-util push --relative ./dist/
```

### Download a release

```bash
# Using configuration file
nexus-util pull -d ./downloads myproject/v1.0.0/

# Override destination repository
nexus-util pull -r staging -d ./downloads myproject/v1.0.0/

# Download latest files
nexus-util pull -d ./downloads latest/
```

### Clean up old releases

```bash
# Using configuration file
nexus-util delete myproject/v0.9.0/

# Override user for admin operations
nexus-util delete -u admin -p adminpass myproject/v0.9.0/

# Dry run to see what would be deleted
nexus-util delete --dry old-files/
```

### Using Custom Configuration Files

```bash
# Use different config for different environments
nexus-util --config ./prod-config.yaml push file.txt
nexus-util --config ./staging-config.yaml pull -d ./downloads file.txt
```

## Development

### Prerequisites

- Go 1.21 or later
- Make (optional, for using Makefile)

### Building

```bash
# Install dependencies
make deps

# Build for current platform
make build

# Build for all platforms
make build-all

# Build for specific platform
make build-linux-amd64
make build-windows-amd64
make build-darwin-amd64
make build-darwin-arm64
```

### Testing

```bash
# Run tests
make test

# Run linter
make lint

# Format code
make fmt
```

### Creating releases

```bash
# Create release packages
make release
```

## Migration from Python scripts

This Go application provides the same functionality as the original Python scripts:

| Python Script | Go Command | Notes |
|---------------|------------|-------|
| `nexus_push.py` | `nexus-util push` | Same functionality, improved error handling |
| `nexus_pull.py` | `nexus-util pull` | Same functionality, better progress reporting |
| `nexus_delete.py` | `nexus-util delete` | Same functionality, more robust file discovery |

### Command mapping

| Python | Go |
|--------|----|
| `python nexus_push.py -a ADDR -r REPO -u USER -p PASS file.txt` | `nexus-util push -a ADDR -r REPO -u USER -p PASS file.txt` |
| `python nexus_pull.py -a ADDR -r REPO -u USER -p PASS -d DEST file.txt` | `nexus-util pull -a ADDR -r REPO -u USER -p PASS -d DEST file.txt` |
| `python nexus_delete.py -a ADDR -r REPO -u USER -p PASS file.txt` | `nexus-util delete -a ADDR -r REPO -u USER -p PASS file.txt` |

## Supported Platforms

- **Linux**: AMD64, ARM64, ARM v7, ARM v6, 386
- **Windows**: AMD64, 386
- **macOS**: AMD64, ARM64 (Apple Silicon)
- **FreeBSD**: AMD64, 386
- **OpenBSD**: AMD64, 386
- **NetBSD**: AMD64, 386

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run `make test` and `make lint`
6. Submit a pull request

## Changelog

### v1.0.0
- Initial release
- Combined functionality of nexus_push.py, nexus_pull.py, and nexus_delete.py
- Cross-platform builds for multiple operating systems and architectures
- Improved error handling and user experience
- Added dry-run and quiet modes