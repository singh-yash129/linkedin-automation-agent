// Package main provides the main entry point for the LinkedIn automation tool.
// It orchestrates all components and provides a CLI interface.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/singh-yash129/link/internal/auth"
	"github.com/singh-yash129/link/internal/browser"
	"github.com/singh-yash129/link/internal/config"
	"github.com/singh-yash129/link/internal/connection"
	"github.com/singh-yash129/link/internal/logger"
	"github.com/singh-yash129/link/internal/messaging"
	"github.com/singh-yash129/link/internal/search"
	"github.com/singh-yash129/link/internal/stealth"
	"github.com/singh-yash129/link/internal/storage"
)

// Version information
const (
	Version = "1.0.0"
	AppName = "LinkedIn Automation Tool"
)

// App holds all application components.
type App struct {
	config      *config.Config
	log         *logger.Logger
	storage     storage.Storage
	browser     *browser.BrowserManager
	stealth     *stealth.StealthManager
	rateLimiter *stealth.RateLimiter
	auth        *auth.AuthManager
	search      *search.SearchManager
	connection  *connection.ConnectionManager
	messaging   *messaging.MessagingManager
}

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	envPath := flag.String("env", ".env", "Path to .env file")
	mode := flag.String("mode", "full", "Operation mode: full, search, connect, message, check")
	headless := flag.Bool("headless", false, "Run in headless mode")
	debug := flag.Bool("debug", false, "Enable debug logging")
	version := flag.Bool("version", false, "Show version information")
	keywords := flag.String("keywords", "", "Search keywords (overrides config)")
	limit := flag.Int("limit", 0, "Limit number of operations (0 = use config)")
	dryRun := flag.Bool("dry-run", false, "Simulate actions without actually performing them")
	flag.Parse()

	// Show version
	if *version {
		fmt.Printf("%s v%s\n", AppName, Version)
		os.Exit(0)
	}

	// Load environment variables
	if err := godotenv.Load(*envPath); err != nil {
		// .env file is optional
		fmt.Printf("Note: No .env file found at %s\n", *envPath)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Override from flags
	if *headless {
		cfg.Browser.Headless = true
	}
	if *debug {
		cfg.Logging.Level = "debug"
	}
	if *keywords != "" {
		cfg.Search.Keywords = []string{*keywords}
	}
	if *limit > 0 {
		cfg.Connection.DailyLimit = *limit
		cfg.Connection.HourlyLimit = *limit
	}
	cfg.DryRun = *dryRun

	// Initialize logger
	log := logger.New(cfg.Logging.Level)

	log.Info("Starting LinkedIn Automation Tool", map[string]interface{}{
		"version": Version,
		"mode":    *mode,
	})

	// Create application
	app := &App{
		config: cfg,
		log:    log,
	}

	// Initialize components
	if err := app.initialize(); err != nil {
		log.Fatal("Failed to initialize application", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Setup graceful shutdown
	app.setupShutdown()

	// Run based on mode
	switch *mode {
	case "full":
		app.runFullMode()
	case "search":
		app.runSearchMode()
	case "connect":
		app.runConnectMode()
	case "message":
		app.runMessageMode()
	case "check":
		app.runCheckMode()
	default:
		log.Error("Unknown mode", map[string]interface{}{
			"mode": *mode,
		})
		os.Exit(1)
	}
}

// initialize sets up all application components.
func (app *App) initialize() error {
	app.log.Info("Initializing components...", nil)

	// Initialize storage based on type
	var store storage.Storage
	if app.config.Storage.Type == "json" {
		jsonStore := storage.NewJSONStorage(app.config, app.log)
		if err := jsonStore.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize JSON storage: %w", err)
		}
		store = jsonStore
	} else {
		sqlStore := storage.NewSQLiteStorage(app.config, app.log)
		if err := sqlStore.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize SQLite storage: %w", err)
		}
		store = sqlStore
	}
	app.storage = store

	// Initialize stealth manager
	app.stealth = stealth.NewStealthManager(app.config, app.log)

	// Initialize rate limiter
	app.rateLimiter = stealth.NewRateLimiter(app.config, app.log)

	// Initialize browser manager
	app.browser = browser.NewBrowserManager(app.config, app.log)
	if err := app.browser.Launch(); err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}

	// Initialize auth manager
	app.auth = auth.NewAuthManager(app.browser, app.config, app.log, app.stealth)

	// Initialize search manager
	app.search = search.NewSearchManager(app.browser, app.config, app.log, app.stealth, app.storage)

	// Initialize connection manager
	app.connection = connection.NewConnectionManager(
		app.browser, app.config, app.log, app.stealth, app.storage, app.rateLimiter,
	)

	// Initialize messaging manager
	app.messaging = messaging.NewMessagingManager(
		app.browser, app.config, app.log, app.stealth, app.storage, app.rateLimiter,
	)

	app.log.Info("All components initialized successfully", nil)
	return nil
}

// setupShutdown configures graceful shutdown handling.
func (app *App) setupShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		app.log.Info("Received shutdown signal", map[string]interface{}{
			"signal": sig.String(),
		})
		app.cleanup()
		os.Exit(0)
	}()
}

