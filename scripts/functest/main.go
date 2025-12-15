package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/singh-yash129/link/internal/config"
	"github.com/singh-yash129/link/internal/logger"
	"github.com/singh-yash129/link/internal/stealth"
	"github.com/singh-yash129/link/internal/storage"
)

// TestResult represents a single test result
type TestResult struct {
	Category string
	Name     string
	Passed   bool
	Details  string
	Duration time.Duration
}

var results []TestResult
var passCount, failCount int

func main() {
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("  LinkedIn Automation Tool - Functional Test Suite")
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println()

	// Test Configuration
	testConfiguration()

	// Test Logger
	testLogger()

	// Test Storage
	testStorage()

	// Test Stealth Manager
	testStealthManager()

	// Test Rate Limiter
	testRateLimiter()

	// Print Summary
	printSummary()
}

func test(category, name string, testFunc func() error) {
	start := time.Now()
	err := testFunc()
	duration := time.Since(start)

	passed := err == nil
	details := ""
	if err != nil {
		details = err.Error()
	}

	results = append(results, TestResult{category, name, passed, details, duration})

	status := "✅ PASS"
	if !passed {
		status = "❌ FAIL"
		failCount++
	} else {
		passCount++
	}

	fmt.Printf("%s | %s (%v)\n", status, name, duration.Round(time.Millisecond))
	if details != "" {
		fmt.Printf("         └── %s\n", details)
	}
}

// ============================================================================
// CONFIGURATION TESTS
// ============================================================================

func testConfiguration() {
	fmt.Println("\n⚙️  CONFIGURATION TESTS")
	fmt.Println(strings.Repeat("-", 60))

	// Set dummy credentials for testing
	os.Setenv("LINKEDIN_EMAIL", "test@example.com")
	os.Setenv("LINKEDIN_PASSWORD", "testpassword123")
	defer func() {
		os.Unsetenv("LINKEDIN_EMAIL")
		os.Unsetenv("LINKEDIN_PASSWORD")
	}()

	test("Config", "Load config.yaml", func() error {
		cfg, err := config.LoadConfig("config.yaml")
		if err != nil {
			// Try example config
			cfg, err = config.LoadConfig("config.yaml.example")
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
		}
		if cfg == nil {
			return fmt.Errorf("config is nil")
		}
		return nil
	})

	test("Config", "Validate required fields exist", func() error {
		cfg, err := config.LoadConfig("config.yaml")
		if err != nil {
			cfg, _ = config.LoadConfig("config.yaml.example")
		}
		if cfg == nil {
			return fmt.Errorf("no config loaded")
		}

		// Check critical fields
		if cfg.Browser.UserDataDir == "" {
			return fmt.Errorf("Browser.UserDataDir is empty")
		}
		if cfg.Connection.DailyLimit == 0 {
			return fmt.Errorf("Connection.DailyLimit is 0")
		}
		return nil
	})

	test("Config", "Environment variable override works", func() error {
		// Env vars already set at start of testConfiguration
		cfg, err := config.LoadConfig("config.yaml")
		if err != nil {
			cfg, _ = config.LoadConfig("config.yaml.example")
		}
		if cfg == nil {
			return fmt.Errorf("no config loaded")
		}

		// Check that env override works (we set test@example.com at start)
		if cfg.LinkedIn.Email != "test@example.com" {
			return fmt.Errorf("env override didn't work, got: %s", cfg.LinkedIn.Email)
		}
		return nil
	})

	test("Config", "Default values are set", func() error {
		cfg, err := config.LoadConfig("config.yaml")
		if err != nil {
			cfg, _ = config.LoadConfig("config.yaml.example")
		}
		if cfg == nil {
			return fmt.Errorf("no config loaded")
		}

		// These should have sensible defaults
		if cfg.RateLimits.MinActionDelay == 0 {
			return fmt.Errorf("MinActionDelay should have default")
		}
		return nil
	})
}

// ============================================================================
// LOGGER TESTS
// ============================================================================

