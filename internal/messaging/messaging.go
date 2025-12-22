// Package messaging handles LinkedIn messaging functionality.
package messaging

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/singh-yash129/linkedin-automation-agent/internal/browser"
	"github.com/singh-yash129/linkedin-automation-agent/internal/config"
	"github.com/singh-yash129/linkedin-automation-agent/internal/logger"
	"github.com/singh-yash129/linkedin-automation-agent/internal/search"
	"github.com/singh-yash129/linkedin-automation-agent/internal/stealth"
	"github.com/singh-yash129/linkedin-automation-agent/internal/storage"
)

// MessagingManager handles LinkedIn messaging operations.
type MessagingManager struct {
	browser     *browser.BrowserManager
	config      *config.Config
	log         *logger.Logger
	stealth     *stealth.StealthManager
	storage     storage.Storage
	rateLimiter *stealth.RateLimiter
}

// NewMessagingManager creates a new messaging manager.
func NewMessagingManager(bm *browser.BrowserManager, cfg *config.Config, log *logger.Logger, sm *stealth.StealthManager, store storage.Storage, rl *stealth.RateLimiter) *MessagingManager {
	return &MessagingManager{
		browser:     bm,
		config:      cfg,
		log:         log,
		stealth:     sm,
		storage:     store,
		rateLimiter: rl,
	}
}

// SendMessage sends a message to a connection.
func (mm *MessagingManager) SendMessage(profile *search.SearchResult, message string) error {
	// Check rate limits
	if !mm.rateLimiter.CanProceed("message") {
		return fmt.Errorf("message rate limit reached")
	}

	// Wait for schedule
	mm.stealth.WaitForSchedule()

	// Take break if needed
	if mm.stealth.ShouldTakeBreak() {
		mm.stealth.TakeBreak()
	}

	mm.log.Info("Sending message", map[string]interface{}{
		"name": profile.Name,
		"url":  profile.ProfileURL,
	})

	// Navigate to profile
	if err := mm.browser.Navigate(profile.ProfileURL); err != nil {
		return fmt.Errorf("failed to navigate to profile: %w", err)
	}

	// Wait for page to fully load
	fmt.Println("[DEBUG] Navigated to profile, waiting for page load...")
	time.Sleep(5 * time.Second)

	// Take a debug screenshot
	mm.browser.Screenshot("./logs/debug_before_message.png")
	fmt.Println("[DEBUG] Screenshot saved to ./logs/debug_before_message.png")

	// Click Message button
	if err := mm.clickMessageButton(); err != nil {
		return err
	}

	mm.stealth.RandomDelay()

	// Wait for message modal
	time.Sleep(1 * time.Second)

	// Personalize and type message
	personalizedMsg := mm.personalizeMessage(message, profile)

	if err := mm.typeMessage(personalizedMsg); err != nil {
		return err
	}

	mm.stealth.ThinkDelay()

	// Send the message
	if err := mm.clickSendButton(); err != nil {
		return err
	}

	// Record the action
	mm.rateLimiter.RecordAction("message")

	// Save to storage
	mm.saveMessage(profile, personalizedMsg)

	mm.log.Info("Message sent", map[string]interface{}{
		"name": profile.Name,
	})

	return nil
}

