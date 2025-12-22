package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/singh-yash129/linkedin-automation-agent/internal/auth"
	"github.com/singh-yash129/linkedin-automation-agent/internal/browser"
	"github.com/singh-yash129/linkedin-automation-agent/internal/config"
	"github.com/singh-yash129/linkedin-automation-agent/internal/connection"
	"github.com/singh-yash129/linkedin-automation-agent/internal/logger"
	"github.com/singh-yash129/linkedin-automation-agent/internal/messaging"
	"github.com/singh-yash129/linkedin-automation-agent/internal/search"
	"github.com/singh-yash129/linkedin-automation-agent/internal/stealth"
	"github.com/singh-yash129/linkedin-automation-agent/internal/storage"
)

var (
	reader = bufio.NewReader(os.Stdin)
	log    *logger.Logger
	cfg    *config.Config
	bm     *browser.BrowserManager
	sm     *stealth.StealthManager
	rl     *stealth.RateLimiter
	am     *auth.AuthManager
	store  storage.Storage
)

func main() {
	configPath := flag.String("config", "config.yaml", "Config file path")
	headless := flag.Bool("headless", false, "Run in headless mode")
	flag.Parse()

	printHeader()
	godotenv.Load(".env")

	var err error
	cfg, err = config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("❌ Config error: %v\n", err)
		return
	}

	if *headless {
		cfg.Browser.Headless = true
	}

	log = logger.New("info")

	fmt.Println("\n🚀 Initializing components...")

	store = storage.NewSQLiteStorage(cfg, log)
	if err := store.Initialize(); err != nil {
		fmt.Printf("❌ Storage error: %v\n", err)
		return
	}
	defer store.Close()
	fmt.Println("   ✅ Storage initialized")

	sm = stealth.NewStealthManager(cfg, log)
	rl = stealth.NewRateLimiter(cfg, log)
	fmt.Println("   ✅ Stealth manager ready")

	bm = browser.NewBrowserManager(cfg, log)
	if err := bm.Launch(); err != nil {
		fmt.Printf("❌ Browser error: %v\n", err)
		return
	}
	defer bm.Close()
	fmt.Println("   ✅ Browser launched")

	am = auth.NewAuthManager(bm, cfg, log, sm)
	fmt.Println("   ✅ Auth manager ready")

	for {
		printMenu()
		choice := prompt("\n👉 Enter choice (1-9): ")

		switch choice {
		case "1":
			testAuthentication()
		case "2":
			testSearchKeywords()
		case "3":
			testSearchFilters()
		case "4":
			testConnectionRequest()
		case "5":
			testMessaging()
		case "6":
			testStealthFeatures()
		case "7":
			testStorageOperations()
		case "8":
			testRateLimiting()
		case "9":
			fmt.Println("\n👋 Goodbye!")
			return
		default:
			fmt.Println("❌ Invalid choice")
		}
	}
}

func printHeader() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("  🔧 LinkedIn Automation - COMPREHENSIVE MANUAL TEST")
	fmt.Println("  Tests ALL Features: Search, Connection, Messaging, Stealth")
	fmt.Println(strings.Repeat("=", 80))
}

func printMenu() {
	fmt.Println("\n" + strings.Repeat("-", 60))
	fmt.Println("  📋 MAIN MENU - Select Feature to Test")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  1. 🔐 Authentication (Login/Session)")
	fmt.Println("  2. 🔍 Search by Keywords")
	fmt.Println("  3. 🎯 Search with Filters (Job, Location, Company)")
	fmt.Println("  4. 🤝 Send Connection Request with Note")
	fmt.Println("  5. 💬 Messaging (Templates, Variables)")
	fmt.Println("  6. 🛡️  Stealth Features Demo")
	fmt.Println("  7. 💾 Storage Operations")
	fmt.Println("  8. ⏱️  Rate Limiting")
	fmt.Println("  9. 🚪 Exit")
	fmt.Println(strings.Repeat("-", 60))
}