func testLogger() {
	fmt.Println("\n📝 LOGGER TESTS")
	fmt.Println(strings.Repeat("-", 60))

	test("Logger", "Create logger instance", func() error {
		log := logger.New("debug")
		if log == nil {
			return fmt.Errorf("logger is nil")
		}
		return nil
	})

	test("Logger", "Log levels work (Debug)", func() error {
		log := logger.New("debug")
		log.Debug("Test debug message", map[string]interface{}{"key": "value"})
		return nil
	})

	test("Logger", "Log levels work (Info)", func() error {
		log := logger.New("info")
		log.Info("Test info message", map[string]interface{}{"key": "value"})
		return nil
	})

	test("Logger", "Log levels work (Warn)", func() error {
		log := logger.New("warn")
		log.Warn("Test warn message", map[string]interface{}{"key": "value"})
		return nil
	})

	test("Logger", "Log levels work (Error)", func() error {
		log := logger.New("error")
		log.Error("Test error message", map[string]interface{}{"key": "value"})
		return nil
	})

	test("Logger", "WithField contextual logging", func() error {
		log := logger.New("debug")
		log.Info("Test with fields", map[string]interface{}{
			"user":   "testuser",
			"action": "test",
			"count":  42,
		})
		return nil
	})
}

// ============================================================================
// STORAGE TESTS
// ============================================================================

func testStorage() {
	fmt.Println("\n💾 STORAGE TESTS")
	fmt.Println(strings.Repeat("-", 60))

	// Create test config
	cfg := &config.Config{
		Storage: config.StorageConfig{
			Type:         "sqlite",
			DatabasePath: "data/test_storage.db",
		},
	}
	log := logger.New("error") // Quiet logging for tests

	test("Storage", "Initialize SQLite storage", func() error {
		store := storage.NewSQLiteStorage(cfg, log)
		if err := store.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize: %w", err)
		}
		defer store.Close()
		return nil
	})

	test("Storage", "Save and retrieve profile", func() error {
		store := storage.NewSQLiteStorage(cfg, log)
		if err := store.Initialize(); err != nil {
			return err
		}
		defer store.Close()

		profileID := fmt.Sprintf("test-%d", time.Now().UnixNano())
		profile := &storage.Profile{
			ID:         profileID,
			ProfileURL: "https://linkedin.com/in/testuser-" + profileID,
			Name:       "Test User",
			Title:      "Software Engineer",
			Company:    "Test Corp",
			Location:   "San Francisco",
			CreatedAt:  time.Now(),
		}

		if err := store.SaveProfile(profile); err != nil {
			return fmt.Errorf("failed to save profile: %w", err)
		}

		retrieved, err := store.GetProfile(profileID)
		if err != nil {
			return fmt.Errorf("failed to retrieve profile: %w", err)
		}
		if retrieved.Name != profile.Name {
			return fmt.Errorf("name mismatch: got %s, want %s", retrieved.Name, profile.Name)
		}
		return nil
	})

	test("Storage", "Check if profile exists (ProfileExists)", func() error {
		store := storage.NewSQLiteStorage(cfg, log)
		if err := store.Initialize(); err != nil {
			return err
		}
		defer store.Close()

		exists := store.ProfileExists("https://linkedin.com/in/nonexistent-user-12345")
		if exists {
			return fmt.Errorf("non-existent profile should not exist")
		}
		return nil
	})

	test("Storage", "Track connection request (SaveConnectionRequest)", func() error {
		store := storage.NewSQLiteStorage(cfg, log)
		if err := store.Initialize(); err != nil {
			return err
		}
		defer store.Close()

		profileURL := "https://linkedin.com/in/testconn-" + fmt.Sprintf("%d", time.Now().UnixNano())

		// Should not exist initially
		sent := store.WasConnectionSent(profileURL)
		if sent {
			return fmt.Errorf("connection should not exist initially")
		}

		// Record connection
		req := &storage.ConnectionRequest{
			ProfileURL:  profileURL,
			ProfileName: "Test User",
			Note:        "Test note",
			Status:      "pending",
			SentAt:      time.Now(),
		}
		if err := store.SaveConnectionRequest(req); err != nil {
			return fmt.Errorf("failed to save connection request: %w", err)
		}

		// Should exist now
		sent = store.WasConnectionSent(profileURL)
		if !sent {
			return fmt.Errorf("connection should exist after recording")
		}
		return nil
	})

	test("Storage", "Track message sent (SaveMessage)", func() error {
		store := storage.NewSQLiteStorage(cfg, log)
		if err := store.Initialize(); err != nil {
			return err
		}
		defer store.Close()

		profileURL := "https://linkedin.com/in/testmsg-" + fmt.Sprintf("%d", time.Now().UnixNano())

		// Should not exist initially
		sent := store.WasMessageSent(profileURL)
		if sent {
			return fmt.Errorf("message should not exist initially")
		}

		// Record message
		msg := &storage.Message{
			ProfileURL:  profileURL,
			ProfileName: "Test User",
			Content:     "Hello!",
			Direction:   "sent",
			SentAt:      time.Now(),
		}
		if err := store.SaveMessage(msg); err != nil {
			return fmt.Errorf("failed to save message: %w", err)
		}

		// Should exist now
		sent = store.WasMessageSent(profileURL)
		if !sent {
			return fmt.Errorf("message should exist after recording")
		}
		return nil
	})

	test("Storage", "Get connection stats", func() error {
		store := storage.NewSQLiteStorage(cfg, log)
		if err := store.Initialize(); err != nil {
			return err
		}
		defer store.Close()

		stats, err := store.GetConnectionStats()
		if err != nil {
			return fmt.Errorf("failed to get connection stats: %w", err)
		}
		_ = stats // Just check it doesn't error
		return nil
	})

	test("Storage", "Get message stats", func() error {
		store := storage.NewSQLiteStorage(cfg, log)
		if err := store.Initialize(); err != nil {
			return err
		}
		defer store.Close()

		stats, err := store.GetMessageStats()
		if err != nil {
			return fmt.Errorf("failed to get message stats: %w", err)
		}
		_ = stats
		return nil
	})

	// Clean up test database
	os.Remove("data/test_storage.db")
}

