// Package config provides configuration management for the LinkedIn automation tool.
// It supports YAML configuration files with environment variable overrides.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration settings for the automation tool.
type Config struct {
	// LinkedIn credentials
	LinkedIn LinkedInConfig `yaml:"linkedin"`

	// Browser configuration
	Browser BrowserConfig `yaml:"browser"`

	// Stealth/anti-detection settings
	Stealth StealthConfig `yaml:"stealth"`

	// Search parameters
	Search SearchConfig `yaml:"search"`

	// Connection request settings
	Connection ConnectionConfig `yaml:"connection"`

	// Messaging configuration
	Messaging MessagingConfig `yaml:"messaging"`

	// Rate limiting settings
	RateLimits RateLimitConfig `yaml:"rate_limits"`

	// Activity scheduling
	Schedule ScheduleConfig `yaml:"schedule"`

	// Storage settings
	Storage StorageConfig `yaml:"storage"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging"`
}

// LinkedInConfig holds LinkedIn authentication settings.
type LinkedInConfig struct {
	Email    string `yaml:"email"`
	Password string `yaml:"password"`
}

// BrowserConfig holds browser-related settings.
type BrowserConfig struct {
	Headless        bool     `yaml:"headless"`
	SlowMotion      int      `yaml:"slow_motion_ms"`
	Timeout         int      `yaml:"timeout_seconds"`
	UserDataDir     string   `yaml:"user_data_dir"`
	UserAgents      []string `yaml:"user_agents"`
	ViewportWidth   int      `yaml:"viewport_width"`
	ViewportHeight  int      `yaml:"viewport_height"`
	DisableImages   bool     `yaml:"disable_images"`
	ProxyURL        string   `yaml:"proxy_url"`
	ChromePath      string   `yaml:"chrome_path"`
}

// StealthConfig holds anti-detection settings.
type StealthConfig struct {
	EnableMouseSimulation   bool    `yaml:"enable_mouse_simulation"`
	EnableTypingSimulation  bool    `yaml:"enable_typing_simulation"`
	EnableScrollSimulation  bool    `yaml:"enable_scroll_simulation"`
	EnableHoverEvents       bool    `yaml:"enable_hover_events"`
	MouseSpeed              float64 `yaml:"mouse_speed"`
	TypingSpeedWPM          int     `yaml:"typing_speed_wpm"`
	TypoFrequency           float64 `yaml:"typo_frequency"`
	ScrollVariation         float64 `yaml:"scroll_variation"`
	EnableWebDriverMasking  bool    `yaml:"enable_webdriver_masking"`
	RandomizeFingerprint    bool    `yaml:"randomize_fingerprint"`
	EnableCanvasNoise       bool    `yaml:"enable_canvas_noise"`
	EnableWebGLNoise        bool    `yaml:"enable_webgl_noise"`
}

// SearchConfig holds search-related settings.
type SearchConfig struct {
	Keywords      []string `yaml:"keywords"`
	JobTitles     []string `yaml:"job_titles"`
	Companies     []string `yaml:"companies"`
	Locations     []string `yaml:"locations"`
	MaxResults    int      `yaml:"max_results"`
	MaxPages      int      `yaml:"max_pages"`
	FilterBy2nd   bool     `yaml:"filter_by_2nd_degree"`
	FilterBy3rd   bool     `yaml:"filter_by_3rd_degree"`
}

// ConnectionConfig holds connection request settings.
type ConnectionConfig struct {
	DailyLimit           int      `yaml:"daily_limit"`
	HourlyLimit          int      `yaml:"hourly_limit"`
	SendNote             bool     `yaml:"send_note"`
	NoteTemplates        []string `yaml:"note_templates"`
	MaxNoteLength        int      `yaml:"max_note_length"`
	SkipWithoutPhoto     bool     `yaml:"skip_without_photo"`
	SkipOpenToWork       bool     `yaml:"skip_open_to_work"`
	MinMutualConnections int      `yaml:"min_mutual_connections"`
}

// MessagingConfig holds messaging settings.
type MessagingConfig struct {
	EnableFollowUp      bool     `yaml:"enable_follow_up"`
	FollowUpDelayDays   int      `yaml:"follow_up_delay_days"`
	MessageTemplates    []string `yaml:"message_templates"`
	MaxMessageLength    int      `yaml:"max_message_length"`
	DailyMessageLimit   int      `yaml:"daily_message_limit"`
}