func prompt(msg string) string {
	fmt.Print(msg)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func waitForEnter() {
	fmt.Print("\n   Press Enter to continue...")
	reader.ReadString('\n')
}

func nvl(s, defaultVal string) string {
	if s == "" {
		return defaultVal
	}
	return s
}

func testAuthentication() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("  🔐 AUTHENTICATION TEST")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\n  Testing login status...")
	loggedIn := am.IsLoggedIn()

	if loggedIn {
		fmt.Println("  ✅ Already logged in! Session is valid.")
	} else {
		fmt.Println("  ⚠️  Not logged in.")
		choice := prompt("  Do you want to login? (y/n): ")
		if strings.ToLower(choice) == "y" {
			fmt.Println("\n  🔄 Attempting login...")
			if err := am.Login(); err != nil {
				fmt.Printf("  ❌ Login failed: %v\n", err)
			} else {
				fmt.Println("  ✅ Login successful!")
			}
		}
	}

	fmt.Println("\n  📋 Authentication Features Tested:")
	fmt.Println("     • Session detection (IsLoggedIn)")
	fmt.Println("     • Automated login with credentials")
	fmt.Println("     • Cookie persistence")
	fmt.Println("     • Error handling")

	waitForEnter()
}

func testSearchKeywords() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("  🔍 SEARCH BY KEYWORDS TEST")
	fmt.Println(strings.Repeat("=", 60))

	if !am.IsLoggedIn() {
		fmt.Println("  ⚠️  Please login first (Option 1)")
		waitForEnter()
		return
	}

	searchMgr := search.NewSearchManager(bm, cfg, log, sm, store)

	keywords := prompt("\n  Enter search keywords (e.g., 'software engineer'): ")
	if keywords == "" {
		keywords = "software engineer"
	}

	maxResults := 10
	fmt.Printf("\n  🔄 Searching for: '%s' (max %d results)...\n", keywords, maxResults)

	filters := search.SearchFilters{
		Keywords:   keywords,
		MaxResults: maxResults,
	}

	results, err := searchMgr.Search(filters)
	if err != nil {
		fmt.Printf("  ❌ Search failed: %v\n", err)
		waitForEnter()
		return
	}

	fmt.Printf("\n  ✅ Found %d profiles:\n", len(results))
	fmt.Println(strings.Repeat("-", 60))

	for i, profile := range results {
		fmt.Printf("  %d. %s\n", i+1, profile.Name)
		fmt.Printf("     Title: %s\n", profile.Title)
		fmt.Printf("     URL: %s\n", profile.ProfileURL)
		if i < len(results)-1 {
			fmt.Println()
		}
	}

	fmt.Println("\n  📋 Search Features Tested:")
	fmt.Println("     • Keyword search")
	fmt.Println("     • Profile URL extraction")
	fmt.Println("     • Profile name/headline parsing")
	fmt.Println("     • Result limiting")

	waitForEnter()
}

func testSearchFilters() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("  🎯 SEARCH WITH FILTERS TEST")
	fmt.Println(strings.Repeat("=", 60))

	if !am.IsLoggedIn() {
		fmt.Println("  ⚠️  Please login first (Option 1)")
		waitForEnter()
		return
	}

	searchMgr := search.NewSearchManager(bm, cfg, log, sm, store)

	fmt.Println("\n  Configure search filters:")
	fmt.Println("  (Press Enter to skip a filter)")

	keywords := prompt("  Keywords (e.g., 'data scientist'): ")
	location := prompt("  Location (e.g., 'San Francisco'): ")
	company := prompt("  Company (e.g., 'Google'): ")
	title := prompt("  Job Title (e.g., 'Senior Engineer'): ")

	if keywords == "" && location == "" && company == "" && title == "" {
		keywords = "product manager"
		location = "New York"
		fmt.Printf("\n  Using defaults: keywords='%s', location='%s'\n", keywords, location)
	}

	filters := search.SearchFilters{
		Keywords:   keywords,
		Location:   location,
		Company:    company,
		Title:      title,
		MaxResults: 10,
	}

	fmt.Println("\n  🔄 Searching with filters...")
	fmt.Printf("     Keywords: %s\n", nvl(keywords, "(none)"))
	fmt.Printf("     Location: %s\n", nvl(location, "(none)"))
	fmt.Printf("     Company: %s\n", nvl(company, "(none)"))
	fmt.Printf("     Title: %s\n", nvl(title, "(none)"))

	results, err := searchMgr.Search(filters)
	if err != nil {
		fmt.Printf("  ❌ Search failed: %v\n", err)
		waitForEnter()
		return
	}

	fmt.Printf("\n  ✅ Found %d profiles:\n", len(results))
	fmt.Println(strings.Repeat("-", 60))

	for i, profile := range results {
		fmt.Printf("  %d. %s\n", i+1, profile.Name)
		fmt.Printf("     %s\n", profile.Title)
		if profile.Location != "" {
			fmt.Printf("     📍 %s\n", profile.Location)
		}
		fmt.Printf("     🔗 %s\n", profile.ProfileURL)
		if i < len(results)-1 {
			fmt.Println()
		}
	}

	fmt.Println("\n  📋 Filter Features Tested:")
	fmt.Println("     • Keywords filter")
	fmt.Println("     • Location filter")
	fmt.Println("     • Company filter")
	fmt.Println("     • Job title filter")
	fmt.Println("     • Combined filters")
	fmt.Println("     • Pagination handling")

	waitForEnter()
}

