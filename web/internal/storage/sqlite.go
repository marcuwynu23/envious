package storage

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"envious-web/internal/config"
	"envious-web/internal/env"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

const (
	// DialectSQLite is the embedded single-file backend (default).
	DialectSQLite = "sqlite"
	// DialectPostgres is the server-based backend for multi-instance use.
	DialectPostgres = "postgres"
)

type Storage struct {
	db            *sql.DB
	dialect       string
	encryptionKey []byte
}

var (
	ErrDuplicateKey = errors.New("duplicate key")
	ErrNotFound     = errors.New("not found")
)

func Open(cfg *config.Config) (*Storage, error) {
	switch cfg.Driver {
	case "", DialectSQLite:
		return openSQLite(cfg)
	case DialectPostgres:
		return openPostgres(cfg)
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER %q: want sqlite or postgres", cfg.Driver)
	}
}

func poolOf(cfg *config.Config, defOpen, defIdle int) (int, int) {
	maxOpen, maxIdle := defOpen, defIdle
	if cfg.DBMaxOpenConns > 0 {
		maxOpen = cfg.DBMaxOpenConns
	}
	if cfg.DBMaxIdleConns > 0 {
		maxIdle = cfg.DBMaxIdleConns
	}
	return maxOpen, maxIdle
}

func openSQLite(cfg *config.Config) (*Storage, error) {
	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return nil, err
	}
	maxOpen, maxIdle := poolOf(cfg, 10, 10)
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(5 * time.Minute)
	// WAL lets readers run concurrently with the single writer; busy_timeout
	// turns lock contention into waits instead of instant SQLITE_BUSY errors.
	for _, pragma := range []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA synchronous = NORMAL",
	} {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("%s: %w", pragma, err)
		}
	}

	s := &Storage{db: db, dialect: DialectSQLite, encryptionKey: cfg.EncryptionKey}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func openPostgres(cfg *config.Config) (*Storage, error) {
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return nil, fmt.Errorf("DATABASE_URL is required when DB_DRIVER=postgres")
	}
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	maxOpen, maxIdle := poolOf(cfg, 25, 5)
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(5 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}

	s := &Storage{db: db, dialect: DialectPostgres, encryptionKey: cfg.EncryptionKey}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	// Repair the applications sequence on every open: rows inserted with an
	// explicit id (e.g. the id=1 default app, or databases migrated before
	// the sequence fix) don't advance BIGSERIAL, which would surface as
	// phantom duplicate-key errors on the next CreateApp.
	if _, err := s.db.Exec(`SELECT setval(pg_get_serial_sequence('applications', 'id'), COALESCE((SELECT MAX(id) FROM applications), 1))`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sequence repair: %w", err)
	}
	return s, nil
}

// Dialect reports the active backend ("sqlite" or "postgres").
func (s *Storage) Dialect() string { return s.dialect }

// rebind converts ? placeholders to $1..$n for Postgres; SQLite passes through.
func (s *Storage) rebind(query string) string {
	if s.dialect != DialectPostgres {
		return query
	}
	var b strings.Builder
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			b.WriteString("$" + itoa(n))
		} else {
			b.WriteByte(query[i])
		}
	}
	return b.String()
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }

func (s *Storage) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.db.ExecContext(ctx, s.rebind(query), args...)
}

func (s *Storage) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, s.rebind(query), args...)
}

func (s *Storage) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return s.db.QueryRowContext(ctx, s.rebind(query), args...)
}

