// Package messaging handles LinkedIn messaging functionality.
package messaging

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

	mm.stealth.PageDelay()

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
	messageSelectors := []string{
		// Primary Message button - aria-label is "Message {Name}"
		"button[aria-label^='Message ']",
		"button[aria-label*='Message']",
		"button.artdeco-button--secondary:has-text('Message')",
		".pvs-profile-actions button:has-text('Message')",
		"button:has-text('Message'):not([disabled])",
		"[data-control-name='message']",
	}

	for _, selector := range messageSelectors {
		if mm.browser.HasElement(selector) {
			mm.stealth.HoverElement(mm.browser.GetPage(), selector)
			mm.stealth.ThinkDelay()
			if err := mm.browser.Click(selector); err == nil {
				return nil
			}
		}
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
