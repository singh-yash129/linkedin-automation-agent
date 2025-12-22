// Package storage provides data persistence using SQLite and JSON files.
package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/singh-yash129/linkedin-automation-agent/internal/config"
	"github.com/singh-yash129/linkedin-automation-agent/internal/logger"
	_ "modernc.org/sqlite"
)

// Profile represents a LinkedIn profile.
type Profile struct {
	ID            string    `json:"id"`
	ProfileURL    string    `json:"profile_url"`
	Name          string    `json:"name"`
	Title         string    `json:"title"`
	Company       string    `json:"company"`
	Location      string    `json:"location"`
	ConnectionDeg string    `json:"connection_degree"`
	IsConnected   bool      `json:"is_connected"`
	MessageSent   bool      `json:"message_sent"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ConnectionRequest represents a connection request record.
type ConnectionRequest struct {
	ID          int64     `json:"id"`
	ProfileID   string    `json:"profile_id"`
	ProfileURL  string    `json:"profile_url"`
	ProfileName string    `json:"profile_name"`
	Note        string    `json:"note"`
	Status      string    `json:"status"` // pending, accepted, declined
	SentAt      time.Time `json:"sent_at"`
	RespondedAt time.Time `json:"responded_at,omitempty"`
}

// Message represents a message record.
type Message struct {
	ID          int64     `json:"id"`
	ProfileID   string    `json:"profile_id"`
	ProfileURL  string    `json:"profile_url"`
	ProfileName string    `json:"profile_name"`
	Content     string    `json:"content"`
	Direction   string    `json:"direction"` // sent, received
	SentAt      time.Time `json:"sent_at"`
}

// Storage interface for data persistence.
type Storage interface {
	Initialize() error
	Close() error
	// Profile operations
	SaveProfile(profile *Profile) error
	GetProfile(id string) (*Profile, error)
	GetProfileByURL(url string) (*Profile, error)
	GetAllProfiles() ([]*Profile, error)
	UpdateProfile(profile *Profile) error
	DeleteProfile(id string) error
	// Connection operations
	SaveConnectionRequest(req *ConnectionRequest) error
	GetConnectionRequests(status string) ([]*ConnectionRequest, error)
	UpdateConnectionStatus(id int64, status string) error
	GetConnectionStats() (map[string]int, error)
	// Message operations
	SaveMessage(msg *Message) error
	GetMessages(profileID string) ([]*Message, error)
	GetMessageStats() (map[string]int, error)
	// Utility
	ProfileExists(profileURL string) bool
	WasConnectionSent(profileURL string) bool
	WasMessageSent(profileURL string) bool
}

// SQLiteStorage implements Storage using SQLite.
type SQLiteStorage struct {
	db     *sql.DB
	config *config.Config
	log    *logger.Logger
	mu     sync.RWMutex
}

// NewSQLiteStorage creates a new SQLite storage instance.
func NewSQLiteStorage(cfg *config.Config, log *logger.Logger) *SQLiteStorage {
	return &SQLiteStorage{
		config: cfg,
		log:    log,
	}
}

// Initialize sets up the database.
func (s *SQLiteStorage) Initialize() error {
	dbPath := s.config.Storage.DatabasePath

	// Ensure directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	s.db = db

	// Create tables
	if err := s.createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	s.log.Info("SQLite storage initialized", map[string]interface{}{
		"path": dbPath,
	})

	return nil
}

// createTables creates the necessary database tables.
func (s *SQLiteStorage) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS profiles (
			id TEXT PRIMARY KEY,
			profile_url TEXT UNIQUE,
			name TEXT,
			title TEXT,
			company TEXT,
			location TEXT,
			connection_degree TEXT,
			is_connected INTEGER DEFAULT 0,
			message_sent INTEGER DEFAULT 0,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS connection_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_id TEXT,
			profile_url TEXT,
			profile_name TEXT,
			note TEXT,
			status TEXT DEFAULT 'pending',
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			responded_at DATETIME,
			FOREIGN KEY (profile_id) REFERENCES profiles(id)
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_id TEXT,
			profile_url TEXT,
			profile_name TEXT,
			content TEXT,
			direction TEXT,
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (profile_id) REFERENCES profiles(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_profiles_url ON profiles(profile_url)`,
		`CREATE INDEX IF NOT EXISTS idx_connections_status ON connection_requests(status)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_profile ON messages(profile_id)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

// Close closes the database connection.
func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SaveProfile saves a profile to the database.
func (s *SQLiteStorage) SaveProfile(profile *Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `INSERT OR REPLACE INTO profiles 
		(id, profile_url, name, title, company, location, connection_degree, is_connected, message_sent, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now

	_, err := s.db.Exec(query,
		profile.ID, profile.ProfileURL, profile.Name, profile.Title,
		profile.Company, profile.Location, profile.ConnectionDeg,
		profile.IsConnected, profile.MessageSent, profile.Notes,
		profile.CreatedAt, profile.UpdatedAt)

	return err
}

// GetProfile retrieves a profile by ID.
func (s *SQLiteStorage) GetProfile(id string) (*Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, profile_url, name, title, company, location, 
		connection_degree, is_connected, message_sent, notes, created_at, updated_at
		FROM profiles WHERE id = ?`

	var p Profile
	err := s.db.QueryRow(query, id).Scan(
		&p.ID, &p.ProfileURL, &p.Name, &p.Title, &p.Company, &p.Location,
		&p.ConnectionDeg, &p.IsConnected, &p.MessageSent, &p.Notes,
		&p.CreatedAt, &p.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetProfileByURL retrieves a profile by URL.
func (s *SQLiteStorage) GetProfileByURL(url string) (*Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, profile_url, name, title, company, location, 
		connection_degree, is_connected, message_sent, notes, created_at, updated_at
		FROM profiles WHERE profile_url = ?`

	var p Profile
	err := s.db.QueryRow(query, url).Scan(
		&p.ID, &p.ProfileURL, &p.Name, &p.Title, &p.Company, &p.Location,
		&p.ConnectionDeg, &p.IsConnected, &p.MessageSent, &p.Notes,
		&p.CreatedAt, &p.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetAllProfiles retrieves all profiles.
func (s *SQLiteStorage) GetAllProfiles() ([]*Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, profile_url, name, title, company, location, 
		connection_degree, is_connected, message_sent, notes, created_at, updated_at
		FROM profiles ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []*Profile
	for rows.Next() {
		var p Profile
		if err := rows.Scan(
			&p.ID, &p.ProfileURL, &p.Name, &p.Title, &p.Company, &p.Location,
			&p.ConnectionDeg, &p.IsConnected, &p.MessageSent, &p.Notes,
			&p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, &p)
	}
	return profiles, nil
}

// UpdateProfile updates an existing profile.
func (s *SQLiteStorage) UpdateProfile(profile *Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile.UpdatedAt = time.Now()

	query := `UPDATE profiles SET 
		name = ?, title = ?, company = ?, location = ?, 
		connection_degree = ?, is_connected = ?, message_sent = ?, 
		notes = ?, updated_at = ?
		WHERE id = ?`

	_, err := s.db.Exec(query,
		profile.Name, profile.Title, profile.Company, profile.Location,
		profile.ConnectionDeg, profile.IsConnected, profile.MessageSent,
		profile.Notes, profile.UpdatedAt, profile.ID)

	return err
}

// DeleteProfile deletes a profile.
func (s *SQLiteStorage) DeleteProfile(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("DELETE FROM profiles WHERE id = ?", id)
	return err
}

// SaveConnectionRequest saves a connection request.
func (s *SQLiteStorage) SaveConnectionRequest(req *ConnectionRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `INSERT INTO connection_requests 
		(profile_id, profile_url, profile_name, note, status, sent_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	if req.SentAt.IsZero() {
		req.SentAt = time.Now()
	}

	result, err := s.db.Exec(query,
		req.ProfileID, req.ProfileURL, req.ProfileName,
		req.Note, req.Status, req.SentAt)
	if err != nil {
		return err
	}

	req.ID, _ = result.LastInsertId()
	return nil
}

// GetConnectionRequests retrieves connection requests by status.
func (s *SQLiteStorage) GetConnectionRequests(status string) ([]*ConnectionRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var query string
	var rows *sql.Rows
	var err error

	if status == "" {
		query = `SELECT id, profile_id, profile_url, profile_name, note, status, sent_at, responded_at
			FROM connection_requests ORDER BY sent_at DESC`
		rows, err = s.db.Query(query)
	} else {
		query = `SELECT id, profile_id, profile_url, profile_name, note, status, sent_at, responded_at
			FROM connection_requests WHERE status = ? ORDER BY sent_at DESC`
		rows, err = s.db.Query(query, status)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*ConnectionRequest
	for rows.Next() {
		var req ConnectionRequest
		var respondedAt sql.NullTime
		if err := rows.Scan(
			&req.ID, &req.ProfileID, &req.ProfileURL, &req.ProfileName,
			&req.Note, &req.Status, &req.SentAt, &respondedAt); err != nil {
			return nil, err
		}
		if respondedAt.Valid {
			req.RespondedAt = respondedAt.Time
		}
		requests = append(requests, &req)
	}
	return requests, nil
}

// UpdateConnectionStatus updates the status of a connection request.
func (s *SQLiteStorage) UpdateConnectionStatus(id int64, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `UPDATE connection_requests SET status = ?, responded_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, status, time.Now(), id)
	return err
}

// GetConnectionStats returns connection statistics.
func (s *SQLiteStorage) GetConnectionStats() (map[string]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]int)

	query := `SELECT status, COUNT(*) FROM connection_requests GROUP BY status`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
	}

	// Total
	var total int
	s.db.QueryRow("SELECT COUNT(*) FROM connection_requests").Scan(&total)
	stats["total"] = total

	return stats, nil
}