// clickMessageButton finds and clicks the Message button.
func (mm *MessagingManager) clickMessageButton() error {
	// Wait for page to fully load
	fmt.Println("[DEBUG] Waiting 3 seconds for page to load...")
	time.Sleep(3 * time.Second)

	fmt.Println("[DEBUG] Looking for Message button")

	// Try aria-label selectors first (most reliable)
	ariaSelectors := []string{
		"button[aria-label^='Message ']",
		"button[aria-label*='Message']",
	}

	for _, selector := range ariaSelectors {
		fmt.Printf("[DEBUG] Trying selector: %s\n", selector)
		if mm.browser.HasElement(selector) {
			fmt.Printf("[DEBUG] Found element with selector: %s, clicking...\n", selector)
			mm.stealth.HoverElement(mm.browser.GetPage(), selector)
			mm.stealth.ThinkDelay()
			if err := mm.browser.Click(selector); err == nil {
				fmt.Println("[DEBUG] Click successful!")
				return nil
			}
			fmt.Printf("[DEBUG] Click failed for selector: %s\n", selector)
		} else {
			fmt.Printf("[DEBUG] Element NOT found: %s\n", selector)
		}
	}

	// Fallback: find button by text content
	fmt.Println("[DEBUG] Trying text-based search for 'Message' button")
	if mm.browser.HasElementWithText("button", "Message") {
		fmt.Println("[DEBUG] Found button with text 'Message'")
		mm.stealth.ThinkDelay()
		if err := mm.browser.ClickElementWithText("button", "Message"); err == nil {
			return nil
		}
	} else {
		fmt.Println("[DEBUG] No button with text 'Message' found")
	}

	// Try primary button class (visible in the HTML)
	primarySelector := "button.artdeco-button--primary"
	fmt.Printf("[DEBUG] Trying primary button selector: %s\n", primarySelector)
	if mm.browser.HasElementWithText(primarySelector, "Message") {
		fmt.Println("[DEBUG] Found primary button with Message text")
		mm.stealth.ThinkDelay()
		if err := mm.browser.ClickElementWithText(primarySelector, "Message"); err == nil {
			return nil
		}
	}

	// Last resort: try generic selectors
	fallbackSelectors := []string{
		".entry-point button",
		".pvs-profile-actions button.artdeco-button--primary",
		"[data-control-name='message']",
	}

	for _, selector := range fallbackSelectors {
		fmt.Printf("[DEBUG] Trying fallback selector: %s\n", selector)
		if mm.browser.HasElement(selector) {
			fmt.Printf("[DEBUG] Found element: %s\n", selector)
			mm.stealth.HoverElement(mm.browser.GetPage(), selector)
			mm.stealth.ThinkDelay()
			if err := mm.browser.Click(selector); err == nil {
				return nil
			}
		}
	}

	// Debug: Check current URL
	currentURL, _ := mm.browser.GetCurrentURL()
	fmt.Printf("[DEBUG] Current URL: %s\n", currentURL)

	// Debug: Check if we're on a LinkedIn page
	pageHTML := mm.browser.GetPageHTML()
	if strings.Contains(pageHTML, "Message") {
		fmt.Println("[DEBUG] Page HTML contains 'Message' text")
	} else {
		fmt.Println("[DEBUG] Page HTML does NOT contain 'Message' text")
	}

	// Check for common LinkedIn elements
	if mm.browser.HasElement("button") {
		fmt.Println("[DEBUG] Page has buttons")
	}
	if mm.browser.HasElement(".artdeco-button") {
		fmt.Println("[DEBUG] Page has artdeco buttons")
	}

	return fmt.Errorf("message button not found - may not be connected")
}

// typeMessage types the message in the compose box.
func (mm *MessagingManager) typeMessage(message string) error {
	// Find message input
	inputSelectors := []string{
		// Exact LinkedIn message input selector
		"div.msg-form__contenteditable[contenteditable='true'][role='textbox']",
		"div.msg-form__contenteditable[contenteditable='true']",
		".msg-form__contenteditable",
		".msg-form__message-texteditor div[contenteditable='true']",
		"div[role='textbox'][contenteditable='true'][aria-label*='Write a message']",
		"form.msg-form div[contenteditable='true']",
	}

	for _, selector := range inputSelectors {
		if mm.browser.HasElement(selector) {
			// Click to focus
			mm.browser.Click(selector)
			mm.stealth.RandomDelay()

			// Type with human-like behavior
			return mm.stealth.HumanType(mm.browser.GetPage(), message)
		}
	}

	return fmt.Errorf("message input not found")
}

// clickSendButton clicks the send button.
func (mm *MessagingManager) clickSendButton() error {
	sendSelectors := []string{
		// Exact LinkedIn send button selector
		"button.msg-form__send-button[type='submit']",
		"button.msg-form__send-button",
		".msg-form__right-actions button.msg-form__send-button",
		"form.msg-form button[type='submit']",
		"button[type='submit']:has-text('Send')",
	}

	mm.stealth.ThinkDelay()

	for _, selector := range sendSelectors {
		if mm.browser.HasElement(selector) {
			return mm.browser.Click(selector)
		}
	}

	return fmt.Errorf("send button not found")
}

// personalizeMessage replaces placeholders in the message template.
func (mm *MessagingManager) personalizeMessage(template string, profile *search.SearchResult) string {
	msg := template

	// Replace placeholders
	msg = strings.ReplaceAll(msg, "{name}", profile.Name)
	msg = strings.ReplaceAll(msg, "{firstName}", mm.getFirstName(profile.Name))
	msg = strings.ReplaceAll(msg, "{title}", profile.Title)
	msg = strings.ReplaceAll(msg, "{company}", profile.Company)
	msg = strings.ReplaceAll(msg, "{location}", profile.Location)

	return msg
}

