# Matrix Archive - Professional Chat History Management

A comprehensive tool for importing, exporting, and managing Matrix chat histories with enterprise-grade features.

A professional Go application using idiomatic project structure and the [mautrix-go](https://github.com/mautrix/go) library for Matrix client functionality with end-to-end encryption support.

## ✨ Key Features

- **End-to-End Encryption Support**: Full E2EE message decryption using Matrix key backup
- **Professional Export Formats**: HTML, JSON, YAML, and TXT with rich metadata
- **Advanced Username Mapping**: Intelligent bridge user resolution for Discord/Telegram bridges  
- **Secure Authentication**: Beeper OAuth integration with credential management
- **Rich Template System**: Beautiful, responsive HTML exports with professional styling
- **Comprehensive CLI**: Intuitive command grouping with `auth`, `crypto`, and `media` subcommands

Originally developed at [Dinacon 2018](https://www.dinacon.org) for use by the documentation team.

⚠️ **Use this responsibly and ethically.** Don't re-publish people's messages without their knowledge and consent.

## Project Structure

```go
matrix-archive/
├── cmd/matrix-archive/     # Main application entry point
├── lib/                    # Core library code (reusable)
├── internal/beeperapi/     # Beeper API client (internal use only)
├── templates/              # Export templates
├── go.mod                  # Go module definition
├── go.sum                  # Go module checksums
├── Makefile               # Build and test automation
└── README.md              # This file
```

This follows Go's standard project layout conventions for professional applications.

## Prerequisites

- Go 1.21 or later

## Installation

### From Source

```bash
git clone https://github.com/osteele/matrix-archive.git
cd matrix-archive
go build ./cmd/matrix-archive
```

### Build and Install

```bash
make build          # Build the binary
make install        # Install to $GOPATH/bin
make test           # Run the test suite
make coverage       # Run tests with coverage
```

## Quick Start

### 1. Authentication
```bash
# Authenticate with Beeper
./matrix-archive auth login

# Authenticate with specific domain  
./matrix-archive auth login --domain beeper.com

# Clear credentials
./matrix-archive auth logout
```

### 2. Import Messages
```bash
# Import from all rooms
./matrix-archive import

# Import from specific room with limit
./matrix-archive import --room-id "!room:example.com" --limit 1000
```

### 3. Export Archives  
```bash
# Export to HTML (recommended)
./matrix-archive export archive.html --room-id "!room:example.com"

# Export to different formats
./matrix-archive export data.json    # JSON format
./matrix-archive export data.yaml    # YAML format  
./matrix-archive export data.txt     # Plain text
```

### 4. Key Recovery (for E2EE rooms)
```bash
# Recover encryption keys from backup
./matrix-archive crypto recover-keys --recovery-key "your-recovery-key"

# Recover keys for specific room
./matrix-archive crypto recover-keys --recovery-key "key" --room-id "!room:example.com"
```

### 5. Media Download
```bash
# Download media files
./matrix-archive media download

# Download thumbnails only
./matrix-archive media download --thumbnails
```

## Detailed Usage

### Authentication

```bash
./matrix-archive auth login [--domain beeper.com]
```

Authenticates with Beeper using email and verification code. This will:
- Prompt for your Beeper email address
- Send a login code to your email
- Save your credentials for future use

To clear saved credentials:

```bash
./matrix-archive auth logout [--domain beeper.com]
```

### List Rooms

```bash
./matrix-archive list [pattern]
```

Lists all Matrix rooms that you have access to, optionally filtered by a regex pattern matching the room name.

### Import Messages

```bash
# Import from all joined rooms
./matrix-archive import

# Import from a specific room
./matrix-archive import --room-id \"!roomid:matrix.org\"

# Import with a message limit
./matrix-archive import --limit 1000
```

Imports messages from Matrix rooms into DuckDB for archival. If no room ID is specified, imports from all joined rooms.

Options:

- `--room-id ROOM_ID`: Import from a specific room (optional, imports all joined rooms if not specified)
- `--limit N`: Limit the number of messages to import (optional)

### Export Messages

```bash
./matrix-archive export [filename] [--room-id ROOM_ID] [--local-images]
```

Exports messages from the database to various formats based on file extension:
- `.html`: HTML format
- `.txt`: Plain text format  
- `.json`: JSON format
- `.yaml`: YAML format

Options:
- `--room-id ROOM_ID`: Export from a specific room (optional, defaults to first configured room)
- `--local-images`: Use local image paths instead of Matrix URLs (default: true)

Examples:
```bash
./matrix-archive export archive.html
./matrix-archive export messages.json --room-id '!roomid:matrix.org'
./matrix-archive export chat.txt --no-local-images
```

### Download Images

```bash
./matrix-archive download-images [output-dir] [--thumbnails]
```

Downloads all images referenced in messages to a local directory.

Options:
- `--thumbnails`: Download thumbnails instead of full images (default: true)
- `--no-thumbnails`: Download full-size images

Examples:
```bash
./matrix-archive download-images                    # Downloads thumbnails to ./thumbnails/
./matrix-archive download-images --no-thumbnails    # Downloads full images to ./images/
./matrix-archive download-images my-images          # Downloads thumbnails to ./my-images/
```

## Templates

Export templates are located in the `templates/` directory:
- `templates/default.html.tpl`: HTML export template
- `templates/default.txt.tpl`: Text export template

You can modify these templates to customize the export format.

## Dependencies

- [mautrix/go](https://github.com/mautrix/go): Matrix client library
- [spf13/cobra](https://github.com/spf13/cobra): CLI framework
- [DuckDB Go Driver](https://github.com/marcboeker/go-duckdb): DuckDB database driver
- [joho/godotenv](https://github.com/joho/godotenv): Environment variable loading

## Differences from Python Version

- Uses the mautrix/go library instead of matrix_client
- DuckDB operations use the official Go driver instead of mongoengine
- CLI built with Cobra instead of Click
- Template rendering uses Go's html/template instead of Jinja2
- Error handling follows Go conventions

## References

- [Matrix Client-Server API](https://matrix.org/docs/spec/r0.0.0/client_server.html)
- [mautrix/go Documentation](https://docs.mau.fi/go/index.html)

## License

MIT
