package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kod2ulz/gostart/ierrors"
	"github.com/kod2ulz/gostart/errors"
)

// SessionStore interface for managing user sessions
type SessionStore interface {
	Create(ctx context.Context, session *Session) error
	Get(ctx context.Context, sessionID string) (*Session, error)
	Update(ctx context.Context, session *Session) error
	Delete(ctx context.Context, sessionID string) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*Session, error)
	Cleanup(ctx context.Context) error
}

// Session represents a user session
type Session struct {
	ID        string                 `json:"id"`
	UserID    uuid.UUID              `json:"user_id"`
	Token     string                 `json:"token,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	ExpiresAt time.Time              `json:"expires_at"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
}

// IsExpired checks if the session is expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// MemorySessionStore implements in-memory session storage
type MemorySessionStore struct {
	sessions map[string]*Session
}

// NewMemorySessionStore creates a new in-memory session store
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions: make(map[string]*Session),
	}
}

// Create creates a new session
func (m *MemorySessionStore) Create(ctx context.Context, session *Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}
	session.UpdatedAt = time.Now()
	m.sessions[session.ID] = session
	return nil
}

// Get retrieves a session by ID
func (m *MemorySessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	if session.IsExpired() {
		delete(m.sessions, sessionID)
		return nil, fmt.Errorf("session expired")
	}
	return session, nil
}

// Update updates an existing session
func (m *MemorySessionStore) Update(ctx context.Context, session *Session) error {
	if _, exists := m.sessions[session.ID]; !exists {
		return fmt.Errorf("session not found")
	}
	session.UpdatedAt = time.Now()
	m.sessions[session.ID] = session
	return nil
}

// Delete deletes a session
func (m *MemorySessionStore) Delete(ctx context.Context, sessionID string) error {
	delete(m.sessions, sessionID)
	return nil
}

// DeleteByUserID deletes all sessions for a user
func (m *MemorySessionStore) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	for id, session := range m.sessions {
		if session.UserID == userID {
			delete(m.sessions, id)
		}
	}
	return nil
}