// RateLimitConfig holds rate limiting settings.
type RateLimitConfig struct {
	MinActionDelay     int `yaml:"min_action_delay_ms"`
	MaxActionDelay     int `yaml:"max_action_delay_ms"`
	MinPageDelay       int `yaml:"min_page_delay_ms"`
	MaxPageDelay       int `yaml:"max_page_delay_ms"`
	MinTypingDelay     int `yaml:"min_typing_delay_ms"`
	MaxTypingDelay     int `yaml:"max_typing_delay_ms"`
	CooldownAfterBatch int `yaml:"cooldown_after_batch_min"`
	BatchSize          int `yaml:"batch_size"`
}

// ScheduleConfig holds activity scheduling settings.
type ScheduleConfig struct {
	Enabled         bool   `yaml:"enabled"`
	StartHour       int    `yaml:"start_hour"`
	EndHour         int    `yaml:"end_hour"`
	WorkDaysOnly    bool   `yaml:"work_days_only"`
	Timezone        string `yaml:"timezone"`
	EnableBreaks    bool   `yaml:"enable_breaks"`
	MinBreakMinutes int    `yaml:"min_break_minutes"`
	MaxBreakMinutes int    `yaml:"max_break_minutes"`
	BreakFrequency  int    `yaml:"break_frequency_minutes"`
}

// StorageConfig holds data persistence settings.
type StorageConfig struct {
	Type           string `yaml:"type"` // "sqlite" or "json"
	DatabasePath   string `yaml:"database_path"`
	CookiesPath    string `yaml:"cookies_path"`
	ProfilesPath   string `yaml:"profiles_path"`
	BackupEnabled  bool   `yaml:"backup_enabled"`
	BackupInterval int    `yaml:"backup_interval_hours"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level      string `yaml:"level"` // debug, info, warn, error
	Format     string `yaml:"format"` // text, json
	Output     string `yaml:"output"` // stdout, file, both
	FilePath   string `yaml:"file_path"`
	MaxSizeMB  int    `yaml:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups"`
}

// DefaultConfig returns a configuration with sensible default values.
func DefaultConfig() *Config {
	return &Config{
		Browser: BrowserConfig{
			Headless:       false,
			SlowMotion:     0,
			Timeout:        30,
			UserDataDir:    "./data/browser",
			ViewportWidth:  1920,
			ViewportHeight: 1080,
			UserAgents: []string{
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
				"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
			},
		},
		Stealth: StealthConfig{
			EnableMouseSimulation:  true,
			EnableTypingSimulation: true,
			EnableScrollSimulation: true,
			EnableHoverEvents:      true,
			MouseSpeed:             1.0,
			TypingSpeedWPM:         60,
			TypoFrequency:          0.02,
			ScrollVariation:        0.3,
			EnableWebDriverMasking: true,
			RandomizeFingerprint:   true,
			EnableCanvasNoise:      true,
			EnableWebGLNoise:       true,
		},
		Search: SearchConfig{
			MaxResults:  100,
			MaxPages:    10,
			FilterBy2nd: true,
			FilterBy3rd: false,
		},
		Connection: ConnectionConfig{
			DailyLimit:    20,
			HourlyLimit:   5,
			SendNote:      true,
			MaxNoteLength: 300,
			NoteTemplates: []string{
				"Hi {{.FirstName}}, I noticed we share an interest in {{.Industry}}. I'd love to connect and learn more about your work at {{.Company}}.",
				"Hello {{.FirstName}}, I came across your profile and was impressed by your experience in {{.Title}}. Would love to connect!",
			},
		},
		Messaging: MessagingConfig{
			EnableFollowUp:    true,
			FollowUpDelayDays: 3,
			MaxMessageLength:  8000,
			DailyMessageLimit: 50,
			MessageTemplates: []string{
				"Hi {{.FirstName}}, thanks for connecting! I'd love to learn more about your role at {{.Company}}. How are you finding the current market?",
			},
		},
		RateLimits: RateLimitConfig{
			MinActionDelay:     2000,
			MaxActionDelay:     5000,
			MinPageDelay:       3000,
			MaxPageDelay:       8000,
			MinTypingDelay:     50,
			MaxTypingDelay:     150,
			CooldownAfterBatch: 15,
			BatchSize:          5,
		},
		Schedule: ScheduleConfig{
			Enabled:         true,
			StartHour:       9,
			EndHour:         18,
			WorkDaysOnly:    true,
			Timezone:        "America/New_York",
			EnableBreaks:    true,
			MinBreakMinutes: 5,
			MaxBreakMinutes: 15,
			BreakFrequency:  60,
		},
		Storage: StorageConfig{
			Type:           "sqlite",
			DatabasePath:   "./data/linkedin.db",
			CookiesPath:    "./data/cookies.json",
			ProfilesPath:   "./data/profiles.json",
			BackupEnabled:  true,
			BackupInterval: 24,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "text",
			Output:     "both",
			FilePath:   "./logs/linkedin.log",
			MaxSizeMB:  100,
			MaxBackups: 5,
		},
	}
}

