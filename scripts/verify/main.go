package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

type CheckResult struct {
	Category string
	Item     string
	Passed   bool
	Details  string
}

var results []CheckResult
var passCount, failCount int

func main() {
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("  LinkedIn Automation Assignment - Checklist Verification")
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println()

	checkPhase1()
	checkPhase2()
	checkPhase3()
	checkPhase4()
	checkPhase5()
	checkPhase6()
	printSummary()
}

func check(category, item string, passed bool, details string) {
	results = append(results, CheckResult{category, item, passed, details})
	status := "✅ PASS"
	if !passed {
		status = "❌ FAIL"
		failCount++
	} else {
		passCount++
	}
	fmt.Printf("%s | %s\n", status, item)
	if details != "" && !passed {
		fmt.Printf("         └── %s\n", details)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fileContains(path string, patterns ...string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	for _, p := range patterns {
		if strings.Contains(string(content), p) {
			return true
		}
	}
	return false
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func countMatches(path string, pattern string) int {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	re := regexp.MustCompile(pattern)
	return len(re.FindAllString(string(content), -1))
}

func checkPhase1() {
	fmt.Println("\n📦 PHASE 1: Project Initialization & Architecture")
	fmt.Println(strings.Repeat("-", 60))

	check("Phase 1", "Go module initialized (go.mod exists)",
		fileExists("go.mod"), "Missing go.mod file")

	check("Phase 1", "Rod library imported",
		fileContains("go.mod", "go-rod/rod"), "Rod library not found in go.mod")

	packages := map[string]string{
		"internal/auth":       "authentication",
		"internal/search":     "search",
		"internal/messaging":  "messaging",
		"internal/stealth":    "stealth",
		"internal/config":     "config",
		"internal/connection": "connection",
		"internal/browser":    "browser",
		"internal/storage":    "storage",
		"internal/logger":     "logger",
	}

	for path, name := range packages {
		check("Phase 1", fmt.Sprintf("Package exists: %s", name),
			dirExists(path), fmt.Sprintf("Missing %s directory", path))
	}
}

func checkPhase2() {
	fmt.Println("\n⚙️ PHASE 2: Configuration & State Management")
	fmt.Println(strings.Repeat("-", 60))

	check("Phase 2", "YAML config file support",
		fileExists("config.yaml") || fileExists("config.yaml.example"),
		"Missing config.yaml file")

	check("Phase 2", "YAML parsing in config package",
		fileContains("internal/config/config.go", "yaml"),
		"YAML parsing not implemented")

	check("Phase 2", ".env file loading (godotenv)",
		fileContains("go.mod", "godotenv") || fileContains("cmd/linkedin/main.go", "godotenv"),
		"godotenv not found")

	check("Phase 2", "Environment variable overrides",
		fileContains("internal/config/config.go", "os.Getenv"),
		"Environment variable reading not found")

	check("Phase 2", "Config validation",
		fileContains("internal/config/config.go", "error"),
		"Config validation not found")

	check("Phase 2", "SQLite storage",
		fileContains("go.mod", "sqlite") && fileContains("internal/storage/storage.go", "sql.Open"),
		"SQLite storage not implemented")

	check("Phase 2", "Track sent connection requests",
		fileContains("internal/storage/storage.go", "connection_request") ||
			fileContains("internal/storage/storage.go", "WasConnectionSent"),
		"Connection request tracking not found")

	check("Phase 2", "Track messages",
		fileContains("internal/storage/storage.go", "message") ||
			fileContains("internal/storage/storage.go", "WasMessageSent"),
		"Message tracking not found")

	check("Phase 2", "Resume capability (cookies/state)",
		fileContains("internal/browser/browser.go", "LoadCookies") ||
			fileContains("internal/browser/browser.go", "SaveCookies"),
		"Session persistence not found")
}

func checkPhase3() {
	fmt.Println("\n🔧 PHASE 3: Core Functional Requirements")
	fmt.Println(strings.Repeat("-", 60))

	fmt.Println("\n  A. Authentication System")
	check("Phase 3A", "Automated login with credentials",
		fileContains("internal/auth/auth.go", "Login"),
		"Login automation not found")

	check("Phase 3A", "Login error detection",
		fileContains("internal/auth/auth.go", "error"),
		"Login error handling not found")

	check("Phase 3A", "2FA/CAPTCHA detection",
		fileContains("internal/auth/auth.go", "captcha", "CAPTCHA", "checkpoint", "verification"),
		"Security checkpoint detection not found")

	check("Phase 3A", "Session/cookie persistence",
		fileContains("internal/browser/browser.go", "cookie", "Cookie"),
		"Cookie handling not found")

	fmt.Println("\n  B. Search & Targeting")
	check("Phase 3B", "Search filters (Keywords, Location)",
		fileContains("internal/search/search.go", "Keywords"),
		"Search filters not found")

	check("Phase 3B", "Profile URL scraping",
		fileContains("internal/search/search.go", "ProfileURL", "/in/"),
		"Profile URL extraction not found")

	check("Phase 3B", "Pagination handling",
		fileContains("internal/search/search.go", "NextPage", "goToNextPage", "page="),
		"Pagination not implemented")

	check("Phase 3B", "Deduplication",
		fileContains("internal/storage/storage.go", "ProfileExists") ||
			fileContains("internal/search/search.go", "seenURLs"),
		"Deduplication not found")

	fmt.Println("\n  C. Connection Requests")
	check("Phase 3C", "Profile navigation",
		fileContains("internal/connection/connection.go", "Navigate"),
		"Profile navigation not found")

	check("Phase 3C", "Connect button clicking",
		fileContains("internal/connection/connection.go", "Connect"),
		"Connect button handling not found")

	check("Phase 3C", "Personalized notes",
		fileContains("internal/connection/connection.go", "note", "Note"),
		"Connection note support not found")

	check("Phase 3C", "Character limit enforcement",
		fileContains("internal/connection/connection.go", "300") ||
			fileContains("internal/connection/connection.go", "MaxNoteLength"),
		"Character limit not enforced")

	check("Phase 3C", "Daily limits",
		fileContains("internal/connection/connection.go", "DailyLimit") ||
			fileContains("internal/stealth/stealth.go", "DailyLimit"),
		"Daily limits not implemented")

	fmt.Println("\n  D. Messaging System")
	check("Phase 3D", "Follow-up messaging",
		fileContains("internal/messaging/messaging.go", "SendMessage"),
		"Follow-up messaging not found")

	check("Phase 3D", "Message templates with variables",
		fileContains("internal/messaging/messaging.go", "template", "Template", "FirstName"),
		"Message templating not found")

	check("Phase 3D", "Message logging/tracking",
		fileContains("internal/messaging/messaging.go", "storage"),
		"Message tracking not found")
}

func checkPhase4() {
	fmt.Println("\n🛡️ PHASE 4: Anti-Bot Detection Strategy (8 techniques required)")
	fmt.Println(strings.Repeat("-", 60))

	stealthFile := "internal/stealth/stealth.go"
	browserFile := "internal/browser/browser.go"
	techniqueCount := 0

	fmt.Println("\n  MANDATORY (All 3 required):")

	// 1. Human-like Mouse Movement (Bézier curves)
	hasBezier := fileContains(stealthFile, "Bezier", "bezier", "controlPoint", "curve")
	if hasBezier {
		techniqueCount++
	}
	check("Phase 4", "1. Human-like Mouse Movement (Bézier curves)",
		hasBezier, "Missing Bézier curves implementation")

	// 2. Randomized Timing
	hasRandomDelay := fileContains(stealthFile, "RandomDelay", "rand.Intn", "ThinkTime")
	if hasRandomDelay {
		techniqueCount++
	}
	check("Phase 4", "2. Randomized Timing (delays, think time)",
		hasRandomDelay, "Missing random delays")

	// 3. Browser Fingerprint Masking
	hasStealth := fileContains("go.mod", "go-rod/stealth")
	hasFingerprint := fileContains(browserFile, "stealth") ||
		fileContains(stealthFile, "UserAgent", "Viewport", "webdriver")
	if hasStealth || hasFingerprint {
		techniqueCount++
	}
	check("Phase 4", "3. Browser Fingerprint Masking (stealth library or manual)",
		hasStealth || hasFingerprint, "Missing fingerprint masking")

	fmt.Println("\n  ADDITIONAL (Need at least 5):")
	additionalCount := 0

	// Random Scrolling
	if fileContains(stealthFile, "Scroll", "scroll") {
		additionalCount++
		check("Phase 4", "4. Random Scrolling", true, "")
	} else {
		check("Phase 4", "4. Random Scrolling", false, "Not implemented")
	}

	// Realistic Typing
	if fileContains(stealthFile, "HumanType", "TypeWithTypos", "typo") {
		additionalCount++
		check("Phase 4", "5. Realistic Typing (typos)", true, "")
	} else {
		check("Phase 4", "5. Realistic Typing (typos)", false, "Not implemented")
	}

	// Mouse Hovering
	if fileContains(stealthFile, "Hover", "hover") {
		additionalCount++
		check("Phase 4", "6. Mouse Hovering", true, "")
	} else {
		check("Phase 4", "6. Mouse Hovering", false, "Not implemented")
	}

	// Cursor Wandering
	if fileContains(stealthFile, "Wander", "wander", "IdleMovement") {
		additionalCount++
		check("Phase 4", "7. Natural Cursor Wandering", true, "")
	} else {
		check("Phase 4", "7. Natural Cursor Wandering", false, "Not implemented")
	}

	// Business Hours
	if fileContains(stealthFile, "BusinessHours", "Schedule", "IsWithinSchedule") {
		additionalCount++
		check("Phase 4", "8. Business Hours restriction", true, "")
	} else {
		check("Phase 4", "8. Business Hours restriction", false, "Not implemented")
	}

	// Break Patterns
	if fileContains(stealthFile, "Break", "CoffeeBreak", "TakeBreak") {
		additionalCount++
		check("Phase 4", "9. Break Patterns", true, "")
	} else {
		check("Phase 4", "9. Break Patterns", false, "Not implemented")
	}

	// Throttling/Cooldowns
	if fileContains(stealthFile, "Cooldown", "cooldown", "ActionDelay") {
		additionalCount++
		check("Phase 4", "10. Throttling/Cooldowns", true, "")
	} else {
		check("Phase 4", "10. Throttling/Cooldowns", false, "Not implemented")
	}

	// Rate Limiting
	if fileContains(stealthFile, "RateLimiter", "DailyLimit", "HourlyLimit") {
		additionalCount++
		check("Phase 4", "11. Rate Limiting", true, "")
	} else {
		check("Phase 4", "11. Rate Limiting", false, "Not implemented")
	}

	techniqueCount += additionalCount
	fmt.Printf("\n  📊 Total Stealth Techniques: %d/8 required\n", techniqueCount)
	if techniqueCount >= 8 {
		fmt.Println("  ✅ STEALTH REQUIREMENT MET!")
	} else {
		fmt.Printf("  ❌ Need %d more techniques\n", 8-techniqueCount)
	}
}

func checkPhase5() {
	fmt.Println("\n📝 PHASE 5: Code Quality Standards")
	fmt.Println(strings.Repeat("-", 60))

	fmt.Println("\n  Error Handling:")
	check("Phase 5", "Comprehensive error detection",
		countMatches("internal/auth/auth.go", `if err != nil`) >= 3,
		"Insufficient error handling")

	check("Phase 5", "Graceful degradation (recover/defer)",
		fileContains("cmd/linkedin/main.go", "recover") ||
			fileContains("internal/browser/browser.go", "defer"),
		"No graceful degradation found")

	check("Phase 5", "Retry logic",
		fileContains("internal/stealth/stealth.go", "Retry", "retry") ||
			fileContains("internal/connection/connection.go", "retry"),
		"Retry logic not found")

	fmt.Println("\n  Logging:")
	check("Phase 5", "Structured logging library (logrus/zap)",
		fileContains("go.mod", "logrus", "zap", "zerolog"),
		"No structured logging library found")

	check("Phase 5", "Log levels (Debug, Info, Warn, Error)",
		fileContains("internal/logger/logger.go", "Debug") &&
			fileContains("internal/logger/logger.go", "Info") &&
			fileContains("internal/logger/logger.go", "Error"),
		"Missing log levels")

	check("Phase 5", "Contextual logging (WithField/fields)",
		fileContains("internal/logger/logger.go", "WithField", "fields"),
		"Contextual logging not found")

	fmt.Println("\n  Documentation:")
	check("Phase 5", "README.md exists",
		fileExists("README.md"), "Missing README.md")

	readmeContent, _ := os.ReadFile("README.md")
	readme := string(readmeContent)

	check("Phase 5", "README has setup instructions",
		strings.Contains(readme, "Setup") || strings.Contains(readme, "install") ||
			strings.Contains(readme, "Installation"),
		"README missing setup instructions")

	check("Phase 5", "README documents features",
		strings.Contains(readme, "Feature") || strings.Contains(readme, "feature") ||
			strings.Contains(readme, "Anti-Detection"),
		"README missing feature documentation")

	stealthComments := countMatches("internal/stealth/stealth.go", `//`)
	check("Phase 5", "Inline comments in stealth code (>15)",
		stealthComments > 15,
		fmt.Sprintf("Only %d comments found", stealthComments))
}

func checkPhase6() {
	fmt.Println("\n📦 PHASE 6: Deliverables")
	fmt.Println(strings.Repeat("-", 60))

	check("Phase 6", "Logical directory structure (cmd/, internal/)",
		dirExists("cmd") && dirExists("internal"),
		"Missing expected directories")

	check("Phase 6", "go.mod configured correctly",
		fileContains("go.mod", "module"),
		"go.mod not properly configured")

	check("Phase 6", ".env.example exists",
		fileExists(".env.example"), "Missing .env.example template")

	if fileExists(".env.example") {
		check("Phase 6", ".env.example documents variables",
			fileContains(".env.example", "EMAIL", "LINKEDIN", "PASSWORD"),
			".env.example missing variable documentation")
	}

	check("Phase 6", ".gitignore includes .env",
		fileExists(".gitignore") && fileContains(".gitignore", ".env"),
		".env not in .gitignore - SECURITY RISK!")

	check("Phase 6", "Demo video link in README",
		fileContains("README.md", "video", "Video", "demo", "Demo", "youtube", "loom"),
		"No demo video link found in README")
}

func printSummary() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📊 SUMMARY")
	fmt.Println(strings.Repeat("=", 80))

	total := passCount + failCount
	percentage := float64(passCount) / float64(total) * 100

	fmt.Printf("\n  Total Checks: %d\n", total)
	fmt.Printf("  ✅ Passed: %d\n", passCount)
	fmt.Printf("  ❌ Failed: %d\n", failCount)
	fmt.Printf("  📈 Score: %.1f%%\n", percentage)

	fmt.Println("\n" + strings.Repeat("-", 80))

	if percentage >= 90 {
		fmt.Println("  🎉 EXCELLENT! Project meets most requirements.")
	} else if percentage >= 70 {
		fmt.Println("  👍 GOOD! Project is mostly complete but needs some work.")
	} else if percentage >= 50 {
		fmt.Println("  ⚠️  NEEDS WORK! Several requirements are missing.")
	} else {
		fmt.Println("  ❌ INCOMPLETE! Many requirements are missing.")
	}

	if failCount > 0 {
		fmt.Println("\n  ❌ Failed Items:")
		for _, r := range results {
			if !r.Passed {
				fmt.Printf("    • %s\n", r.Item)
				if r.Details != "" {
					fmt.Printf("      └── %s\n", r.Details)
				}
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
}