// GetByUserID retrieves all sessions for a user
func (m *MemorySessionStore) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*Session, error) {
	var sessions []*Session
	for _, session := range m.sessions {
		if session.UserID == userID && !session.IsExpired() {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

// Cleanup removes expired sessions
func (m *MemorySessionStore) Cleanup(ctx context.Context) error {
	for id, session := range m.sessions {
		if session.IsExpired() {
			delete(m.sessions, id)
		}
	}
	return nil
}

// RedisSessionStore implements Redis-based session storage
type RedisSessionStore struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// NewRedisSessionStore creates a new Redis session store
func NewRedisSessionStore(client *redis.Client, prefix string, ttl time.Duration) *RedisSessionStore {
	if prefix == "" {
		prefix = "session:"
	}
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	return &RedisSessionStore{
		client: client,
		prefix: prefix,
		ttl:    ttl,
	}
}

// Create creates a new session
func (r *RedisSessionStore) Create(ctx context.Context, session *Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}
	session.UpdatedAt = time.Now()
	if session.ExpiresAt.IsZero() {
		session.ExpiresAt = time.Now().Add(r.ttl)
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := r.prefix + session.ID
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("session already expired")
	}

	err = r.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}

	// Index by user ID for quick lookup
	userKey := r.prefix + "user:" + session.UserID.String()
	err = r.client.SAdd(ctx, userKey, session.ID).Err()
	if err != nil {
		return fmt.Errorf("failed to index session by user: %w", err)
	}
	r.client.Expire(ctx, userKey, ttl)

	return nil
}

// Get retrieves a session by ID
func (r *RedisSessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	key := r.prefix + sessionID
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve session: %w", err)
	}

	var session Session
	err = json.Unmarshal(data, &session)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	if session.IsExpired() {
		r.Delete(ctx, sessionID)
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

// Update updates an existing session
func (r *RedisSessionStore) Update(ctx context.Context, session *Session) error {
	session.UpdatedAt = time.Now()
	return r.Create(ctx, session)
}

// Delete deletes a session
func (r *RedisSessionStore) Delete(ctx context.Context, sessionID string) error {
	key := r.prefix + sessionID
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// DeleteByUserID deletes all sessions for a user
func (r *RedisSessionStore) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	userKey := r.prefix + "user:" + userID.String()
	sessionIDs, err := r.client.SMembers(ctx, userKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get user sessions: %w", err)
	}

	for _, sessionID := range sessionIDs {
		r.Delete(ctx, sessionID)
	}

	r.client.Del(ctx, userKey)
	return nil
}

// GetByUserID retrieves all sessions for a user
func (r *RedisSessionStore) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*Session, error) {
	userKey := r.prefix + "user:" + userID.String()
	sessionIDs, err := r.client.SMembers(ctx, userKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	var sessions []*Session
	for _, sessionID := range sessionIDs {
		session, err := r.Get(ctx, sessionID)
		if err == nil {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// Cleanup removes expired sessions (handled automatically by Redis TTL)
func (r *RedisSessionStore) Cleanup(ctx context.Context) error {
	return nil // Redis automatically expires keys
}

// DatabaseSessionStore implements database-based session storage
type DatabaseSessionStore struct {
	db *pgxpool.Pool
}

// NewDatabaseSessionStore creates a new database session store
func NewDatabaseSessionStore(db *pgxpool.Pool) *DatabaseSessionStore {
	return &DatabaseSessionStore{db: db}
}

// Create creates a new session
func (d *DatabaseSessionStore) Create(ctx context.Context, session *Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}
	session.UpdatedAt = time.Now()

	dataJSON, err := json.Marshal(session.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	query := `
		INSERT INTO sessions (id, user_id, token, data, created_at, updated_at, expires_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			token = EXCLUDED.token,
			data = EXCLUDED.data,
			updated_at = EXCLUDED.updated_at,
			expires_at = EXCLUDED.expires_at
	`

	_, err = d.db.Exec(ctx, query,
		session.ID,
		session.UserID,
		session.Token,
		dataJSON,
		session.CreatedAt,
		session.UpdatedAt,
		session.ExpiresAt,
		session.IPAddress,
		session.UserAgent,
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// Get retrieves a session by ID
func (d *DatabaseSessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	query := `
		SELECT id, user_id, token, data, created_at, updated_at, expires_at, ip_address, user_agent
		FROM sessions
		WHERE id = $1 AND expires_at > NOW()
	`

	var session Session
	var dataJSON []byte

	err := d.db.QueryRow(ctx, query, sessionID).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&dataJSON,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.ExpiresAt,
		&session.IPAddress,
		&session.UserAgent,
	)

	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	if len(dataJSON) > 0 {
		err = json.Unmarshal(dataJSON, &session.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
		}
	}

	return &session, nil
}

// Update updates an existing session
func (d *DatabaseSessionStore) Update(ctx context.Context, session *Session) error {
	return d.Create(ctx, session)
}

// Delete deletes a session
func (d *DatabaseSessionStore) Delete(ctx context.Context, sessionID string) error {
	query := `DELETE FROM sessions WHERE id = $1`
	_, err := d.db.Exec(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// DeleteByUserID deletes all sessions for a user
func (d *DatabaseSessionStore) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM sessions WHERE user_id = $1`
	_, err := d.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}

// GetByUserID retrieves all sessions for a user
func (d *DatabaseSessionStore) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*Session, error) {
	query := `
		SELECT id, user_id, token, data, created_at, updated_at, expires_at, ip_address, user_agent
		FROM sessions
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY created_at DESC
	`

	rows, err := d.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var session Session
		var dataJSON []byte

		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.Token,
			&dataJSON,
			&session.Created At,
			&session.UpdatedAt,
			&session.ExpiresAt,
			&session.IPAddress,
			&session.UserAgent,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		if len(dataJSON) > 0 {
			err = json.Unmarshal(dataJSON, &session.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
			}
		}

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// Cleanup removes expired sessions
func (d *DatabaseSessionStore) Cleanup(ctx context.Context) error {
	query := `DELETE FROM sessions WHERE expires_at <= NOW()`
	_, err := d.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to cleanup sessions: %w", err)
	}
	return nil
}

// SessionManager manages user sessions with automatic cleanup
type SessionManager struct {
	store          SessionStore
	cleanupTicker  *time.Ticker
	cleanupDone    chan bool
}

// NewSessionManager creates a new session manager with automatic cleanup
func NewSessionManager(store SessionStore, cleanupInterval time.Duration) *SessionManager {
	if cleanupInterval == 0 {
		cleanupInterval = 1 * time.Hour
	}

	manager := &SessionManager{
		store:         store,
		cleanupTicker: time.NewTicker(cleanupInterval),
		cleanupDone:   make(chan bool),
	}

	// Start cleanup goroutine
	go func() {
		for {
			select {
			case <-manager.cleanupTicker.C:
				ctx := context.Background()
				manager.store.Cleanup(ctx)
			case <-manager.cleanupDone:
				return
			}
		}
	}()

	return manager
}

// CreateSession creates a new session
func (s *SessionManager) CreateSession(ctx context.Context, userID uuid.UUID, duration time.Duration, data map[string]interface{}) (*Session, ierrors.Error) {
	if duration == 0 {
		duration = 24 * time.Hour
	}

	session := &Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		Data:      data,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(duration),
	}

	err := s.store.Create(ctx, session)
	if err != nil {
		return nil, errors.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// GetSession retrieves a session
func (s *SessionManager) GetSession(ctx context.Context, sessionID string) (*Session, ierrors.Error) {
	session, err := s.store.Get(ctx, sessionID)
	if err != nil {
		return nil, errors.Errorf("failed to get session: %w", err)
	}
	return session, nil
}

// InvalidateSession deletes a session
func (s *SessionManager) InvalidateSession(ctx context.Context, sessionID string) ierrors.Error {
	err := s.store.Delete(ctx, sessionID)
	if err != nil {
		return errors.Errorf("failed to invalidate session: %w", err)
	}
	return nil
}

// InvalidateAllUserSessions deletes all sessions for a user
func (s *SessionManager) InvalidateAllUserSessions(ctx context.Context, userID uuid.UUID) ierrors.Error {
	err := s.store.DeleteByUserID(ctx, userID)
	if err != nil {
		return errors.Errorf("failed to invalidate user sessions: %w", err)
	}
	return nil
}

// Stop stops the session manager cleanup goroutine
func (s *SessionManager) Stop() {
	s.cleanupTicker.Stop()
	s.cleanupDone <- true
}