// SaveMessage saves a message.
func (s *SQLiteStorage) SaveMessage(msg *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `INSERT INTO messages 
		(profile_id, profile_url, profile_name, content, direction, sent_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	if msg.SentAt.IsZero() {
		msg.SentAt = time.Now()
	}

	result, err := s.db.Exec(query,
		msg.ProfileID, msg.ProfileURL, msg.ProfileName,
		msg.Content, msg.Direction, msg.SentAt)
	if err != nil {
		return err
	}

	msg.ID, _ = result.LastInsertId()
	return nil
}

// GetMessages retrieves messages for a profile.
func (s *SQLiteStorage) GetMessages(profileID string) ([]*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, profile_id, profile_url, profile_name, content, direction, sent_at
		FROM messages WHERE profile_id = ? ORDER BY sent_at DESC`

	rows, err := s.db.Query(query, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(
			&msg.ID, &msg.ProfileID, &msg.ProfileURL, &msg.ProfileName,
			&msg.Content, &msg.Direction, &msg.SentAt); err != nil {
			return nil, err
		}
		messages = append(messages, &msg)
	}
	return messages, nil
}

// GetMessageStats returns message statistics.
func (s *SQLiteStorage) GetMessageStats() (map[string]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]int)

	query := `SELECT direction, COUNT(*) FROM messages GROUP BY direction`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var direction string
		var count int
		if err := rows.Scan(&direction, &count); err != nil {
			return nil, err
		}
		stats[direction] = count
	}

	// Total
	var total int
	s.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&total)
	stats["total"] = total

	return stats, nil
}