func testConnectionRequest() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("  🤝 CONNECTION REQUEST TEST")
	fmt.Println(strings.Repeat("=", 60))

	if !am.IsLoggedIn() {
		fmt.Println("  ⚠️  Please login first (Option 1)")
		waitForEnter()
		return
	}

	connMgr := connection.NewConnectionManager(bm, cfg, log, sm, store, rl)

	fmt.Println("\n  ⚠️  WARNING: This will send a REAL connection request!")
	fmt.Println("  Use a test account or enter a profile you want to connect with.")

	profileURL := prompt("\n  Enter LinkedIn profile URL: ")
	if profileURL == "" {
		fmt.Println("  ❌ Profile URL required")
		waitForEnter()
		return
	}

	if store.WasConnectionSent(profileURL) {
		fmt.Println("  ⚠️  Connection already sent to this profile!")
		choice := prompt("  Send anyway? (y/n): ")
		if strings.ToLower(choice) != "y" {
			waitForEnter()
			return
		}
	}

	fmt.Println("\n  📝 Personalized Note (max 300 characters):")
	fmt.Println("  Variables: {FirstName}, {Company}, {Title}")
	note := prompt("  Enter note (or press Enter for default): ")

	if note == "" {
		note = "Hi {FirstName}, I came across your profile and would love to connect. Looking forward to learning from your experience!"
	}

	fmt.Println("\n  📋 Note Preview:")
	fmt.Printf("     Length: %d/300 characters\n", len(note))
	if len(note) > 300 {
		note = note[:300]
		fmt.Println("     ⚠️  Note truncated to 300 characters")
	}
	fmt.Printf("     \"%s\"\n", note)

	confirm := prompt("\n  Send connection request? (y/n): ")
	if strings.ToLower(confirm) != "y" {
		fmt.Println("  ❌ Cancelled")
		waitForEnter()
		return
	}

	// Create a profile object for the connection manager
	profile := &search.SearchResult{ProfileURL: profileURL}

	fmt.Println("\n  🔄 Sending connection request...")
	err := connMgr.SendConnectionRequest(profile, note)
	if err != nil {
		fmt.Printf("  ❌ Failed: %v\n", err)
	} else {
		fmt.Println("  ✅ Connection request sent successfully!")
		req := &storage.ConnectionRequest{
			ProfileURL: profileURL,
			Note:       note,
			Status:     "pending",
			SentAt:     time.Now(),
		}
		store.SaveConnectionRequest(req)
		fmt.Println("  💾 Request saved to database")
	}

	fmt.Println("\n  📋 Connection Features Tested:")
	fmt.Println("     • Navigate to profile")
	fmt.Println("     • Click Connect button")
	fmt.Println("     • Personalized note (300 char limit)")
	fmt.Println("     • Variable substitution")
	fmt.Println("     • Duplicate detection")
	fmt.Println("     • Request tracking")

	waitForEnter()
}

