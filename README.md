# Meadowlark 🦜

Meadowlark is a real-time messaging system built with Go, featuring WebSocket communication, end-to-end encryption support, and a modern web interface.

## Features

- Real-time messaging with WebSocket connections
- User authentication and registration
- SQLite database for user storage
- End-to-end encryption with public key infrastructure
- Modern, responsive web interface
- Chat rooms support

## Prerequisites

Before running Meadowlark, ensure you have the following installed:

1. **Go** (version 1.24.6 or later)
   - Download from: https://go.dev/dl/

2. **C Compiler** (required for SQLite support)
   - **Windows**: [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) or [MinGW-w64](https://www.mingw-w64.org/)
   - **macOS**: Xcode Command Line Tools (`xcode-select --install`)
   - **Linux**: GCC (`sudo apt-get install build-essential`)

3. **CGO Enabled**
   - CGO must be enabled to compile the SQLite driver

## Installation

1. **Clone or extract the repository**
   ```bash
   cd meadowlark-main
   ```

2. **Install Go dependencies**
   ```bash
   go mod download
   ```

## Configuration

### Enable CGO (Required for SQLite)

#### Linux/macOS:
```bash
export CGO_ENABLED=1
```

#### Windows PowerShell:
```powershell
$env:CGO_ENABLED=1
```

#### Windows Command Prompt:
```cmd
set CGO_ENABLED=1
```

### VS Code Configuration (Optional but Recommended)

Add to your `.vscode/settings.json`:
```json
{
    "go.toolsEnvVars": {
        "CGO_ENABLED": "1"
    }
}
```

## Running the Application

### 1. Start the Server

From the `meadowlark-main` directory:

```bash
go run cmd/server/main.go
```

You should see:
```
HTTP server started on :8080
Serving static files from: ./cmd/static
```

### 2. Access the Web Interface

Open your browser and navigate to:
```
http://localhost:8080
```

### 3. Run the CLI Client (Optional)

In a separate terminal:

```bash
go run cmd/client/main.go
```

## Project Structure

```
meadowlark-main/
├── cmd/
│   ├── client/          # CLI client application
│   │   └── main.go
│   ├── server/          # Server application
│   │   └── main.go
│   └── static/          # Web interface files
│       ├── index.html
│       ├── app.js
│       └── styles.css
├── internal/
│   ├── auth/            # User authentication and storage
│   │   └── auth.go
│   ├── client/          # Client connection logic
│   │   └── client.go
│   ├── protocol/        # Message protocol definitions
│   │   └── message.go
│   └── server/          # Server core logic
│       ├── server.go
│       ├── hub.go
│       └── client.go
├── go.mod
├── go.sum
└── README.md
```

## API Endpoints

The server exposes the following endpoints:

- `POST /register` - Register a new user
- `GET /keys/{username}` - Get a user's public key
- `GET /ws` - WebSocket connection endpoint
- `GET /` - Serves the web interface

## Database

Meadowlark uses SQLite for user storage. The database file (`chat.db`) is automatically created in the project root directory when the server starts.

### Database Schema

```sql
CREATE TABLE users (
    username TEXT NOT NULL PRIMARY KEY,
    hashed_password BLOB NOT NULL,
    public_key BLOB NOT NULL
);
```

## Troubleshooting

### "CGO_ENABLED=0" Error
**Problem**: `Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work`

**Solution**: Enable CGO using the commands in the Configuration section above.

### "gcc not found" Error
**Problem**: `cgo: C compiler "gcc" not found`

**Solution**: Install a C compiler (see Prerequisites section).

### 404 Not Found
**Problem**: Web interface shows 404 error

**Solution**: 
- Ensure you're running the server from the `meadowlark-main` root directory
- Verify the `cmd/static` folder exists and contains the web files
- Check that the server has been modified to serve static files (see `internal/server/server.go`)

### Port Already in Use
**Problem**: `bind: address already in use`

**Solution**: 
- Stop any other process using port 8080
- Or modify the port in `internal/server/server.go` (line with `:8080`)

## Development

### VS Code Launch Configuration

Create `.vscode/launch.json`:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Server",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/server/main.go",
            "env": {
                "CGO_ENABLED": "1"
            }
        },
        {
            "name": "Launch Client",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/client/main.go",
            "env": {
                "CGO_ENABLED": "1"
            }
        }
    ]
}
```

## Dependencies

- [gorilla/websocket](https://github.com/gorilla/websocket) - WebSocket implementation
- [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) - SQLite database driver
- [golang.org/x/crypto](https://golang.org/x/crypto) - Cryptographic functions

## License

This project is available for educational and personal use.

## Contributing

Feel free to submit issues and pull requests to improve Meadowlark!