package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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
	hostname   TEXT
);

CREATE INDEX IF NOT EXISTS idx_commands_timestamp ON commands(timestamp);
CREATE INDEX IF NOT EXISTS idx_commands_directory ON commands(directory);
CREATE INDEX IF NOT EXISTS idx_commands_command   ON commands(command);
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

	return &DB{conn: conn}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.conn.Close()
}

// Insert records a new command entry.
func (d *DB) Insert(r Record) error {
	_, err := d.conn.Exec(
		`INSERT INTO commands (timestamp, command, directory, exit_code, session_id, hostname)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		r.Timestamp.UTC().Format(time.RFC3339Nano),
		r.Command,
		r.Directory,
		r.ExitCode,
		r.SessionID,
		r.Hostname,
	)
	return err
}

// QueryOptions defines filtering for queries.
type QueryOptions struct {
	Limit     int
	Offset    int
	Search    string
	Directory string
	Since     *time.Time
	Until     *time.Time
}

// Query returns records matching the given options, newest first.
func (d *DB) Query(opts QueryOptions) ([]Record, error) {
	query := `SELECT id, timestamp, command, directory, exit_code, session_id, hostname FROM commands WHERE 1=1`
	args := []interface{}{}

	if opts.Search != "" {
		query += ` AND command LIKE ?`
		args = append(args, "%"+opts.Search+"%")
	}
	if opts.Directory != "" {
		query += ` AND directory = ?`
		args = append(args, opts.Directory)
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
		if err := rows.Scan(&r.ID, &ts, &r.Command, &r.Directory, &exitCode, &r.SessionID, &r.Hostname); err != nil {
			return nil, err
		}
		r.Timestamp, _ = time.Parse(time.RFC3339Nano, ts)
		if exitCode.Valid {
			v := int(exitCode.Int64)
			r.ExitCode = &v
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