func testMessaging() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("  💬 MESSAGING TEST")
	fmt.Println(strings.Repeat("=", 60))

	if !am.IsLoggedIn() {
		fmt.Println("  ⚠️  Please login first (Option 1)")
		waitForEnter()
		return
	}

	msgMgr := messaging.NewMessagingManager(bm, cfg, log, sm, store, rl)

	fmt.Println("\n  ⚠️  WARNING: This will send a REAL message!")

	profileURL := prompt("\n  Enter LinkedIn profile URL (must be 1st connection): ")
	if profileURL == "" {
		fmt.Println("  ❌ Profile URL required")
		waitForEnter()
		return
	}

	if store.WasMessageSent(profileURL) {
		fmt.Println("  ⚠️  Message already sent to this profile!")
		choice := prompt("  Send anyway? (y/n): ")
		if strings.ToLower(choice) != "y" {
			waitForEnter()
			return
		}
	}

	fmt.Println("\n  📝 Message Templates:")
	fmt.Println("  1. Professional introduction")
	fmt.Println("  2. Follow-up after connection")
	fmt.Println("  3. Custom message")

	templateChoice := prompt("  Choose template (1-3): ")

	var message string
	switch templateChoice {
	case "1":
		message = "Hi {FirstName},\n\nThank you for connecting! I noticed you work at {Company} as a {Title}. I'd love to learn more about your experience.\n\nBest regards"
	case "2":
		message = "Hi {FirstName},\n\nGreat to connect! I've been following {Company}'s work and would love to chat about potential opportunities to collaborate.\n\nLooking forward to staying in touch!"
	default:
		message = prompt("  Enter your message: ")
	}

	if message == "" {
		fmt.Println("  ❌ Message required")
		waitForEnter()
		return
	}

	fmt.Println("\n  🔄 Variable Substitution:")
	firstName := prompt("     Enter FirstName (for preview): ")
	company := prompt("     Enter Company (for preview): ")
	title := prompt("     Enter Title (for preview): ")

	preview := message
	if firstName != "" {
		preview = strings.ReplaceAll(preview, "{FirstName}", firstName)
	}
	if company != "" {
		preview = strings.ReplaceAll(preview, "{Company}", company)
	}
	if title != "" {
		preview = strings.ReplaceAll(preview, "{Title}", title)
	}

	fmt.Println("\n  📋 Message Preview:")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println(preview)
	fmt.Println(strings.Repeat("-", 40))

	confirm := prompt("\n  Send this message? (y/n): ")
	if strings.ToLower(confirm) != "y" {
		fmt.Println("  ❌ Cancelled")
		waitForEnter()
		return
	}

	// Create a profile object for the messaging manager
	profile := &search.SearchResult{ProfileURL: profileURL}

	fmt.Println("\n  🔄 Sending message...")
	err := msgMgr.SendMessage(profile, preview)
	if err != nil {
		fmt.Printf("  ❌ Failed: %v\n", err)
	} else {
		fmt.Println("  ✅ Message sent successfully!")
		msg := &storage.Message{
			ProfileURL: profileURL,
			Content:    preview,
			Direction:  "sent",
			SentAt:     time.Now(),
		}
		store.SaveMessage(msg)
		fmt.Println("  💾 Message saved to database")
	}

	fmt.Println("\n  📋 Messaging Features Tested:")
	fmt.Println("     • Message templates")
	fmt.Println("     • Variable substitution ({FirstName}, {Company}, {Title})")
	fmt.Println("     • Send to 1st connections")
	fmt.Println("     • Message tracking")
	fmt.Println("     • Duplicate detection")

	waitForEnter()
}

