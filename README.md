# terminal-track (`tt`)

A transparent shell history recorder for zsh. Every command you type — across all terminals and tmux sessions — is silently captured with its timestamp, working directory, and exit code. Browse your full history through a CLI search or a web-based vertical timeline.

## How It Works

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  zsh         │     │  zsh         │     │  tmux pane   │
│  (terminal 1)│     │  (terminal 2)│     │  (terminal 3)│
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       └───────────┬───────┴───────────────────┘
                   │  preexec / precmd hooks
                   ▼
            tt record (background)
                   │
                   ▼
            SQLite (WAL mode)
            ~/.terminal-track/history.db
                   ▲
                   │
           ┌───────┴───────┐
           │  tt serve      │
           │  tt search     │
           └───────────────┘
```

A small zsh hook is sourced from your `.zshrc`. It uses two built-in zsh mechanisms:

- **`preexec`** — fires just before a command runs. Captures the command text, `$PWD`, and a timestamp.
- **`precmd`** — fires after a command finishes, before the next prompt. Captures `$?` (exit code).

The hook calls `tt record` in the background (`&!`), so there is **zero perceptible delay** to your prompt. Each shell session gets a unique session ID, so concurrent terminals and tmux panes never conflict.

### Why not zsh's built-in `HISTFILE`?

- `HISTFILE` doesn't store the working directory or exit code.
- Sharing history across concurrent sessions is fragile and lossy.
- No structured query capability.

### Storage

All data lives in a single SQLite database at `~/.terminal-track/history.db`. WAL (Write-Ahead Logging) mode is enabled so multiple shells can write concurrently without locking issues.

**Schema:**

| Column     | Type    | Description                    |
|------------|---------|--------------------------------|
| id         | INTEGER | Auto-incrementing primary key  |
| timestamp  | TEXT    | RFC3339 UTC timestamp          |
| command    | TEXT    | The command that was executed   |
| directory  | TEXT    | Working directory at execution |
| exit_code  | INTEGER | Exit code (nullable)           |
| session_id | TEXT    | Unique per-shell session ID    |
| hostname   | TEXT    | Machine hostname               |

### Tech Stack

- **Go** — single static binary, no runtime dependencies
- **SQLite** via `modernc.org/sqlite` — pure Go, no CGO required
- **Cobra** — CLI framework
- **Embedded HTML** — the web UI is compiled into the binary via `go:embed`

## Installation

### From Source

Requires Go 1.21+.

```bash
git clone https://github.com/greatbody/terminal-track.git
cd terminal-track
go build -o tt .
```

Move the binary somewhere on your `$PATH`:

```bash
# Option A: system-wide
sudo cp tt /usr/local/bin/

# Option B: user-local
mkdir -p ~/bin
cp tt ~/bin/
# Make sure ~/bin is in your PATH:
# export PATH="$HOME/bin:$PATH"  (add to .zshrc if needed)
```

### Install the Zsh Hook

```bash
tt install
```

This does two things:

1. Writes the hook script to `~/.terminal-track/tt-hook.zsh`
2. Appends a source line to your `~/.zshrc`

Then restart your shell or run:

```bash
source ~/.zshrc
```

That's it. Every new zsh session (including tmux panes) will automatically record commands.

### Uninstall

```bash
tt install --uninstall
```

This removes the hook from `.zshrc`. You can also delete the database and hook directory:

```bash
rm -rf ~/.terminal-track
```

## Usage

### Automatic Recording

Once installed, there's nothing to do. Commands are recorded transparently. You can verify it's working:

```bash
echo "hello"
tt search "hello"
```

### Search from CLI

```bash
# Show recent commands (default: last 50)
tt search

# Search by pattern
tt search "docker"

# Filter by directory
tt search --dir /path/to/project

# Filter by time
tt search --since 1h      # last hour
tt search --since 7d      # last 7 days
tt search --since 2024-01-01

# Combine filters
tt search "git" --dir ~/code/myproject --since 1d -n 20
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--dir` | `-d` | Filter by working directory |
| `--limit` | `-n` | Max number of results (default 50) |
| `--since` | | Time filter: duration (`1h`, `30m`, `7d`, `2w`) or date (`2024-01-01`) |

### Web Timeline

```bash
tt serve
```

Opens a web UI at `http://localhost:8080` with a dark-themed vertical timeline showing all your commands. Features:

- Search bar with live filtering
- Color-coded exit status (green dot = success, red dot = failure)
- Date separators grouping commands by day
- Working directory shown for each command
- "Load more" pagination
- JSON API at `/api/commands` for programmatic access

Specify a custom port:

```bash
tt serve --port 3000
```

### All Commands

| Command | Description |
|---------|-------------|
| `tt search [pattern]` | Search command history from the terminal |
| `tt serve` | Start the web timeline UI |
| `tt install` | Install the zsh hook into `.zshrc` |
| `tt install --uninstall` | Remove the zsh hook from `.zshrc` |
| `tt record` | Record a command (called by the hook, not meant for manual use) |

## API

When running `tt serve`, a JSON API is available:

```
GET /api/commands?q=docker&dir=/path&limit=50&offset=0&since=2024-01-01T00:00:00Z
```

Response:

```json
{
  "records": [
    {
      "ID": 42,
      "Timestamp": "2024-03-06T10:15:30Z",
      "Command": "docker compose up -d",
      "Directory": "/home/user/project",
      "ExitCode": 0,
      "SessionID": "abc123",
      "Hostname": "my-machine"
    }
  ],
  "total": 1
}
```

## Project Structure

```
terminal-track/
├── main.go                        # Entry point
├── cmd/
│   ├── root.go                    # Root cobra command
│   ├── record.go                  # tt record (called by zsh hook)
│   ├── install.go                 # tt install / --uninstall
│   ├── search.go                  # tt search [pattern]
│   └── serve.go                   # tt serve --port 8080
├── internal/
│   ├── db/
│   │   └── db.go                  # SQLite: schema, open, insert, query
│   ├── hook/
│   │   ├── hook.go                # Embeds the zsh script via go:embed
│   │   └── tt.zsh                 # Zsh preexec/precmd hook
│   └── web/
│       ├── server.go              # HTTP server + JSON API
│       └── templates/
│           └── index.html         # Timeline web UI (embedded)
├── go.mod
├── go.sum
└── .gitignore
```

## License

MIT