// LoadConfig loads configuration from a YAML file with environment variable overrides.
func LoadConfig(path string) (*Config, error) {
	config := DefaultConfig()

	// Load from YAML file if it exists
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
			// File doesn't exist, use defaults
		} else {
			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		}
	}

	// Override with environment variables
	config.applyEnvOverrides()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// applyEnvOverrides applies environment variable overrides to the configuration.
func (c *Config) applyEnvOverrides() {
	// LinkedIn credentials (required from environment for security)
	if email := os.Getenv("LINKEDIN_EMAIL"); email != "" {
		c.LinkedIn.Email = email
	}
	if password := os.Getenv("LINKEDIN_PASSWORD"); password != "" {
		c.LinkedIn.Password = password
	}

	// Browser settings
	if headless := os.Getenv("BROWSER_HEADLESS"); headless != "" {
		c.Browser.Headless = headless == "true" || headless == "1"
	}
	if proxy := os.Getenv("BROWSER_PROXY"); proxy != "" {
		c.Browser.ProxyURL = proxy
	}
	if chromePath := os.Getenv("CHROME_PATH"); chromePath != "" {
		c.Browser.ChromePath = chromePath
	}

	// Rate limits
	if daily := os.Getenv("CONNECTION_DAILY_LIMIT"); daily != "" {
		if val, err := strconv.Atoi(daily); err == nil {
			c.Connection.DailyLimit = val
		}
	}
	if hourly := os.Getenv("CONNECTION_HOURLY_LIMIT"); hourly != "" {
		if val, err := strconv.Atoi(hourly); err == nil {
			c.Connection.HourlyLimit = val
		}
	}

	// Storage paths
	if dbPath := os.Getenv("DATABASE_PATH"); dbPath != "" {
		c.Storage.DatabasePath = dbPath
	}
	if cookiesPath := os.Getenv("COOKIES_PATH"); cookiesPath != "" {
		c.Storage.CookiesPath = cookiesPath
	}

	// Logging
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		c.Logging.Level = logLevel
	}
	if logPath := os.Getenv("LOG_FILE_PATH"); logPath != "" {
		c.Logging.FilePath = logPath
	}
}

// Validate checks if the configuration values are valid.
func (c *Config) Validate() error {
	// LinkedIn credentials are required
	if c.LinkedIn.Email == "" {
		return fmt.Errorf("LinkedIn email is required (set LINKEDIN_EMAIL environment variable)")
	}
	if c.LinkedIn.Password == "" {
		return fmt.Errorf("LinkedIn password is required (set LINKEDIN_PASSWORD environment variable)")
	}

	// Validate rate limits
	if c.Connection.DailyLimit < 0 || c.Connection.DailyLimit > 100 {
		return fmt.Errorf("daily connection limit must be between 0 and 100")
	}
	if c.Connection.HourlyLimit < 0 || c.Connection.HourlyLimit > 20 {
		return fmt.Errorf("hourly connection limit must be between 0 and 20")
	}

	// Validate schedule
	if c.Schedule.Enabled {
		if c.Schedule.StartHour < 0 || c.Schedule.StartHour > 23 {
			return fmt.Errorf("start hour must be between 0 and 23")
		}
		if c.Schedule.EndHour < 0 || c.Schedule.EndHour > 23 {
			return fmt.Errorf("end hour must be between 0 and 23")
		}
		if c.Schedule.StartHour >= c.Schedule.EndHour {
			return fmt.Errorf("start hour must be less than end hour")
		}
	}

	// Validate logging level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.Logging.Level)
	}

	return nil
}

// SaveConfig saves the configuration to a YAML file.
func (c *Config) SaveConfig(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetActionDelay returns a random delay within the configured range.
func (c *Config) GetActionDelay() time.Duration {
	return randomDuration(c.RateLimits.MinActionDelay, c.RateLimits.MaxActionDelay)
}

// GetPageDelay returns a random page navigation delay.
func (c *Config) GetPageDelay() time.Duration {
	return randomDuration(c.RateLimits.MinPageDelay, c.RateLimits.MaxPageDelay)
}

// GetTypingDelay returns a random typing delay.
func (c *Config) GetTypingDelay() time.Duration {
	return randomDuration(c.RateLimits.MinTypingDelay, c.RateLimits.MaxTypingDelay)
}

// randomDuration returns a random duration between min and max milliseconds.
func randomDuration(minMs, maxMs int) time.Duration {
	if minMs >= maxMs {
		return time.Duration(minMs) * time.Millisecond
	}
	// Simple random using current time nanoseconds
	delta := maxMs - minMs
	offset := int(time.Now().UnixNano() % int64(delta))
	return time.Duration(minMs+offset) * time.Millisecond
}