// ============================================================================
// STEALTH MANAGER TESTS
// ============================================================================

func testStealthManager() {
	fmt.Println("\n🛡️  STEALTH MANAGER TESTS")
	fmt.Println(strings.Repeat("-", 60))

	cfg := &config.Config{
		Stealth: config.StealthConfig{
			EnableMouseSimulation:  true,
			EnableTypingSimulation: true,
			EnableScrollSimulation: true,
			EnableHoverEvents:      true,
			EnableWebDriverMasking: true,
			MouseSpeed:             1.0,
			TypingSpeedWPM:         60,
			TypoFrequency:          0.02,
			ScrollVariation:        0.1,
		},
		RateLimits: config.RateLimitConfig{
			MinActionDelay: 500,
			MaxActionDelay: 2000,
			MinPageDelay:   1000,
			MaxPageDelay:   3000,
		},
		Schedule: config.ScheduleConfig{
			Enabled:         true,
			Timezone:        "America/New_York",
			StartHour:       9,
			EndHour:         17,
			WorkDaysOnly:    true,
			EnableBreaks:    true,
			BreakFrequency:  60,
			MinBreakMinutes: 5,
			MaxBreakMinutes: 15,
		},
	}
	log := logger.New("error")

	test("Stealth", "Create StealthManager instance", func() error {
		sm := stealth.NewStealthManager(cfg, log)
		if sm == nil {
			return fmt.Errorf("StealthManager is nil")
		}
		return nil
	})

	test("Stealth", "RandomDelay works (timing)", func() error {
		sm := stealth.NewStealthManager(cfg, log)
		start := time.Now()
		sm.RandomDelay()
		elapsed := time.Since(start)
		// Should have waited at least min delay
		if elapsed < time.Duration(cfg.RateLimits.MinActionDelay)*time.Millisecond {
			return fmt.Errorf("delay too short: %v", elapsed)
		}
		return nil
	})

	test("Stealth", "ThinkDelay (think time) works", func() error {
		sm := stealth.NewStealthManager(cfg, log)
		start := time.Now()
		sm.ThinkDelay()
		elapsed := time.Since(start)
		// Should have waited at least 500ms
		if elapsed < 500*time.Millisecond {
			return fmt.Errorf("think delay too short: %v", elapsed)
		}
		return nil
	})

	test("Stealth", "IsWithinSchedule (business hours)", func() error {
		sm := stealth.NewStealthManager(cfg, log)
		// Just check it doesn't panic and returns a boolean
		result := sm.IsWithinSchedule()
		_ = result
		return nil
	})

	test("Stealth", "ShouldTakeBreak (break patterns)", func() error {
		sm := stealth.NewStealthManager(cfg, log)
		// Just check it doesn't panic
		result := sm.ShouldTakeBreak()
		_ = result
		return nil
	})

	test("Stealth", "Retry with exponential backoff - success case", func() error {
		sm := stealth.NewStealthManager(cfg, log)

		attempts := 0
		err := sm.Retry(func() error {
			attempts++
			if attempts < 2 {
				return fmt.Errorf("simulated failure")
			}
			return nil // Success on second attempt
		}, stealth.RetryConfig{
			MaxRetries:    3,
			InitialDelay:  10 * time.Millisecond,
			MaxDelay:      100 * time.Millisecond,
			BackoffFactor: 2.0,
			RetryableError: func(err error) bool {
				return true
			},
		})

		if err != nil {
			return fmt.Errorf("retry should have succeeded: %w", err)
		}
		if attempts != 2 {
			return fmt.Errorf("expected 2 attempts, got %d", attempts)
		}
		return nil
	})

	test("Stealth", "Retry with exponential backoff - failure case", func() error {
		sm := stealth.NewStealthManager(cfg, log)

		err := sm.Retry(func() error {
			return fmt.Errorf("always fails")
		}, stealth.RetryConfig{
			MaxRetries:    2,
			InitialDelay:  5 * time.Millisecond,
			MaxDelay:      20 * time.Millisecond,
			BackoffFactor: 2.0,
			RetryableError: func(err error) bool {
				return true
			},
		})

		if err == nil {
			return fmt.Errorf("retry should have failed after max retries")
		}
		return nil
	})

	test("Stealth", "RetryWithDefault works", func() error {
		sm := stealth.NewStealthManager(cfg, log)

		err := sm.RetryWithDefault(func() error {
			return nil // Immediate success
		})
		if err != nil {
			return fmt.Errorf("should have succeeded: %w", err)
		}
		return nil
	})
}