// insertID runs an INSERT and returns the new row id (RETURNING on Postgres,
// LastInsertId on SQLite).
func (s *Storage) insertID(ctx context.Context, query string, args ...any) (int64, error) {
	if s.dialect == DialectPostgres {
		var id int64
		if err := s.queryRow(ctx, query+" RETURNING id", args...).Scan(&id); err != nil {
			return 0, err
		}
		return id, nil
	}
	res, err := s.exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Storage) insertIDTx(ctx context.Context, tx *sql.Tx, query string, args ...any) (int64, error) {
	if s.dialect == DialectPostgres {
		var id int64
		rebound, params := rebindWithArgs(query+" RETURNING id", args)
		if err := tx.QueryRowContext(ctx, rebound, params...).Scan(&id); err != nil {
			return 0, err
		}
		return id, nil
	}
	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// rebindWithArgs is the static variant used inside transactions.
func rebindWithArgs(query string, args []any) (string, []any) {
	var b strings.Builder
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			b.WriteString("$" + itoa(n))
		} else {
			b.WriteByte(query[i])
		}
	}
	return b.String(), args
}

// nowUTC stamps created/updated columns in a dialect-neutral RFC3339 format.
func nowUTC() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func (s *Storage) Close() error { return s.db.Close() }

// Ping reports database reachability (readiness probes).
func (s *Storage) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

func (s *Storage) migrate() error {
	if s.dialect == DialectPostgres {
		return s.migratePostgres()
	}
	return s.migrateSQLite()
}