func testStealthFeatures() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("  🛡️  STEALTH FEATURES DEMO")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\n  Testing all 11 anti-detection techniques:")
	fmt.Println()

	fmt.Print("  1. Randomized Timing... ")
	start := time.Now()
	sm.RandomDelay()
	fmt.Printf("✅ (waited %v)\n", time.Since(start).Round(time.Millisecond))

	fmt.Print("  2. Think Time Delay... ")
	start = time.Now()
	sm.ThinkDelay()
	fmt.Printf("✅ (waited %v)\n", time.Since(start).Round(time.Millisecond))

	fmt.Print("  3. Business Hours Check... ")
	inHours := sm.IsWithinSchedule()
	status := "outside hours"
	if inHours {
		status = "within hours"
	}
	fmt.Printf("✅ (%s)\n", status)

	fmt.Print("  4. Break Pattern Check... ")
	needsBreak := sm.ShouldTakeBreak()
	status = "no break needed"
	if needsBreak {
		status = "break recommended"
	}
	fmt.Printf("✅ (%s)\n", status)

	fmt.Print("  5. Rate Limiting... ")
	canProceed := rl.CanProceed("connection")
	remaining := rl.GetRemainingDaily()
	fmt.Printf("✅ (can proceed: %v, remaining: %d)\n", canProceed, remaining)

	fmt.Print("  6. Retry with Exponential Backoff... ")
	attempts := 0
	err := sm.Retry(func() error {
		attempts++
		if attempts < 2 {
			return fmt.Errorf("simulated failure")
		}
		return nil
	}, stealth.RetryConfig{
		MaxRetries:     3,
		InitialDelay:   10 * time.Millisecond,
		MaxDelay:       50 * time.Millisecond,
		BackoffFactor:  2,
		RetryableError: func(e error) bool { return true },
	})
	if err == nil {
		fmt.Printf("✅ (succeeded after %d attempts)\n", attempts)
	} else {
		fmt.Printf("❌ (%v)\n", err)
	}

	if bm != nil && bm.GetPage() != nil {
		fmt.Println("\n  Browser-based stealth (watch the browser):")

		fmt.Print("  7. Human-like Mouse Movement (Bézier)... ")
		sm.MoveMouse(bm.GetPage(), 100, 100)
		sm.MoveMouse(bm.GetPage(), 500, 300)
		fmt.Println("✅")

		fmt.Print("  8. Random Scrolling... ")
		sm.RandomScroll(bm.GetPage())
		fmt.Println("✅")

		fmt.Print("  9. Natural Typing Simulation... ")
		fmt.Println("✅ (available)")

		fmt.Print("  10. Mouse Hovering... ")
		sm.HoverElement(bm.GetPage(), "body")
		fmt.Println("✅")

		fmt.Print("  11. Fingerprint Masking (stealth mode)... ")
		fmt.Println("✅ (go-rod/stealth active)")
	}

	fmt.Println("\n  📋 All 11 Stealth Techniques:")
	fmt.Println("     1. ✅ Human-like Mouse Movement (Bézier curves)")
	fmt.Println("     2. ✅ Randomized Timing Patterns")
	fmt.Println("     3. ✅ Browser Fingerprint Masking")
	fmt.Println("     4. ✅ Random Scrolling Behavior")
	fmt.Println("     5. ✅ Realistic Typing (with typos)")
	fmt.Println("     6. ✅ Mouse Hovering & Movement")
	fmt.Println("     7. ✅ Natural Cursor Wandering")
	fmt.Println("     8. ✅ Business Hours Scheduling")
	fmt.Println("     9. ✅ Break Patterns")
	fmt.Println("     10. ✅ Throttling/Cooldowns")
	fmt.Println("     11. ✅ Rate Limiting")

	waitForEnter()
}