// ============================================================================
// RATE LIMITER TESTS
// ============================================================================

func testRateLimiter() {
	fmt.Println("\n⏱️  RATE LIMITER TESTS")
	fmt.Println(strings.Repeat("-", 60))

	cfg := &config.Config{
		Connection: config.ConnectionConfig{
			HourlyLimit: 10,
			DailyLimit:  50,
		},
		Messaging: config.MessagingConfig{
			DailyMessageLimit: 20,
		},
		RateLimits: config.RateLimitConfig{
			BatchSize:          5,
			CooldownAfterBatch: 1, // 1 minute
		},
	}
	log := logger.New("error")

	test("RateLimiter", "Create RateLimiter instance", func() error {
		rl := stealth.NewRateLimiter(cfg, log)
		if rl == nil {
			return fmt.Errorf("RateLimiter is nil")
		}
		return nil
	})

	test("RateLimiter", "CanProceed allows initial requests", func() error {
		rl := stealth.NewRateLimiter(cfg, log)
		if !rl.CanProceed("connection") {
			return fmt.Errorf("should allow initial connection request")
		}
		return nil
	})

	test("RateLimiter", "RecordAction increments counter", func() error {
		rl := stealth.NewRateLimiter(cfg, log)

		initialHourly := rl.GetRemainingHourly()
		rl.RecordAction("connection")
		afterHourly := rl.GetRemainingHourly()

		if afterHourly >= initialHourly {
			return fmt.Errorf("counter should have decreased")
		}
		return nil
	})

	test("RateLimiter", "Hourly limit enforced", func() error {
		limitCfg := &config.Config{
			Connection: config.ConnectionConfig{
				HourlyLimit: 2,
				DailyLimit:  100,
			},
			RateLimits: config.RateLimitConfig{
				BatchSize:          100,
				CooldownAfterBatch: 0,
			},
		}
		rl := stealth.NewRateLimiter(limitCfg, log)

		// Record up to limit
		for i := 0; i < 2; i++ {
			rl.RecordAction("connection")
		}

		// Should now be blocked
		if rl.CanProceed("connection") {
			return fmt.Errorf("should be blocked after reaching hourly limit")
		}
		return nil
	})

	test("RateLimiter", "Daily limit enforced", func() error {
		limitCfg := &config.Config{
			Connection: config.ConnectionConfig{
				HourlyLimit: 100,
				DailyLimit:  2,
			},
			RateLimits: config.RateLimitConfig{
				BatchSize:          100,
				CooldownAfterBatch: 0,
			},
		}
		rl := stealth.NewRateLimiter(limitCfg, log)

		// Record up to limit
		for i := 0; i < 2; i++ {
			rl.RecordAction("connection")
		}

		// Should now be blocked
		if rl.CanProceed("connection") {
			return fmt.Errorf("should be blocked after reaching daily limit")
		}
		return nil
	})

	test("RateLimiter", "GetRemainingHourly returns correct value", func() error {
		rl := stealth.NewRateLimiter(cfg, log)
		remaining := rl.GetRemainingHourly()
		if remaining != cfg.Connection.HourlyLimit {
			return fmt.Errorf("expected %d, got %d", cfg.Connection.HourlyLimit, remaining)
		}
		return nil
	})

	test("RateLimiter", "GetRemainingDaily returns correct value", func() error {
		rl := stealth.NewRateLimiter(cfg, log)
		remaining := rl.GetRemainingDaily()
		if remaining != cfg.Connection.DailyLimit {
			return fmt.Errorf("expected %d, got %d", cfg.Connection.DailyLimit, remaining)
		}
		return nil
	})
}