// ProfileExists checks if a profile URL already exists.
func (s *SQLiteStorage) ProfileExists(profileURL string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM profiles WHERE profile_url = ?", profileURL).Scan(&count)
	return count > 0
}

// WasConnectionSent checks if a connection was sent to a profile.
func (s *SQLiteStorage) WasConnectionSent(profileURL string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM connection_requests WHERE profile_url = ?", profileURL).Scan(&count)
	return count > 0
}

// WasMessageSent checks if a message was sent to a profile.
func (s *SQLiteStorage) WasMessageSent(profileURL string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM messages WHERE profile_url = ? AND direction = 'sent'", profileURL).Scan(&count)
	return count > 0
}

// JSONStorage implements Storage using JSON files.
type JSONStorage struct {
	dataDir  string
	log      *logger.Logger
	mu       sync.RWMutex
	profiles map[string]*Profile
	requests []*ConnectionRequest
	messages []*Message
}

// NewJSONStorage creates a new JSON storage instance.
func NewJSONStorage(cfg *config.Config, log *logger.Logger) *JSONStorage {
	dataDir := filepath.Dir(cfg.Storage.DatabasePath)
	return &JSONStorage{
		dataDir:  dataDir,
		log:      log,
		profiles: make(map[string]*Profile),
		requests: []*ConnectionRequest{},
		messages: []*Message{},
	}
}

// Initialize loads existing data from JSON files.
func (s *JSONStorage) Initialize() error {
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return err
	}

	// Load profiles
	profilesFile := filepath.Join(s.dataDir, "profiles.json")
	if data, err := os.ReadFile(profilesFile); err == nil {
		var profiles []*Profile
		if err := json.Unmarshal(data, &profiles); err == nil {
			for _, p := range profiles {
				s.profiles[p.ProfileURL] = p
			}
		}
	}

	// Load connection requests
	requestsFile := filepath.Join(s.dataDir, "connections.json")
	if data, err := os.ReadFile(requestsFile); err == nil {
		json.Unmarshal(data, &s.requests)
	}

	// Load messages
	messagesFile := filepath.Join(s.dataDir, "messages.json")
	if data, err := os.ReadFile(messagesFile); err == nil {
		json.Unmarshal(data, &s.messages)
	}

	s.log.Info("JSON storage initialized", map[string]interface{}{
		"profiles":    len(s.profiles),
		"connections": len(s.requests),
		"messages":    len(s.messages),
	})

	return nil
}