// cleanup performs cleanup operations.
func (app *App) cleanup() {
	app.log.Info("Cleaning up...", nil)

	if app.browser != nil {
		app.browser.Close()
	}
	if app.storage != nil {
		app.storage.Close()
	}

	app.log.Info("Cleanup complete", nil)
}

// login performs LinkedIn login.
func (app *App) login() error {
	app.log.Info("Attempting login...", nil)

	if err := app.auth.Login(); err != nil {
		// Check if it's a security challenge
		if authErr, ok := err.(*auth.AuthError); ok {
			switch authErr.Type {
			case auth.ErrTypeCaptcha, auth.ErrType2FA, auth.ErrTypePhoneVerify:
				app.log.Warn("Manual verification required", map[string]interface{}{
					"type": authErr.Type,
				})
				return app.auth.HandleSecurityChallenge()
			}
		}
		return err
	}

	return nil
}

// runFullMode runs the complete automation workflow.
func (app *App) runFullMode() {
	app.log.Info("Running full automation mode", nil)

	// Login
	if err := app.login(); err != nil {
		app.log.Error("Login failed", map[string]interface{}{
			"error": err.Error(),
		})
		app.cleanup()
		os.Exit(1)
	}

	// Main loop
	for {
		// Check schedule
		app.stealth.WaitForSchedule()

		// Check for breaks
		if app.stealth.ShouldTakeBreak() {
			app.stealth.TakeBreak()
		}

		// Search for profiles
		app.log.Info("Starting search phase...", nil)
		filters := search.SearchFilters{
			Keywords:   app.config.Search.Keywords[0],
			MaxResults: app.config.Search.MaxResults,
		}
		if len(app.config.Search.Locations) > 0 {
			filters.Location = app.config.Search.Locations[0]
		}
		if len(app.config.Search.Companies) > 0 {
			filters.Company = app.config.Search.Companies[0]
		}

		results, err := app.search.Search(filters)
		if err != nil {
			app.log.Warn("Search failed", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			app.log.Info("Search completed", map[string]interface{}{
				"results": len(results),
			})

			// Save results
			app.search.SaveResults(results)
		}

		// Filter profiles
		filteredProfiles := app.search.FilterResults(results, true, true)

		app.log.Info("Profiles for connection", map[string]interface{}{
			"total":    len(results),
			"filtered": len(filteredProfiles),
		})

		// Send connection requests
		if len(filteredProfiles) > 0 {
			app.log.Info("Starting connection phase...", nil)
			note := ""
			if app.config.Connection.SendNote && len(app.config.Connection.NoteTemplates) > 0 {
				note = app.config.Connection.NoteTemplates[0]
			}
			result := app.connection.SendBulkConnections(filteredProfiles, note)
			app.log.Info("Connection phase completed", map[string]interface{}{
				"sent":   result.Succeeded,
				"failed": result.Failed,
			})
		}

		// Display stats
		app.displayStats()

		// Wait before next cycle
		app.log.Info("Cycle complete, waiting for next iteration...", nil)
		time.Sleep(30 * time.Minute)
	}
}

// runSearchMode runs only the search phase.
func (app *App) runSearchMode() {
	app.log.Info("Running search mode", nil)

	if err := app.login(); err != nil {
		app.log.Error("Login failed", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Build search filters
	filters := search.SearchFilters{
		MaxResults: app.config.Search.MaxResults,
	}
	if len(app.config.Search.Keywords) > 0 {
		filters.Keywords = app.config.Search.Keywords[0]
	}
	if len(app.config.Search.Locations) > 0 {
		filters.Location = app.config.Search.Locations[0]
	}

	results, err := app.search.Search(filters)
	if err != nil {
		app.log.Error("Search failed", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	app.log.Info("Search completed", map[string]interface{}{
		"results": len(results),
	})

	// Print results
	for i, r := range results {
		fmt.Printf("%d. %s (%s) - %s\n", i+1, r.Name, r.Title, r.ProfileURL)
	}

	// Save results
	app.search.SaveResults(results)

	app.cleanup()
}

// runConnectMode runs only the connection phase.
func (app *App) runConnectMode() {
	app.log.Info("Running connection mode", map[string]interface{}{
		"dry_run": app.config.DryRun,
	})

	if err := app.login(); err != nil {
		app.log.Error("Login failed", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Check if we have profiles, if not run a search first
	profiles, err := app.storage.GetAllProfiles()
	if err != nil {
		app.log.Error("Failed to get profiles", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// If no profiles exist, run a search first
	if len(profiles) == 0 {
		app.log.Info("No stored profiles, running search first...", nil)

		// Build search filters from config
		filters := search.SearchFilters{
			MaxResults: app.config.Search.MaxResults,
		}
		if len(app.config.Search.Keywords) > 0 {
			filters.Keywords = app.config.Search.Keywords[0]
		}
		if len(app.config.Search.Locations) > 0 {
			filters.Location = app.config.Search.Locations[0]
		}

		results, err := app.search.Search(filters)
		if err != nil {
			app.log.Error("Search failed", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}

		// Save search results
		app.search.SaveResults(results)

		// Convert to storage profiles
		for _, r := range results {
			profiles = append(profiles, &storage.Profile{
				ProfileURL: r.ProfileURL,
				Name:       r.Name,
				Title:      r.Title,
				Company:    r.Company,
				Location:   r.Location,
			})
		}
	}

	// Convert to search results for connection manager
	var searchResults []*search.SearchResult
	for _, p := range profiles {
		if !p.IsConnected && !app.storage.WasConnectionSent(p.ProfileURL) {
			searchResults = append(searchResults, &search.SearchResult{
				ProfileURL:   p.ProfileURL,
				Name:         p.Name,
				Title:        p.Title,
				Company:      p.Company,
				Location:     p.Location,
				IsConnection: p.IsConnected,
			})
		}
	}

	if len(searchResults) == 0 {
		app.log.Info("No profiles to connect with", nil)
		app.cleanup()
		return
	}

	app.log.Info("Found profiles to connect", map[string]interface{}{
		"count": len(searchResults),
	})

	// Dry-run mode - just show what would happen
	if app.config.DryRun {
		app.log.Info("DRY RUN - Would send connection requests to:", nil)
		for i, r := range searchResults {
			fmt.Printf("  %d. %s (%s)\n", i+1, r.Name, r.Title)
			if i >= app.config.Connection.DailyLimit-1 {
				break
			}
		}
		app.cleanup()
		return
	}

	// Send connections
	note := ""
	if app.config.Connection.SendNote && len(app.config.Connection.NoteTemplates) > 0 {
		note = app.config.Connection.NoteTemplates[0]
	}

	result := app.connection.SendBulkConnections(searchResults, note)
	app.log.Info("Connection phase completed", map[string]interface{}{
		"sent":    result.Succeeded,
		"failed":  result.Failed,
		"skipped": result.Skipped,
	})

	app.cleanup()
}

// runMessageMode runs only the messaging phase.
func (app *App) runMessageMode() {
	app.log.Info("Running message mode", nil)

	if err := app.login(); err != nil {
		app.log.Error("Login failed", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Get connected profiles from storage
	profiles, err := app.storage.GetAllProfiles()
	if err != nil {
		app.log.Error("Failed to get profiles", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Filter to connected profiles not yet messaged
	var toMessage []*search.SearchResult
	for _, p := range profiles {
		if p.IsConnected && !app.storage.WasMessageSent(p.ProfileURL) {
			toMessage = append(toMessage, &search.SearchResult{
				ProfileURL:   p.ProfileURL,
				Name:         p.Name,
				Title:        p.Title,
				Company:      p.Company,
				IsConnection: true,
			})
		}
	}

	if len(toMessage) == 0 {
		app.log.Info("No connections to message", nil)
		app.cleanup()
		return
	}

	app.log.Info("Found connections to message", map[string]interface{}{
		"count": len(toMessage),
	})

	// Send messages
	message := ""
	if len(app.config.Messaging.MessageTemplates) > 0 {
		message = app.config.Messaging.MessageTemplates[0]
	} else {
		message = "Hi {firstName}, thanks for connecting! I'd love to learn more about what you're working on."
	}

	result := app.messaging.SendBulkMessages(toMessage, message)
	app.log.Info("Messaging phase completed", map[string]interface{}{
		"sent":    result.Succeeded,
		"failed":  result.Failed,
		"skipped": result.Skipped,
	})

	app.cleanup()
}

// runCheckMode runs a status check.
func (app *App) runCheckMode() {
	app.log.Info("Running check mode", nil)

	if err := app.login(); err != nil {
		app.log.Error("Login failed", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Get session info
	sessionInfo := app.auth.GetSessionInfo()
	fmt.Println("\n=== Session Status ===")
	for k, v := range sessionInfo {
		fmt.Printf("  %s: %v\n", k, v)
	}

	// Get stats
	app.displayStats()

	// Get rate limit status
	quota := app.connection.GetRemainingQuota()
	fmt.Println("\n=== Rate Limit Status ===")
	fmt.Printf("  Hourly remaining: %d\n", quota["hourly"])
	fmt.Printf("  Daily remaining: %d\n", quota["daily"])

	app.cleanup()
}

// displayStats displays current statistics.
func (app *App) displayStats() {
	fmt.Println("\n=== Current Statistics ===")

	// Connection stats
	connStats, err := app.storage.GetConnectionStats()
	if err == nil {
		fmt.Println("Connections:")
		for k, v := range connStats {
			fmt.Printf("  %s: %d\n", k, v)
		}
	}

	// Message stats
	msgStats, err := app.storage.GetMessageStats()
	if err == nil {
		fmt.Println("Messages:")
		for k, v := range msgStats {
			fmt.Printf("  %s: %d\n", k, v)
		}
	}

	// Profile count
	profiles, err := app.storage.GetAllProfiles()
	if err == nil {
		fmt.Printf("Total profiles stored: %d\n", len(profiles))
	}
}