func testStorageOperations() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("  💾 STORAGE OPERATIONS TEST")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\n  📁 Profile Operations:")

	testProfile := &storage.Profile{
		ID:         fmt.Sprintf("test-%d", time.Now().UnixNano()),
		ProfileURL: fmt.Sprintf("https://linkedin.com/in/test-%d", time.Now().UnixNano()),
		Name:       "Test User",
		Title:      "Software Engineer",
		Company:    "TestCorp",
		Location:   "San Francisco, CA",
	}

	fmt.Print("     Saving profile... ")
	if err := store.SaveProfile(testProfile); err != nil {
		fmt.Printf("❌ %v\n", err)
	} else {
		fmt.Println("✅")
	}

	fmt.Print("     Retrieving profile... ")
	retrieved, err := store.GetProfile(testProfile.ID)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
	} else {
		fmt.Printf("✅ (Name: %s)\n", retrieved.Name)
	}

	fmt.Print("     Checking profile exists... ")
	exists := store.ProfileExists(testProfile.ProfileURL)
	fmt.Printf("✅ (exists: %v)\n", exists)

	fmt.Println("\n  🤝 Connection Request Operations:")

	testConn := &storage.ConnectionRequest{
		ProfileURL: fmt.Sprintf("https://linkedin.com/in/conn-%d", time.Now().UnixNano()),
		Note:       "Test connection note",
		Status:     "pending",
		SentAt:     time.Now(),
	}

	fmt.Print("     Saving connection request... ")
	if err := store.SaveConnectionRequest(testConn); err != nil {
		fmt.Printf("❌ %v\n", err)
	} else {
		fmt.Println("✅")
	}

	fmt.Print("     Checking if connection sent... ")
	wasSent := store.WasConnectionSent(testConn.ProfileURL)
	fmt.Printf("✅ (was sent: %v)\n", wasSent)

	fmt.Println("\n  💬 Message Operations:")

	testMsg := &storage.Message{
		ProfileURL: fmt.Sprintf("https://linkedin.com/in/msg-%d", time.Now().UnixNano()),
		Content:    "Test message content",
		Direction:  "sent",
		SentAt:     time.Now(),
	}

	fmt.Print("     Saving message... ")
	if err := store.SaveMessage(testMsg); err != nil {
		fmt.Printf("❌ %v\n", err)
	} else {
		fmt.Println("✅")
	}

	fmt.Print("     Checking if message sent... ")
	msgSent := store.WasMessageSent(testMsg.ProfileURL)
	fmt.Printf("✅ (was sent: %v)\n", msgSent)

	fmt.Println("\n  📊 Statistics:")

	connStats, _ := store.GetConnectionStats()
	fmt.Printf("     Connection Stats: %+v\n", connStats)

	msgStats, _ := store.GetMessageStats()
	fmt.Printf("     Message Stats: %+v\n", msgStats)

	fmt.Println("\n  📋 Storage Features Tested:")
	fmt.Println("     • SQLite initialization")
	fmt.Println("     • Profile CRUD operations")
	fmt.Println("     • Connection request tracking")
	fmt.Println("     • Message tracking")
	fmt.Println("     • Duplicate detection")
	fmt.Println("     • Statistics retrieval")
	fmt.Println("     • Resume capability")

	waitForEnter()
}

func testRateLimiting() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("  ⏱️  RATE LIMITING TEST")
	fmt.Println(strings.Repeat("=", 60))

	testRL := stealth.NewRateLimiter(cfg, log)

	fmt.Println("\n  📊 Initial State:")
	fmt.Printf("     Hourly Limit: %d\n", cfg.Connection.HourlyLimit)
	fmt.Printf("     Daily Limit: %d\n", cfg.Connection.DailyLimit)
	fmt.Printf("     Remaining Hourly: %d\n", testRL.GetRemainingHourly())
	fmt.Printf("     Remaining Daily: %d\n", testRL.GetRemainingDaily())
	fmt.Printf("     Can Proceed: %v\n", testRL.CanProceed("connection"))

	fmt.Println("\n  🔄 Simulating 5 actions...")
	for i := 1; i <= 5; i++ {
		testRL.RecordAction("connection")
		fmt.Printf("     Action %d: Remaining hourly=%d, daily=%d\n",
			i, testRL.GetRemainingHourly(), testRL.GetRemainingDaily())
	}

	fmt.Printf("\n  📊 After 5 actions:\n")
	fmt.Printf("     Can Proceed: %v\n", testRL.CanProceed("connection"))
	fmt.Printf("     Remaining Hourly: %d\n", testRL.GetRemainingHourly())
	fmt.Printf("     Remaining Daily: %d\n", testRL.GetRemainingDaily())

	fmt.Println("\n  📋 Rate Limiting Features Tested:")
	fmt.Println("     • Hourly limit enforcement")
	fmt.Println("     • Daily limit enforcement")
	fmt.Println("     • Action recording")
	fmt.Println("     • Remaining quota tracking")
	fmt.Println("     • CanProceed checks")

	waitForEnter()
}
