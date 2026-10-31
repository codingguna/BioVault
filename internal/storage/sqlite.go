package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"fp-manager/internal/security"
)

// User represents an enrolled local user profile.
type User struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	FingerCount int       `json:"finger_count"`
}

// Template represents a stored fingerprint template.
type Template struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	FingerLabel    string    `json:"finger_label"`
	TemplateFormat string    `json:"template_format"`
	SDKVersion     int       `json:"sdk_version"`
	Quality        int       `json:"quality"`
	NFIQ           int       `json:"nfiq"`
	CreatedAt      time.Time `json:"created_at"`
	DecryptedBytes []byte    `json:"-"` // Never exposed in JSON serialization
}

// AuditEvent represents a logged system or authentication event.
type AuditEvent struct {
	ID             string    `json:"id"`
	EventTime      time.Time `json:"event_time"`
	EventType      string    `json:"event_type"`
	UserID         string    `json:"user_id,omitempty"`
	UserName       string    `json:"user_name,omitempty"`
	Result         string    `json:"result"`
	DiagnosticCode int       `json:"diagnostic_code"`
	Details        string    `json:"details"`
}

// WindowsCredential stores the DPAPI-encrypted Windows account binding.
type WindowsCredential struct {
	UserID        string    `json:"user_id"`
	WindowsUser   string    `json:"windows_user"`
	WindowsDomain string    `json:"windows_domain"`
	PlainPassword string    `json:"-"` // Never serialized to JSON
	UpdatedAt     time.Time `json:"updated_at"`
}

type DB struct {
	db *sql.DB
}