// getFirstName extracts the first name from a full name.
func (mm *MessagingManager) getFirstName(fullName string) string {
	parts := strings.Fields(fullName)
	if len(parts) > 0 {
		return parts[0]
	}
	return fullName
}

// saveMessage saves the message to storage.
func (mm *MessagingManager) saveMessage(profile *search.SearchResult, content string) {
	msg := &storage.Message{
		ProfileID:   mm.generateProfileID(profile.ProfileURL),
		ProfileURL:  profile.ProfileURL,
		ProfileName: profile.Name,
		Content:     content,
		Direction:   "sent",
		SentAt:      time.Now(),
	}

	if err := mm.storage.SaveMessage(msg); err != nil {
		mm.log.Warn("Failed to save message", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// generateProfileID generates a profile ID from URL.
func (mm *MessagingManager) generateProfileID(profileURL string) string {
	parts := strings.Split(profileURL, "/in/")
	if len(parts) < 2 {
		return fmt.Sprintf("profile_%d", time.Now().UnixNano())
	}
	return strings.TrimSuffix(parts[1], "/")
}

// SendBulkMessages sends messages to multiple profiles.
func (mm *MessagingManager) SendBulkMessages(profiles []*search.SearchResult, message string) *BulkMessageResult {
	result := &BulkMessageResult{
		Total:     len(profiles),
		Succeeded: 0,
		Failed:    0,
		Skipped:   0,
		StartTime: time.Now(),
	}

	mm.log.Info("Starting bulk messages", map[string]interface{}{
		"total": len(profiles),
	})

	for i, profile := range profiles {
		// Check rate limit
		if !mm.rateLimiter.CanProceed("message") {
			mm.log.Info("Rate limit reached, stopping bulk send", map[string]interface{}{
				"completed": i,
				"remaining": len(profiles) - i,
			})
			result.Skipped = len(profiles) - i
			break
		}

		// Check schedule
		mm.stealth.WaitForSchedule()

		// Take break if needed
		if mm.stealth.ShouldTakeBreak() {
			mm.stealth.TakeBreak()
		}

		// Skip if already messaged
		if mm.storage.WasMessageSent(profile.ProfileURL) {
			mm.log.Debug("Skipping - message already sent", map[string]interface{}{
				"url": profile.ProfileURL,
			})
			result.Skipped++
			continue
		}

		// Skip if not a connection
		if !profile.IsConnection {
			mm.log.Debug("Skipping - not a connection", map[string]interface{}{
				"url": profile.ProfileURL,
			})
			result.Skipped++
			continue
		}

		// Send message
		if err := mm.SendMessage(profile, message); err != nil {
			mm.log.Warn("Failed to send message", map[string]interface{}{
				"name":  profile.Name,
				"error": err.Error(),
			})
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", profile.Name, err.Error()))
		} else {
			result.Succeeded++
		}

		// Random delay between messages
		mm.stealth.RandomDelay()
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	mm.log.Info("Bulk messaging completed", map[string]interface{}{
		"succeeded": result.Succeeded,
		"failed":    result.Failed,
		"skipped":   result.Skipped,
		"duration":  result.Duration.String(),
	})

	return result
}

// BulkMessageResult contains the results of a bulk messaging operation.
type BulkMessageResult struct {
	Total     int
	Succeeded int
	Failed    int
	Skipped   int
	Errors    []string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
}

// GetMessageStats returns message statistics.
func (mm *MessagingManager) GetMessageStats() (map[string]int, error) {
	return mm.storage.GetMessageStats()
}

// OpenConversation opens a messaging conversation with a profile.
func (mm *MessagingManager) OpenConversation(profileURL string) error {
	mm.log.Debug("Opening conversation", map[string]interface{}{
		"url": profileURL,
	})

	if err := mm.browser.Navigate(profileURL); err != nil {
		return err
	}

	mm.stealth.PageDelay()

	return mm.clickMessageButton()
}

// CloseMessageModal closes any open message modal.
func (mm *MessagingManager) CloseMessageModal() error {
	closeSelectors := []string{
		"button.msg-overlay-bubble-header__control--close",
		"button[aria-label='Close conversation']",
		".msg-overlay-bubble-header button.artdeco-button--circle",
	}

	for _, selector := range closeSelectors {
		if mm.browser.HasElement(selector) {
			return mm.browser.Click(selector)
		}
	}

	return nil
}

// GetNewlyAcceptedConnections fetches connections that were recently accepted.
// It checks the "My Network" page for new connection notifications.
func (mm *MessagingManager) GetNewlyAcceptedConnections() ([]*search.SearchResult, error) {
	mm.log.Info("Checking for newly accepted connections", nil)

	// Navigate to My Network page
	if err := mm.browser.Navigate("https://www.linkedin.com/mynetwork/invite-connect/connections/"); err != nil {
		return nil, fmt.Errorf("failed to navigate to connections: %w", err)
	}

	mm.stealth.PageDelay()
	time.Sleep(2 * time.Second)

	var newConnections []*search.SearchResult

	// Look for connection cards
	connectionSelectors := []string{
		".mn-connection-card",
		".artdeco-list__item",
		"li.mn-connection-card",
	}

	var elements rod.Elements
	var err error
	for _, selector := range connectionSelectors {
		elements, err = mm.browser.GetElements(selector)
		if err == nil && len(elements) > 0 {
			break
		}
	}

	if len(elements) == 0 {
		mm.log.Debug("No connection cards found", nil)
		return newConnections, nil
	}

	// Extract connection info
	for _, el := range elements {
		result := &search.SearchResult{
			IsConnection: true,
		}

		// Extract name
		nameEl, err := el.Element(".mn-connection-card__name")
		if err == nil {
			text, _ := nameEl.Text()
			result.Name = strings.TrimSpace(text)
		}

		// Extract occupation/title
		occEl, err := el.Element(".mn-connection-card__occupation")
		if err == nil {
			text, _ := occEl.Text()
			result.Title = strings.TrimSpace(text)
		}

		// Extract profile link
		linkEl, err := el.Element("a[href*='/in/']")
		if err == nil {
			href, _ := linkEl.Attribute("href")
			if href != nil {
				result.ProfileURL = mm.cleanProfileURL(*href)
			}
		}

		// Check if this is a new connection (not already messaged)
		if result.ProfileURL != "" && !mm.storage.WasMessageSent(result.ProfileURL) {
			newConnections = append(newConnections, result)
		}
	}

	mm.log.Info("Found new connections", map[string]interface{}{
		"count": len(newConnections),
	})

	return newConnections, nil
}

// cleanProfileURL cleans and normalizes a LinkedIn profile URL.
func (mm *MessagingManager) cleanProfileURL(rawURL string) string {
	// Parse the URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	// Get just the path
	path := u.Path

	// Ensure it starts with /in/
	if !strings.HasPrefix(path, "/in/") {
		return ""
	}

	// Remove trailing slashes and query params
	path = strings.TrimSuffix(path, "/")

	return "https://www.linkedin.com" + path
}

// SendFollowUpToNewConnections sends follow-up messages to newly accepted connections.
func (mm *MessagingManager) SendFollowUpToNewConnections(messageTemplate string) *BulkMessageResult {
	result := &BulkMessageResult{
		StartTime: time.Now(),
	}

	// Get new connections
	connections, err := mm.GetNewlyAcceptedConnections()
	if err != nil {
		mm.log.Warn("Failed to get new connections", map[string]interface{}{
			"error": err.Error(),
		})
		return result
	}

	result.Total = len(connections)

	if len(connections) == 0 {
		mm.log.Info("No new connections to message", nil)
		return result
	}

	// Send messages to each new connection
	for _, conn := range connections {
		// Check rate limit
		if !mm.rateLimiter.CanProceed("message") {
			mm.log.Info("Rate limit reached, stopping follow-up messages", nil)
			result.Skipped = result.Total - result.Succeeded - result.Failed
			break
		}

		// Check schedule
		mm.stealth.WaitForSchedule()

		// Take break if needed
		if mm.stealth.ShouldTakeBreak() {
			mm.stealth.TakeBreak()
		}

		// Send message
		if err := mm.SendMessage(conn, messageTemplate); err != nil {
			mm.log.Warn("Failed to send follow-up", map[string]interface{}{
				"name":  conn.Name,
				"error": err.Error(),
			})
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", conn.Name, err.Error()))
		} else {
			result.Succeeded++
		}

		// Random delay between messages
		mm.stealth.RandomDelay()
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	mm.log.Info("Follow-up messages completed", map[string]interface{}{
		"succeeded": result.Succeeded,
		"failed":    result.Failed,
		"skipped":   result.Skipped,
	})

	return result
}