// migratePostgres creates fresh tables. Timestamps are TEXT filled by the
// app (RFC3339) so reads behave identically across backends.
func (s *Storage) migratePostgres() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS applications (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS api_key (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			hash TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS environments (
			id BIGSERIAL PRIMARY KEY,
			app_id BIGINT NOT NULL,
			name TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(app_id, name),
			FOREIGN KEY(app_id) REFERENCES applications(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS variables (
			id BIGSERIAL PRIMARY KEY,
			env_id BIGINT NOT NULL,
			key TEXT NOT NULL,
			value_encrypted TEXT NOT NULL,
			version BIGINT NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE (env_id, key),
			FOREIGN KEY(env_id) REFERENCES environments(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS variable_versions (
			id BIGSERIAL PRIMARY KEY,
			var_id BIGINT NOT NULL,
			env_id BIGINT NOT NULL,
			key TEXT NOT NULL,
			value_encrypted TEXT NOT NULL,
			version BIGINT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY(var_id) REFERENCES variables(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS activity_logs (
			id BIGSERIAL PRIMARY KEY,
			created_at TEXT NOT NULL,
			actor TEXT NOT NULL,
			action TEXT NOT NULL,
			resource_type TEXT NOT NULL DEFAULT '',
			resource_id BIGINT NOT NULL DEFAULT 0,
			detail TEXT NOT NULL DEFAULT '',
			ip TEXT NOT NULL DEFAULT '',
			request_id TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_activity_logs_action ON activity_logs(action)`,
		`CREATE INDEX IF NOT EXISTS idx_activity_logs_created ON activity_logs(created_at)`,
	}
	for _, stmt := range tables {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	if err := s.ensureDefaultApp(); err != nil {
		return err
	}
	return nil
}

func (s *Storage) migrateSQLite() error {
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS applications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS api_key (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			hash TEXT NOT NULL
		);
	`); err != nil {
		return err
	}
	if err := s.ensureDefaultApp(); err != nil {
		return err
	}
	if err := s.migrateEnvironmentsToApps(); err != nil {
		return err
	}
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS variables (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			env_id INTEGER NOT NULL,
			key TEXT NOT NULL,
			value_encrypted TEXT NOT NULL,
			version INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (env_id, key),
			FOREIGN KEY(env_id) REFERENCES environments(id) ON DELETE CASCADE
		);
	`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS variable_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			var_id INTEGER NOT NULL,
			env_id INTEGER NOT NULL,
			key TEXT NOT NULL,
			value_encrypted TEXT NOT NULL,
			version INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(var_id) REFERENCES variables(id) ON DELETE CASCADE
		);
	`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS activity_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			actor TEXT NOT NULL,
			action TEXT NOT NULL,
			resource_type TEXT NOT NULL DEFAULT '',
			resource_id INTEGER NOT NULL DEFAULT 0,
			detail TEXT NOT NULL DEFAULT '',
			ip TEXT NOT NULL DEFAULT '',
			request_id TEXT NOT NULL DEFAULT ''
		);
	`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_activity_logs_action ON activity_logs(action)`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_activity_logs_created ON activity_logs(created_at)`); err != nil {
		return err
	}
	return nil
}

func (s *Storage) ensureDefaultApp() error {
	_, err := s.db.Exec(s.rebind(`
		INSERT INTO applications (id, name, created_at) VALUES (1, 'default', ?)
		ON CONFLICT(id) DO NOTHING
	`), nowUTC())
	return err
}

func (s *Storage) migrateEnvironmentsToApps() error {
	exists, err := s.tableExists("environments")
	if err != nil {
		return err
	}
	if !exists {
		_, err := s.db.Exec(`
			CREATE TABLE environments (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				app_id INTEGER NOT NULL,
				name TEXT NOT NULL,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(app_id, name),
				FOREIGN KEY(app_id) REFERENCES applications(id) ON DELETE CASCADE
			);
		`)
		return err
	}

	hasAppID, err := s.tableHasColumn("environments", "app_id")
	if err != nil {
		return err
	}

	if sqlText, err := s.tableSQL("environments"); err == nil && sqlText != "" {
		if contains(sqlText, "UNIQUE(app_id, name)") && contains(sqlText, "FOREIGN KEY(app_id)") {
			return nil
		}
	}

	if _, err := s.db.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`
		CREATE TABLE environments_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			app_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			UNIQUE(app_id, name),
			FOREIGN KEY(app_id) REFERENCES applications(id) ON DELETE CASCADE
		);
	`); err != nil {
		return err
	}

	if hasAppID {
		if _, err := tx.Exec(`
			INSERT INTO environments_new (id, app_id, name, created_at)
			SELECT id, app_id, name, created_at FROM environments
		`); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(`
			INSERT INTO environments_new (id, app_id, name, created_at)
			SELECT id, 1, name, created_at FROM environments
		`); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`DROP TABLE environments`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE environments_new RENAME TO environments`); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	_, err = s.db.Exec("PRAGMA foreign_keys = ON")
	return err
}

func (s *Storage) tableSQL(name string) (string, error) {
	var sqlText sql.NullString
	err := s.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&sqlText)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !sqlText.Valid {
		return "", nil
	}
	return sqlText.String, nil
}

func (s *Storage) tableExists(name string) (bool, error) {
	var n string
	err := s.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *Storage) tableHasColumn(table, col string) (bool, error) {
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == col {
			return true, nil
		}
	}
	return false, rows.Err()
}

// API key storage
func (s *Storage) GetAPIKeyHash(ctx context.Context) (string, error) {
	var hash string
	err := s.queryRow(ctx, "SELECT hash FROM api_key WHERE id = 1").Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return hash, err
}

func (s *Storage) SetAPIKeyHash(ctx context.Context, hash string) error {
	_, err := s.exec(ctx, `
		INSERT INTO api_key (id, hash) VALUES (1, ?) 
		ON CONFLICT(id) DO UPDATE SET hash=excluded.hash
	`, hash)
	return err
}

// Applications
func (s *Storage) CreateApp(ctx context.Context, name string) (int64, error) {
	id, err := s.insertID(ctx, "INSERT INTO applications (name, created_at) VALUES (?, ?)", name, nowUTC())
	if err != nil {
		if isUniqueConstraint(err) {
			return 0, ErrDuplicateKey
		}
		return 0, err
	}
	return id, nil
}

func (s *Storage) ListApps(ctx context.Context) ([]env.Application, error) {
	rows, err := s.query(ctx, "SELECT id, name, created_at FROM applications ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []env.Application
	for rows.Next() {
		var a env.Application
		var created string
		if err := rows.Scan(&a.ID, &a.Name, &created); err != nil {
			return nil, err
		}
		t, _ := time.Parse(time.RFC3339Nano, created)
		a.CreatedAt = t
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Storage) GetApp(ctx context.Context, id int64) (*env.Application, error) {
	var a env.Application
	var created string
	err := s.queryRow(ctx, "SELECT id, name, created_at FROM applications WHERE id = ?", id).
		Scan(&a.ID, &a.Name, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	t, _ := time.Parse(time.RFC3339Nano, created)
	a.CreatedAt = t
	return &a, nil
}

func (s *Storage) DeleteApp(ctx context.Context, id int64) error {
	if id == 1 {
		return fmt.Errorf("cannot delete default application")
	}
	res, err := s.exec(ctx, "DELETE FROM applications WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Environments
func (s *Storage) CreateEnv(ctx context.Context, appID int64, name string) (int64, error) {
	if appID == 0 {
		appID = 1
	}
	id, err := s.insertID(ctx, "INSERT INTO environments (app_id, name, created_at) VALUES (?, ?, ?)", appID, name, nowUTC())
	if err != nil {
		if isUniqueConstraint(err) {
			return 0, ErrDuplicateKey
		}
		return 0, err
	}
	return id, nil
}

func (s *Storage) ListEnvs(ctx context.Context, appID int64) ([]env.Environment, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if appID == 0 {
		rows, err = s.query(ctx, "SELECT id, app_id, name, created_at FROM environments ORDER BY app_id ASC, id ASC")
	} else {
		rows, err = s.query(ctx, "SELECT id, app_id, name, created_at FROM environments WHERE app_id = ? ORDER BY id ASC", appID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []env.Environment
	for rows.Next() {
		var e env.Environment
		var created string
		if err := rows.Scan(&e.ID, &e.AppID, &e.Name, &created); err != nil {
			return nil, err
		}
		t, _ := time.Parse(time.RFC3339Nano, created)
		e.CreatedAt = t
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Storage) GetEnv(ctx context.Context, id int64) (*env.Environment, error) {
	var e env.Environment
	var created string
	err := s.queryRow(ctx, "SELECT id, app_id, name, created_at FROM environments WHERE id = ?", id).
		Scan(&e.ID, &e.AppID, &e.Name, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	t, _ := time.Parse(time.RFC3339Nano, created)
	e.CreatedAt = t
	return &e, nil
}

func (s *Storage) DeleteEnv(ctx context.Context, id int64) error {
	res, err := s.exec(ctx, "DELETE FROM environments WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Variables
func (s *Storage) ListVars(ctx context.Context, envID int64) ([]env.Variable, error) {
	rows, err := s.query(ctx, `
		SELECT id, env_id, key, value_encrypted, version, created_at, updated_at
		FROM variables WHERE env_id = ? ORDER BY key ASC
	`, envID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []env.Variable
	for rows.Next() {
		var v env.Variable
		var enc string
		var created, updated string
		if err := rows.Scan(&v.ID, &v.EnvID, &v.Key, &enc, &v.Version, &created, &updated); err != nil {
			return nil, err
		}
		val, err := s.decrypt(enc)
		if err != nil {
			return nil, err
		}
		v.Value = val
		v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		v.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Storage) CountVars(ctx context.Context, envID int64) (int64, error) {
	var n int64
	err := s.queryRow(ctx, `SELECT COUNT(*) FROM variables WHERE env_id = ?`, envID).Scan(&n)
	return n, err
}

func (s *Storage) ListVarsPage(ctx context.Context, envID int64, limit, offset int) ([]env.Variable, error) {
	if limit <= 0 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.query(ctx, `
		SELECT id, env_id, key, value_encrypted, version, created_at, updated_at
		FROM variables
		WHERE env_id = ?
		ORDER BY key ASC
		LIMIT ? OFFSET ?
	`, envID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []env.Variable
	for rows.Next() {
		var v env.Variable
		var enc string
		var created, updated string
		if err := rows.Scan(&v.ID, &v.EnvID, &v.Key, &enc, &v.Version, &created, &updated); err != nil {
			return nil, err
		}
		val, err := s.decrypt(enc)
		if err != nil {
			return nil, err
		}
		v.Value = val
		v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		v.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Storage) GetVar(ctx context.Context, envID int64, key string) (*env.Variable, error) {
	var v env.Variable
	var enc, created, updated string
	err := s.queryRow(ctx, `
		SELECT id, env_id, key, value_encrypted, version, created_at, updated_at
		FROM variables WHERE env_id = ? AND key = ?
	`, envID, key).Scan(&v.ID, &v.EnvID, &v.Key, &enc, &v.Version, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	val, err := s.decrypt(enc)
	if err != nil {
		return nil, err
	}
	v.Value = val
	v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	v.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return &v, nil
}

func (s *Storage) SetVar(ctx context.Context, envID int64, key, value string) (*env.Variable, error) {
	// Concurrent writers on the same key collide (SQLITE_BUSY on SQLite,
	// 40001 serialization failures on Postgres, or a lost insert race).
	// Retry with backoff so bursts serialize instead of 500ing.
	var err error
	for attempt := 0; attempt < 10; attempt++ {
		var v *env.Variable
		v, err = s.setVarOnce(ctx, envID, key, value)
		if err == nil {
			return v, nil
		}
		if err != ErrDuplicateKey && !isRetriable(err) {
			return nil, err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		time.Sleep(retryBackoff(attempt))
	}
	return nil, err
}

// retryBackoff grows quadratically with jitter to spread colliding writers.
func retryBackoff(attempt int) time.Duration {
	d := time.Duration(attempt*attempt)*5*time.Millisecond + 5*time.Millisecond
	if d > 200*time.Millisecond {
		d = 200 * time.Millisecond
	}
	return d
}

// isRetriable reports transient write-conflict errors worth retrying.
func isRetriable(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "40001" || pgErr.Code == "40P01"
	}
	msg := err.Error()
	return strings.Contains(msg, "SQLITE_BUSY") ||
		strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "database table is locked")
}

func (s *Storage) setVarOnce(ctx context.Context, envID int64, key, value string) (*env.Variable, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	enc, err := s.encrypt(value)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	// Try update existing
	var id, version int64
	err = s.txQueryRow(tx, "SELECT id, version FROM variables WHERE env_id = ? AND key = ?", envID, key).Scan(&id, &version)
	if errors.Is(err, sql.ErrNoRows) {
		newID, err := s.insertIDTx(ctx, tx, `
			INSERT INTO variables (env_id, key, value_encrypted, version, created_at, updated_at)
			VALUES (?, ?, ?, 1, ?, ?)
		`, envID, key, enc, now, now)
		if err != nil {
			if isUniqueConstraint(err) {
				return nil, ErrDuplicateKey
			}
			return nil, err
		}
		if _, err := s.txExec(ctx, tx, `
			INSERT INTO variable_versions (var_id, env_id, key, value_encrypted, version, created_at)
			VALUES (?, ?, ?, ?, 1, ?)
		`, newID, envID, key, enc, now); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return &env.Variable{ID: newID, EnvID: envID, Key: key, Value: value, Version: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}, nil
	}
	if err != nil {
		return nil, err
	}
	newVersion := version + 1
	if _, err := s.txExec(ctx, tx, `
		UPDATE variables SET value_encrypted = ?, version = ?, updated_at = ? WHERE id = ?
	`, enc, newVersion, now, id); err != nil {
		return nil, err
	}
	if _, err := s.txExec(ctx, tx, `
		INSERT INTO variable_versions (var_id, env_id, key, value_encrypted, version, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, envID, key, enc, newVersion, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &env.Variable{ID: id, EnvID: envID, Key: key, Value: value, Version: newVersion}, nil
}

// txQueryRow / txExec mirror queryRow/exec inside a transaction.
func (s *Storage) txQueryRow(tx *sql.Tx, query string, args ...any) *sql.Row {
	return tx.QueryRow(s.rebind(query), args...)
}

func (s *Storage) txExec(ctx context.Context, tx *sql.Tx, query string, args ...any) (sql.Result, error) {
	return tx.ExecContext(ctx, s.rebind(query), args...)
}

func (s *Storage) UpdateVar(ctx context.Context, varID int64, value string) (*env.Variable, error) {
	// Fetch existing
	var envID int64
	var key string
	err := s.queryRow(ctx, "SELECT env_id, key FROM variables WHERE id = ?", varID).Scan(&envID, &key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.SetVar(ctx, envID, key, value)
}

func (s *Storage) DeleteVar(ctx context.Context, envID int64, key string) error {
	_, err := s.exec(ctx, "DELETE FROM variables WHERE env_id = ? AND key = ?", envID, key)
	return err
}

func (s *Storage) GetVarMetaByID(ctx context.Context, id int64) (int64, string, error) {
	var envID int64
	var key string
	err := s.queryRow(ctx, "SELECT env_id, key FROM variables WHERE id = ?", id).Scan(&envID, &key)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", ErrNotFound
	}
	return envID, key, err
}

// Helpers
func (s *Storage) encrypt(plain string) (string, error) {
	if len(s.encryptionKey) == 0 {
		return base64.StdEncoding.EncodeToString([]byte(plain)), nil
	}
	block, err := aes.NewCipher(normalizeKey(s.encryptionKey))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *Storage) decrypt(enc string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	if len(s.encryptionKey) == 0 {
		return string(data), nil
	}
	block, err := aes.NewCipher(normalizeKey(s.encryptionKey))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", fmt.Errorf("malformed ciphertext")
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func normalizeKey(k []byte) []byte {
	// AES-256 requires 32 bytes; pad or trim as needed.
	key := make([]byte, 32)
	copy(key, k)
	return key
}

func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	// Postgres unique violation.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	// modernc.org/sqlite uses "constraint failed" text for unique violations
	return contains(err.Error(), "UNIQUE constraint failed")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool { return (len(s) > 0 && len(sub) > 0 && (stringContains(s, sub))) })()
}

func stringContains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && (indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Activity is one audit-trail entry. Detail carries metadata only —
// never secret values.
type Activity struct {
	ID           int64     `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	Actor        string    `json:"actor"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   int64     `json:"resource_id"`
	Detail       string    `json:"detail"`
	IP           string    `json:"ip"`
	RequestID    string    `json:"request_id"`
}

// LogActivity appends an audit-trail entry.
func (s *Storage) LogActivity(ctx context.Context, actor, action, resourceType string, resourceID int64, detail, ip, requestID string) error {
	_, err := s.exec(ctx,
		`INSERT INTO activity_logs (created_at, actor, action, resource_type, resource_id, detail, ip, request_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		nowUTC(), actor, action, resourceType, resourceID, detail, ip, requestID)
	return err
}

// ListActivity returns recent audit entries, newest first. An empty action
// matches all; limit <= 0 defaults to 100 and caps at 1000.
func (s *Storage) ListActivity(ctx context.Context, action string, limit int) ([]Activity, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	var (
		rows *sql.Rows
		err  error
	)
	if action == "" {
		rows, err = s.query(ctx,
			`SELECT id, created_at, actor, action, resource_type, resource_id, detail, ip, request_id
			 FROM activity_logs ORDER BY id DESC LIMIT ?`, limit)
	} else {
		rows, err = s.query(ctx,
			`SELECT id, created_at, actor, action, resource_type, resource_id, detail, ip, request_id
			 FROM activity_logs WHERE action = ? ORDER BY id DESC LIMIT ?`, action, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Activity
	for rows.Next() {
		var a Activity
		var created string
		if err := rows.Scan(&a.ID, &created, &a.Actor, &a.Action, &a.ResourceType, &a.ResourceID, &a.Detail, &a.IP, &a.RequestID); err != nil {
			return nil, err
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		if a.CreatedAt.IsZero() {
			a.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

