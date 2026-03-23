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
	"time"

	"envious-web/internal/config"
	"envious-web/internal/env"

	_ "modernc.org/sqlite"
)

type Storage struct {
	db            *sql.DB
	encryptionKey []byte
}

var (
	ErrDuplicateKey = errors.New("duplicate key")
	ErrNotFound     = errors.New("not found")
)

func Open(cfg *config.Config) (*Storage, error) {
	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, err
	}

	s := &Storage{db: db, encryptionKey: cfg.EncryptionKey}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Storage) Close() error { return s.db.Close() }

func (s *Storage) migrate() error {
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
	return nil
}

func (s *Storage) ensureDefaultApp() error {
	_, err := s.db.Exec(`
		INSERT INTO applications (id, name) VALUES (1, 'default')
		ON CONFLICT(id) DO NOTHING
	`)
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
	err := s.db.QueryRowContext(ctx, "SELECT hash FROM api_key WHERE id = 1").Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return hash, err
}

func (s *Storage) SetAPIKeyHash(ctx context.Context, hash string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO api_key (id, hash) VALUES (1, ?) 
		ON CONFLICT(id) DO UPDATE SET hash=excluded.hash
	`, hash)
	return err
}

// Applications
func (s *Storage) CreateApp(ctx context.Context, name string) (int64, error) {
	res, err := s.db.ExecContext(ctx, "INSERT INTO applications (name) VALUES (?)", name)
	if err != nil {
		if isUniqueConstraint(err) {
			return 0, ErrDuplicateKey
		}
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Storage) ListApps(ctx context.Context) ([]env.Application, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name, created_at FROM applications ORDER BY id ASC")
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
	err := s.db.QueryRowContext(ctx, "SELECT id, name, created_at FROM applications WHERE id = ?", id).
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
	_, err := s.db.ExecContext(ctx, "DELETE FROM applications WHERE id = ?", id)
	return err
}

// Environments
func (s *Storage) CreateEnv(ctx context.Context, appID int64, name string) (int64, error) {
	if appID == 0 {
		appID = 1
	}
	res, err := s.db.ExecContext(ctx, "INSERT INTO environments (app_id, name) VALUES (?, ?)", appID, name)
	if err != nil {
		if isUniqueConstraint(err) {
			return 0, ErrDuplicateKey
		}
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Storage) ListEnvs(ctx context.Context, appID int64) ([]env.Environment, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if appID == 0 {
		rows, err = s.db.QueryContext(ctx, "SELECT id, app_id, name, created_at FROM environments ORDER BY app_id ASC, id ASC")
	} else {
		rows, err = s.db.QueryContext(ctx, "SELECT id, app_id, name, created_at FROM environments WHERE app_id = ? ORDER BY id ASC", appID)
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
	err := s.db.QueryRowContext(ctx, "SELECT id, app_id, name, created_at FROM environments WHERE id = ?", id).
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
	_, err := s.db.ExecContext(ctx, "DELETE FROM environments WHERE id = ?", id)
	return err
}

// Variables
func (s *Storage) ListVars(ctx context.Context, envID int64) ([]env.Variable, error) {
	rows, err := s.db.QueryContext(ctx, `
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

func (s *Storage) GetVar(ctx context.Context, envID int64, key string) (*env.Variable, error) {
	var v env.Variable
	var enc, created, updated string
	err := s.db.QueryRowContext(ctx, `
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
	err = tx.QueryRow("SELECT id, version FROM variables WHERE env_id = ? AND key = ?", envID, key).Scan(&id, &version)
	if errors.Is(err, sql.ErrNoRows) {
		res, err := tx.Exec(`
			INSERT INTO variables (env_id, key, value_encrypted, version, created_at, updated_at)
			VALUES (?, ?, ?, 1, ?, ?)
		`, envID, key, enc, now, now)
		if err != nil {
			if isUniqueConstraint(err) {
				return nil, ErrDuplicateKey
			}
			return nil, err
		}
		newID, _ := res.LastInsertId()
		if _, err := tx.Exec(`
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
	if _, err := tx.Exec(`
		UPDATE variables SET value_encrypted = ?, version = ?, updated_at = ? WHERE id = ?
	`, enc, newVersion, now, id); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`
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

func (s *Storage) UpdateVar(ctx context.Context, varID int64, value string) (*env.Variable, error) {
	// Fetch existing
	var envID int64
	var key string
	err := s.db.QueryRowContext(ctx, "SELECT env_id, key FROM variables WHERE id = ?", varID).Scan(&envID, &key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.SetVar(ctx, envID, key, value)
}

func (s *Storage) DeleteVar(ctx context.Context, envID int64, key string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM variables WHERE env_id = ? AND key = ?", envID, key)
	return err
}

func (s *Storage) GetVarMetaByID(ctx context.Context, id int64) (int64, string, error) {
	var envID int64
	var key string
	err := s.db.QueryRowContext(ctx, "SELECT env_id, key FROM variables WHERE id = ?", id).Scan(&envID, &key)
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
	// modernc.org/sqlite uses "constraint failed" text for unique violations
	return err != nil && (contains(err.Error(), "UNIQUE constraint failed") || contains(err.Error(), "constraint failed"))
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

