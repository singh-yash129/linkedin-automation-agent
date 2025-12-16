// Package connection handles sending connection requests on LinkedIn.
package connection

import (
	"fmt"
	"strings"
	"time"

	"github.com/singh-yash129/link/internal/browser"
	"github.com/singh-yash129/link/internal/config"
	"github.com/singh-yash129/link/internal/logger"
	"github.com/singh-yash129/link/internal/search"
	"github.com/singh-yash129/link/internal/stealth"
	"github.com/singh-yash129/link/internal/storage"
)

// ConnectionManager handles connection request operations.
type ConnectionManager struct {
	browser     *browser.BrowserManager
	config      *config.Config
	log         *logger.Logger
	stealth     *stealth.StealthManager
	storage     storage.Storage
	rateLimiter *stealth.RateLimiter
}

// NewConnectionManager creates a new connection manager.
func NewConnectionManager(bm *browser.BrowserManager, cfg *config.Config, log *logger.Logger, sm *stealth.StealthManager, store storage.Storage, rl *stealth.RateLimiter) *ConnectionManager {
	return &ConnectionManager{
		browser:     bm,
		config:      cfg,
		log:         log,
		stealth:     sm,
		storage:     store,
		rateLimiter: rl,
	}
}

// SendConnectionRequest sends a connection request to a profile.
func (cm *ConnectionManager) SendConnectionRequest(profile *search.SearchResult, note string) error {
	// Check rate limits
	if !cm.rateLimiter.CanProceed("connection") {
		return fmt.Errorf("rate limit reached")
	}

	// Wait for schedule
	cm.stealth.WaitForSchedule()

	// Take break if needed
	if cm.stealth.ShouldTakeBreak() {
		cm.stealth.TakeBreak()
	}

	cm.log.Info("Sending connection request", map[string]interface{}{
		"name": profile.Name,
		"url":  profile.ProfileURL,
	})

	// Navigate to profile
	if err := cm.browser.Navigate(profile.ProfileURL); err != nil {
		return fmt.Errorf("failed to navigate to profile: %w", err)
	}

	cm.stealth.PageDelay()

	// Random scroll to simulate reading profile
	cm.stealth.RandomScroll(cm.browser.GetPage())
	cm.stealth.ThinkDelay()

	// Find and click Connect button
	if err := cm.clickConnectButton(); err != nil {
		return err
	}

	cm.stealth.RandomDelay()

	// Handle connection modal
	if note != "" {
		if err := cm.addConnectionNote(note, profile); err != nil {
			cm.log.Warn("Failed to add note, sending without", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Click send
	if err := cm.clickSendButton(); err != nil {
		return err
	}

	// Record the action
	cm.rateLimiter.RecordAction("connection")

	// Save to storage
	cm.saveConnectionRequest(profile, note)

	cm.log.Info("Connection request sent", map[string]interface{}{
		"name": profile.Name,
	})

	return nil
}

// clickConnectButton finds and clicks the Connect button.
func (cm *ConnectionManager) clickConnectButton() error {
	// Wait for page to load
	time.Sleep(2 * time.Second)

	// First, try direct Connect button on profile (sometimes visible)
	connectSelectors := []string{
		"button[aria-label*='Connect']",
	}

	for _, selector := range connectSelectors {
		if cm.browser.HasElement(selector) {
			cm.stealth.HoverElement(cm.browser.GetPage(), selector)
			cm.stealth.ThinkDelay()
			if err := cm.browser.Click(selector); err == nil {
				return nil
			}
		}
	}

	// Try finding Connect button by text
	if cm.browser.HasElementWithText("button", "Connect") {
		cm.stealth.ThinkDelay()
		if err := cm.browser.ClickElementWithText("button", "Connect"); err == nil {
			return nil
		}
	}

	// Connect is usually in "More" dropdown on LinkedIn
	moreButtonSelectors := []string{
		"button[aria-label='More actions']",
		"button[id*='profile-overflow-action']",
	}

	for _, moreSelector := range moreButtonSelectors {
		if cm.browser.HasElement(moreSelector) {
			cm.stealth.HoverElement(cm.browser.GetPage(), moreSelector)
			cm.stealth.ThinkDelay()
			cm.browser.Click(moreSelector)
			time.Sleep(1 * time.Second) // Wait for dropdown to open

			// Now click "Invite to connect" in dropdown
			connectDropdownSelectors := []string{
				"div[aria-label*='to connect'][role='button']",
				"div[aria-label*='Invite'][aria-label*='connect'][role='button']",
			}

			for _, dropdownSelector := range connectDropdownSelectors {
				if cm.browser.HasElement(dropdownSelector) {
					cm.stealth.ThinkDelay()
					if err := cm.browser.Click(dropdownSelector); err == nil {
						return nil
					}
				}
			}

			// Fallback: find by text in dropdown
			if cm.browser.HasElementWithText("div.artdeco-dropdown__item", "Connect") {
				cm.stealth.ThinkDelay()
				if err := cm.browser.ClickElementWithText("div.artdeco-dropdown__item", "Connect"); err == nil {
					return nil
				}
			}
			break
		}
	}

	return fmt.Errorf("connect button not found")
}

// addConnectionNote adds a personalized note to the connection request.
func (cm *ConnectionManager) addConnectionNote(note string, profile *search.SearchResult) error {
	// Wait for modal
	time.Sleep(2 * time.Second)

	// Click "Add a note" button if present
	addNoteSelectors := []string{
		"button[aria-label='Add a note']",
	}

	for _, selector := range addNoteSelectors {
		if cm.browser.HasElement(selector) {
			if err := cm.browser.Click(selector); err == nil {
				break
			}
		}
	}

	// Fallback: find by text
	if cm.browser.HasElementWithText("button", "Add a note") {
		cm.browser.ClickElementWithText("button", "Add a note")
	}

	time.Sleep(1 * time.Second)

	// Personalize the note
	personalizedNote := cm.personalizeNote(note, profile)

	// Find textarea and type note
	noteSelectors := []string{
		"textarea#custom-message",
		"textarea.connect-button-send-invite__custom-message",
		"textarea[name='message']",
		".send-invite textarea",
		".artdeco-modal textarea",
		"div[role='dialog'] textarea",
	}

	for _, selector := range noteSelectors {
		if cm.browser.HasElement(selector) {
			// Use human-like typing
			return cm.stealth.HumanType(cm.browser.GetPage(), personalizedNote)
		}
	}

	return fmt.Errorf("note textarea not found")
}

// personalizeNote replaces placeholders in the note template.
func (cm *ConnectionManager) personalizeNote(template string, profile *search.SearchResult) string {
	note := template

	// Replace placeholders
	note = strings.ReplaceAll(note, "{name}", profile.Name)
	note = strings.ReplaceAll(note, "{firstName}", cm.getFirstName(profile.Name))
	note = strings.ReplaceAll(note, "{title}", profile.Title)
	note = strings.ReplaceAll(note, "{company}", profile.Company)
	note = strings.ReplaceAll(note, "{location}", profile.Location)

	// Ensure note doesn't exceed LinkedIn's limit (300 chars)
	if len(note) > 300 {
		note = note[:297] + "..."
	}

	return note
}

// getFirstName extracts the first name from a full name.
func (cm *ConnectionManager) getFirstName(fullName string) string {
	parts := strings.Fields(fullName)
	if len(parts) > 0 {
		return parts[0]
	}
	return fullName
}

// clickSendButton clicks the send/submit button.
func (cm *ConnectionManager) clickSendButton() error {
	// LinkedIn shows either "Send" (after adding note) or "Send without a note"
	sendSelectors := []string{
		"button[aria-label='Send invitation']",
		"button[aria-label='Send without a note']",
		"button[aria-label='Send now']",
		".artdeco-modal__actionbar button.artdeco-button--primary",
		".send-invite button.artdeco-button--primary",
		"div[role='dialog'] button.artdeco-button--primary",
		"button.artdeco-button--primary:has-text('Send')",
	}

	cm.stealth.ThinkDelay()

	for _, selector := range sendSelectors {
		if cm.browser.HasElement(selector) {
			return cm.browser.Click(selector)
		}
	}

	return fmt.Errorf("send button not found")
}

// saveConnectionRequest saves the connection request to storage.
func (cm *ConnectionManager) saveConnectionRequest(profile *search.SearchResult, note string) {
	req := &storage.ConnectionRequest{
		ProfileID:   cm.generateProfileID(profile.ProfileURL),
		ProfileURL:  profile.ProfileURL,
		ProfileName: profile.Name,
		Note:        note,
		Status:      "pending",
		SentAt:      time.Now(),
	}

	if err := cm.storage.SaveConnectionRequest(req); err != nil {
		cm.log.Warn("Failed to save connection request", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// generateProfileID generates a profile ID from URL.
func (cm *ConnectionManager) generateProfileID(profileURL string) string {
	parts := strings.Split(profileURL, "/in/")
	if len(parts) < 2 {
		return fmt.Sprintf("profile_%d", time.Now().UnixNano())
	}
	return strings.TrimSuffix(parts[1], "/")
}

// SendBulkConnections sends connection requests to multiple profiles.
func (cm *ConnectionManager) SendBulkConnections(profiles []*search.SearchResult, note string) *BulkResult {
	result := &BulkResult{
		Total:     len(profiles),
		Succeeded: 0,
		Failed:    0,
		Skipped:   0,
		StartTime: time.Now(),
	}

	cm.log.Info("Starting bulk connection requests", map[string]interface{}{
		"total": len(profiles),
	})

	for i, profile := range profiles {
		// Check if we should stop
		if !cm.rateLimiter.CanProceed("connection") {
			cm.log.Info("Rate limit reached, stopping bulk send", map[string]interface{}{
				"completed": i,
				"remaining": len(profiles) - i,
			})
			result.Skipped = len(profiles) - i
			break
		}

		// Check schedule
		cm.stealth.WaitForSchedule()

		// Take break if needed
		if cm.stealth.ShouldTakeBreak() {
			cm.stealth.TakeBreak()
		}

		// Skip if already sent
		if cm.storage.WasConnectionSent(profile.ProfileURL) {
			cm.log.Debug("Skipping - connection already sent", map[string]interface{}{
				"url": profile.ProfileURL,
			})
			result.Skipped++
			continue
		}

		// Send connection
		if err := cm.SendConnectionRequest(profile, note); err != nil {
			cm.log.Warn("Failed to send connection", map[string]interface{}{
				"name":  profile.Name,
				"error": err.Error(),
			})
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", profile.Name, err.Error()))
		} else {
			result.Succeeded++
		}

		// Random delay between connections
		cm.stealth.RandomDelay()
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	cm.log.Info("Bulk connection completed", map[string]interface{}{
		"succeeded": result.Succeeded,
		"failed":    result.Failed,
		"skipped":   result.Skipped,
		"duration":  result.Duration.String(),
	})

	return result
}

// BulkResult contains the results of a bulk operation.
type BulkResult struct {
	Total     int
	Succeeded int
	Failed    int
	Skipped   int
	Errors    []string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
}

// WithdrawConnection withdraws a pending connection request.
func (cm *ConnectionManager) WithdrawConnection(profileURL string) error {
	cm.log.Info("Withdrawing connection request", map[string]interface{}{
		"url": profileURL,
	})

	if err := cm.browser.Navigate(profileURL); err != nil {
		return err
	}

	cm.stealth.PageDelay()

	// Look for pending button
	pendingSelectors := []string{
		"button[aria-label*='Pending']",
		"button:has-text('Pending')",
	}

	for _, selector := range pendingSelectors {
		if cm.browser.HasElement(selector) {
			if err := cm.browser.Click(selector); err == nil {
				cm.stealth.RandomDelay()

				// Click withdraw in dropdown/modal
				withdrawSelectors := []string{
					"button[aria-label*='Withdraw']",
					"button:has-text('Withdraw')",
				}

				for _, ws := range withdrawSelectors {
					if cm.browser.HasElement(ws) {
						return cm.browser.Click(ws)
					}
				}
			}
		}
	}

	return fmt.Errorf("pending connection not found")
}

// GetConnectionStats returns connection statistics.
func (cm *ConnectionManager) GetConnectionStats() (map[string]int, error) {
	return cm.storage.GetConnectionStats()
}

// GetRemainingQuota returns the remaining connection quota.
func (cm *ConnectionManager) GetRemainingQuota() map[string]int {
	return map[string]int{
		"hourly": cm.rateLimiter.GetRemainingHourly(),
		"daily":  cm.rateLimiter.GetRemainingDaily(),
	}
}