// Open initializes SQLite database at dbPath and applies migrations.
func Open(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	// Configure connection pool for SQLite
	db.SetMaxOpenConns(1)

	s := &DB{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return s, nil
}

func (s *DB) Close() error {
	return s.db.Close()
}

func (s *DB) migrate() error {
	schema := `
	PRAGMA foreign_keys = ON;

	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		display_name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS fingerprint_templates (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		finger_label TEXT NOT NULL,
		template_format TEXT NOT NULL DEFAULT 'ISO_19794_2',
		sdk_version INTEGER NOT NULL DEFAULT 9025,
		template_blob_encrypted BLOB NOT NULL,
		quality INTEGER NOT NULL DEFAULT 0,
		nfiq INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL,
		UNIQUE(user_id, finger_label)
	);

	CREATE TABLE IF NOT EXISTS audit_events (
		id TEXT PRIMARY KEY,
		event_time DATETIME NOT NULL,
		event_type TEXT NOT NULL,
		user_id TEXT,
		user_name TEXT,
		result TEXT NOT NULL,
		diagnostic_code INTEGER NOT NULL DEFAULT 0,
		details TEXT
	);

	CREATE TABLE IF NOT EXISTS windows_credentials (
		user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		windows_user TEXT NOT NULL,
		windows_domain TEXT NOT NULL,
		encrypted_password BLOB NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_templates_user ON fingerprint_templates(user_id);
	CREATE INDEX IF NOT EXISTS idx_audit_time ON audit_events(event_time DESC);
	`
	_, err := s.db.Exec(schema)
	return err
}

// CreateUser registers a new named profile.
func (s *DB) CreateUser(ctx context.Context, displayName string) (*User, error) {
	if displayName == "" {
		return nil, errors.New("display name cannot be empty")
	}

	now := time.Now().UTC()
	u := &User{
		ID:          uuid.New().String(),
		DisplayName: displayName,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	query := `INSERT INTO users (id, display_name, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, u.ID, u.DisplayName, u.Status, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	_ = s.LogAudit(ctx, "USER_CREATED", u.ID, u.DisplayName, "SUCCESS", 0, "Created user profile")
	return u, nil
}

// ListUsers returns all users along with their enrolled template counts.
func (s *DB) ListUsers(ctx context.Context) ([]User, error) {
	query := `
		SELECT u.id, u.display_name, u.status, u.created_at, u.updated_at,
		       (SELECT COUNT(*) FROM fingerprint_templates t WHERE t.user_id = u.id) as finger_count
		FROM users u
		ORDER BY u.created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.DisplayName, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.FingerCount); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// GetUser returns a user by ID.
func (s *DB) GetUser(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT u.id, u.display_name, u.status, u.created_at, u.updated_at,
		       (SELECT COUNT(*) FROM fingerprint_templates t WHERE t.user_id = u.id) as finger_count
		FROM users u
		WHERE u.id = ?
	`
	var u User
	err := s.db.QueryRowContext(ctx, query, id).Scan(&u.ID, &u.DisplayName, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.FingerCount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &u, nil
}

// DeleteUser removes a user and cascades deletion to their templates.
func (s *DB) DeleteUser(ctx context.Context, id string) error {
	u, err := s.GetUser(ctx, id)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return err
	}

	_ = s.LogAudit(ctx, "USER_DELETED", id, u.DisplayName, "SUCCESS", 0, "User profile and templates deleted")
	return nil
}

// SaveTemplate encrypts the template blob with DPAPI and stores it.
func (s *DB) SaveTemplate(ctx context.Context, userID, fingerLabel string, format string, sdkVer int, rawTemplate []byte, quality, nfiq int) (*Template, error) {
	if len(rawTemplate) == 0 {
		return nil, errors.New("template data cannot be empty")
	}

	// Encrypt at rest via Windows DPAPI
	encBlob, err := security.EncryptTemplate(rawTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt template: %w", err)
	}

	t := &Template{
		ID:             uuid.New().String(),
		UserID:         userID,
		FingerLabel:    fingerLabel,
		TemplateFormat: format,
		SDKVersion:     sdkVer,
		Quality:        quality,
		NFIQ:           nfiq,
		CreatedAt:      time.Now().UTC(),
		DecryptedBytes: rawTemplate,
	}

	// Upsert template for this user and finger label
	query := `
		INSERT INTO fingerprint_templates (id, user_id, finger_label, template_format, sdk_version, template_blob_encrypted, quality, nfiq, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, finger_label) DO UPDATE SET
			template_blob_encrypted = excluded.template_blob_encrypted,
			template_format = excluded.template_format,
			sdk_version = excluded.sdk_version,
			quality = excluded.quality,
			nfiq = excluded.nfiq,
			created_at = excluded.created_at
	`
	_, err = s.db.ExecContext(ctx, query, t.ID, t.UserID, t.FingerLabel, t.TemplateFormat, t.SDKVersion, encBlob, t.Quality, t.NFIQ, t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to store template: %w", err)
	}

	u, _ := s.GetUser(ctx, userID)
	userName := ""
	if u != nil {
		userName = u.DisplayName
	}

	_ = s.LogAudit(ctx, "TEMPLATE_ENROLLED", userID, userName, "SUCCESS", 0, fmt.Sprintf("Enrolled finger: %s (Quality: %d)", fingerLabel, quality))
	return t, nil
}

// GetUserTemplates returns all enrolled templates for a specific user with decrypted bytes.
func (s *DB) GetUserTemplates(ctx context.Context, userID string) ([]Template, error) {
	query := `
		SELECT id, user_id, finger_label, template_format, sdk_version, template_blob_encrypted, quality, nfiq, created_at
		FROM fingerprint_templates
		WHERE user_id = ?
		ORDER BY finger_label ASC
	`
	return s.queryAndDecryptTemplates(ctx, query, userID)
}

// GetAllTemplates returns all enrolled templates in the entire database (for 1:N identification).
func (s *DB) GetAllTemplates(ctx context.Context) ([]Template, error) {
	query := `
		SELECT id, user_id, finger_label, template_format, sdk_version, template_blob_encrypted, quality, nfiq, created_at
		FROM fingerprint_templates
	`
	return s.queryAndDecryptTemplates(ctx, query)
}

func (s *DB) queryAndDecryptTemplates(ctx context.Context, query string, args ...any) ([]Template, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []Template
	for rows.Next() {
		var t Template
		var encBlob []byte
		if err := rows.Scan(&t.ID, &t.UserID, &t.FingerLabel, &t.TemplateFormat, &t.SDKVersion, &encBlob, &t.Quality, &t.NFIQ, &t.CreatedAt); err != nil {
			return nil, err
		}

		plain, err := security.DecryptTemplate(encBlob)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt template %s: %w", t.ID, err)
		}
		t.DecryptedBytes = plain
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

// DeleteTemplate removes a specific enrolled finger template.
func (s *DB) DeleteTemplate(ctx context.Context, templateID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM fingerprint_templates WHERE id = ?`, templateID)
	return err
}

// LogAudit persists an audit trail event.
func (s *DB) LogAudit(ctx context.Context, eventType, userID, userName, result string, diagCode int, details string) error {
	evt := &AuditEvent{
		ID:             uuid.New().String(),
		EventTime:      time.Now().UTC(),
		EventType:      eventType,
		UserID:         userID,
		UserName:       userName,
		Result:         result,
		DiagnosticCode: diagCode,
		Details:        details,
	}

	query := `INSERT INTO audit_events (id, event_time, event_type, user_id, user_name, result, diagnostic_code, details) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, evt.ID, evt.EventTime, evt.EventType, evt.UserID, evt.UserName, evt.Result, evt.DiagnosticCode, evt.Details)
	return err
}

// ListAuditEvents fetches the latest audit events.
func (s *DB) ListAuditEvents(ctx context.Context, limit int) ([]AuditEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, event_time, event_type, COALESCE(user_id, ''), COALESCE(user_name, ''), result, diagnostic_code, COALESCE(details, '')
		FROM audit_events
		ORDER BY event_time DESC
		LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []AuditEvent
	for rows.Next() {
		var e AuditEvent
		if err := rows.Scan(&e.ID, &e.EventTime, &e.EventType, &e.UserID, &e.UserName, &e.Result, &e.DiagnosticCode, &e.Details); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// SaveWindowsCredential binds a user's fingerprint profile to a Windows account with DPAPI-encrypted password.
func (s *DB) SaveWindowsCredential(ctx context.Context, userID, winUser, winDomain, password string) error {
	if userID == "" || winUser == "" || password == "" {
		return errors.New("userID, winUser, and password cannot be empty")
	}

	encPass, err := security.EncryptTemplate([]byte(password))
	if err != nil {
		return fmt.Errorf("failed to encrypt Windows password: %w", err)
	}

	now := time.Now().UTC()
	query := `
		INSERT INTO windows_credentials (user_id, windows_user, windows_domain, encrypted_password, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			windows_user = excluded.windows_user,
			windows_domain = excluded.windows_domain,
			encrypted_password = excluded.encrypted_password,
			updated_at = excluded.updated_at
	`
	_, err = s.db.ExecContext(ctx, query, userID, winUser, winDomain, encPass, now)
	if err != nil {
		return fmt.Errorf("failed to save windows credential: %w", err)
	}

	u, _ := s.GetUser(ctx, userID)
	userName := ""
	if u != nil {
		userName = u.DisplayName
	}
	_ = s.LogAudit(ctx, "WINDOWS_ACCOUNT_BOUND", userID, userName, "SUCCESS", 0,
		fmt.Sprintf("Bound to Windows account: %s\\%s", winDomain, winUser))
	return nil
}

// GetWindowsCredential retrieves and decrypts the bound Windows credentials for a user.
func (s *DB) GetWindowsCredential(ctx context.Context, userID string) (*WindowsCredential, error) {
	query := `SELECT user_id, windows_user, windows_domain, encrypted_password, updated_at FROM windows_credentials WHERE user_id = ?`
	var c WindowsCredential
	var encPass []byte

	err := s.db.QueryRowContext(ctx, query, userID).Scan(&c.UserID, &c.WindowsUser, &c.WindowsDomain, &encPass, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("no Windows credential bound for this user")
		}
		return nil, err
	}

	plainPass, err := security.DecryptTemplate(encPass)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt Windows password: %w", err)
	}
	c.PlainPassword = string(plainPass)
	return &c, nil
}

// HasWindowsCredential checks if a user has a bound Windows account.
func (s *DB) HasWindowsCredential(ctx context.Context, userID string) bool {
	var count int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM windows_credentials WHERE user_id = ?`, userID).Scan(&count)
	return count > 0
}