// Close saves data to files.
func (s *JSONStorage) Close() error {
	return s.save()
}

// save persists all data to JSON files.
func (s *JSONStorage) save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Save profiles
	var profiles []*Profile
	for _, p := range s.profiles {
		profiles = append(profiles, p)
	}
	if data, err := json.MarshalIndent(profiles, "", "  "); err == nil {
		os.WriteFile(filepath.Join(s.dataDir, "profiles.json"), data, 0644)
	}

	// Save connections
	if data, err := json.MarshalIndent(s.requests, "", "  "); err == nil {
		os.WriteFile(filepath.Join(s.dataDir, "connections.json"), data, 0644)
	}

	// Save messages
	if data, err := json.MarshalIndent(s.messages, "", "  "); err == nil {
		os.WriteFile(filepath.Join(s.dataDir, "messages.json"), data, 0644)
	}

	return nil
}

// SaveProfile saves a profile.
func (s *JSONStorage) SaveProfile(profile *Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now
	s.profiles[profile.ProfileURL] = profile
	return s.save()
}

// GetProfile retrieves a profile by ID.
func (s *JSONStorage) GetProfile(id string) (*Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.profiles {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, nil
}

// GetProfileByURL retrieves a profile by URL.
func (s *JSONStorage) GetProfileByURL(url string) (*Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.profiles[url], nil
}

// GetAllProfiles retrieves all profiles.
func (s *JSONStorage) GetAllProfiles() ([]*Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var profiles []*Profile
	for _, p := range s.profiles {
		profiles = append(profiles, p)
	}
	return profiles, nil
}

// UpdateProfile updates an existing profile.
func (s *JSONStorage) UpdateProfile(profile *Profile) error {
	return s.SaveProfile(profile)
}

// DeleteProfile deletes a profile.
func (s *JSONStorage) DeleteProfile(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for url, p := range s.profiles {
		if p.ID == id {
			delete(s.profiles, url)
			break
		}
	}
	return s.save()
}

// SaveConnectionRequest saves a connection request.
func (s *JSONStorage) SaveConnectionRequest(req *ConnectionRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.SentAt.IsZero() {
		req.SentAt = time.Now()
	}
	req.ID = int64(len(s.requests) + 1)
	s.requests = append(s.requests, req)
	return s.save()
}

// GetConnectionRequests retrieves connection requests by status.
func (s *JSONStorage) GetConnectionRequests(status string) ([]*ConnectionRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if status == "" {
		return s.requests, nil
	}

	var filtered []*ConnectionRequest
	for _, r := range s.requests {
		if r.Status == status {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

// UpdateConnectionStatus updates the status of a connection request.
func (s *JSONStorage) UpdateConnectionStatus(id int64, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range s.requests {
		if r.ID == id {
			r.Status = status
			r.RespondedAt = time.Now()
			break
		}
	}
	return s.save()
}

// GetConnectionStats returns connection statistics.
func (s *JSONStorage) GetConnectionStats() (map[string]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]int)
	stats["total"] = len(s.requests)

	for _, r := range s.requests {
		stats[r.Status]++
	}
	return stats, nil
}

// SaveMessage saves a message.
func (s *JSONStorage) SaveMessage(msg *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if msg.SentAt.IsZero() {
		msg.SentAt = time.Now()
	}
	msg.ID = int64(len(s.messages) + 1)
	s.messages = append(s.messages, msg)
	return s.save()
}

// GetMessages retrieves messages for a profile.
func (s *JSONStorage) GetMessages(profileID string) ([]*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []*Message
	for _, m := range s.messages {
		if m.ProfileID == profileID {
			filtered = append(filtered, m)
		}
	}
	return filtered, nil
}

// GetMessageStats returns message statistics.
func (s *JSONStorage) GetMessageStats() (map[string]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]int)
	stats["total"] = len(s.messages)

	for _, m := range s.messages {
		stats[m.Direction]++
	}
	return stats, nil
}

// ProfileExists checks if a profile URL already exists.
func (s *JSONStorage) ProfileExists(profileURL string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.profiles[profileURL]
	return exists
}

// WasConnectionSent checks if a connection was sent to a profile.
func (s *JSONStorage) WasConnectionSent(profileURL string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, r := range s.requests {
		if r.ProfileURL == profileURL {
			return true
		}
	}
	return false
}

// WasMessageSent checks if a message was sent to a profile.
func (s *JSONStorage) WasMessageSent(profileURL string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, m := range s.messages {
		if m.ProfileURL == profileURL && m.Direction == "sent" {
			return true
		}
	}
	return false
}