// ============================================================================
// SUMMARY
// ============================================================================

func printSummary() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📊 FUNCTIONAL TEST SUMMARY")
	fmt.Println(strings.Repeat("=", 80))

	total := passCount + failCount
	percentage := float64(passCount) / float64(total) * 100

	fmt.Printf("\n  Total Tests: %d\n", total)
	fmt.Printf("  ✅ Passed: %d\n", passCount)
	fmt.Printf("  ❌ Failed: %d\n", failCount)
	fmt.Printf("  📈 Pass Rate: %.1f%%\n", percentage)

	// Calculate total duration
	var totalDuration time.Duration
	for _, r := range results {
		totalDuration += r.Duration
	}
	fmt.Printf("  ⏱️  Total Time: %v\n", totalDuration.Round(time.Millisecond))

	fmt.Println("\n" + strings.Repeat("-", 80))

	if percentage >= 100 {
		fmt.Println("  🎉 ALL TESTS PASSED! Core functionality is working.")
	} else if percentage >= 80 {
		fmt.Println("  👍 GOOD! Most tests passed.")
	} else if percentage >= 50 {
		fmt.Println("  ⚠️  WARNING! Several tests failed.")
	} else {
		fmt.Println("  ❌ CRITICAL! Many tests failed.")
	}

	if failCount > 0 {
		fmt.Println("\n  ❌ Failed Tests:")
		for _, r := range results {
			if !r.Passed {
				fmt.Printf("    • [%s] %s\n", r.Category, r.Name)
				if r.Details != "" {
					fmt.Printf("      └── %s\n", r.Details)
				}
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 80))

	// Exit with appropriate code
	if failCount > 0 {
		os.Exit(1)
	}
}
