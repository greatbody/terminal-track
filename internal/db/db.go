package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS commands (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	timestamp  TEXT    NOT NULL,
	command    TEXT    NOT NULL,
	directory  TEXT    NOT NULL,
	exit_code  INTEGER,
	session_id TEXT,
	hostname   TEXT,
	tty        TEXT,
	terminal   TEXT,
	tmux_pane  TEXT
);

CREATE INDEX IF NOT EXISTS idx_commands_timestamp  ON commands(timestamp);
CREATE INDEX IF NOT EXISTS idx_commands_directory  ON commands(directory);
CREATE INDEX IF NOT EXISTS idx_commands_command    ON commands(command);
CREATE INDEX IF NOT EXISTS idx_commands_session_id ON commands(session_id);
`

// migrations adds columns that may not exist in older databases.
const migrations = `
ALTER TABLE commands ADD COLUMN tty       TEXT;
ALTER TABLE commands ADD COLUMN terminal  TEXT;
ALTER TABLE commands ADD COLUMN tmux_pane TEXT;
`

// Record represents a single recorded command.
type Record struct {
	ID        int64
	Timestamp time.Time
	Command   string
	Directory string
	ExitCode  *int
	SessionID string
	Hostname  string
	TTY       string
	Terminal  string
	TmuxPane  string
}

// DB wraps the SQLite database connection.
type DB struct {
	conn *sql.DB
}

// DefaultDBPath returns ~/.terminal-track/history.db
func DefaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".terminal-track", "history.db"), nil
}

// Open opens (or creates) the database at the given path.
func Open(path string) (*DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	conn, err := sql.Open("sqlite", path+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if _, err := conn.Exec(schema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	// Run migrations — ignore errors for columns that already exist
	for _, stmt := range strings.Split(migrations, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			conn.Exec(stmt)
		}
	}

	return &DB{conn: conn}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.conn.Close()
}

// Insert records a new command entry.
func (d *DB) Insert(r Record) error {
	_, err := d.conn.Exec(
		`INSERT INTO commands (timestamp, command, directory, exit_code, session_id, hostname, tty, terminal, tmux_pane)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.Timestamp.UTC().Format(time.RFC3339Nano),
		r.Command,
		r.Directory,
		r.ExitCode,
		r.SessionID,
		r.Hostname,
		r.TTY,
		r.Terminal,
		r.TmuxPane,
	)
	return err
}

// QueryOptions defines filtering for queries.
type QueryOptions struct {
	Limit     int
	Offset    int
	Search    string
	Directory string
	Session   string
	Since     *time.Time
	Until     *time.Time
}

// Query returns records matching the given options, newest first.
func (d *DB) Query(opts QueryOptions) ([]Record, error) {
	query := `SELECT id, timestamp, command, directory, exit_code, session_id, hostname, tty, terminal, tmux_pane FROM commands WHERE 1=1`
	args := []interface{}{}

	if opts.Search != "" {
		query += ` AND command LIKE ?`
		args = append(args, "%"+opts.Search+"%")
	}
	if opts.Directory != "" {
		query += ` AND directory = ?`
		args = append(args, opts.Directory)
	}
	if opts.Session != "" {
		query += ` AND session_id = ?`
		args = append(args, opts.Session)
	}
	if opts.Since != nil {
		query += ` AND timestamp >= ?`
		args = append(args, opts.Since.UTC().Format(time.RFC3339Nano))
	}
	if opts.Until != nil {
		query += ` AND timestamp <= ?`
		args = append(args, opts.Until.UTC().Format(time.RFC3339Nano))
	}

	query += ` ORDER BY timestamp DESC`

	if opts.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, opts.Limit)
	}
	if opts.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, opts.Offset)
	}

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var r Record
		var ts string
		var exitCode sql.NullInt64
		var tty, terminal, tmuxPane sql.NullString
		if err := rows.Scan(&r.ID, &ts, &r.Command, &r.Directory, &exitCode, &r.SessionID, &r.Hostname, &tty, &terminal, &tmuxPane); err != nil {
			return nil, err
		}
		r.Timestamp, _ = time.Parse(time.RFC3339Nano, ts)
		if exitCode.Valid {
			v := int(exitCode.Int64)
			r.ExitCode = &v
		}
		if tty.Valid {
			r.TTY = tty.String
		}
		if terminal.Valid {
			r.Terminal = terminal.String
		}
		if tmuxPane.Valid {
			r.TmuxPane = tmuxPane.String
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// Count returns the total number of records matching the options.
func (d *DB) Count(opts QueryOptions) (int, error) {
	query := `SELECT COUNT(*) FROM commands WHERE 1=1`
	args := []interface{}{}

	if opts.Search != "" {
		query += ` AND command LIKE ?`
		args = append(args, "%"+opts.Search+"%")
	}
	if opts.Directory != "" {
		query += ` AND directory = ?`
		args = append(args, opts.Directory)
	}
	if opts.Session != "" {
		query += ` AND session_id = ?`
		args = append(args, opts.Session)
	}
	if opts.Since != nil {
		query += ` AND timestamp >= ?`
		args = append(args, opts.Since.UTC().Format(time.RFC3339Nano))
	}
	if opts.Until != nil {
		query += ` AND timestamp <= ?`
		args = append(args, opts.Until.UTC().Format(time.RFC3339Nano))
	}

	var count int
	err := d.conn.QueryRow(query, args...).Scan(&count)
	return count, err
}
