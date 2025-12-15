// Package auth handles LinkedIn authentication and session management.
// It handles login, session management, and security checkpoint detection.
package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/singh-yash129/link/internal/browser"
	"github.com/singh-yash129/link/internal/config"
	"github.com/singh-yash129/link/internal/logger"
	"github.com/singh-yash129/link/internal/stealth"
)

// Common LinkedIn URLs and selectors
const (
	LinkedInLoginURL   = "https://www.linkedin.com/login"
	LinkedInHomeURL    = "https://www.linkedin.com/feed/"
	LinkedInCheckURL   = "https://www.linkedin.com/check/add-phone"
	LinkedInCaptchaURL = "https://www.linkedin.com/checkpoint/challenge"

	// Login page selectors
	SelectorEmailInput    = "#username"
	SelectorPasswordInput = "#password"
	SelectorLoginButton   = "button[type='submit']"

	// Post-login selectors
	SelectorFeedContainer = ".feed-shared-update-v2"
	SelectorNavBar        = ".global-nav"
	SelectorProfilePic    = ".feed-identity-module__member-photo"

	// Security checkpoint selectors
	SelectorCaptcha       = ".captcha"
	Selector2FA           = "[data-test='verification-code-input']"
	SelectorPhoneVerify   = ".phone-verification"
	SelectorSecurityCheck = ".checkpoint"
)

// AuthError represents authentication-specific errors.
type AuthError struct {
	Type    string
	Message string
	Err     error
}

func (e *AuthError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Error types
const (
	ErrTypeCredentials   = "CREDENTIALS_ERROR"
	ErrTypeCaptcha       = "CAPTCHA_REQUIRED"
	ErrType2FA           = "2FA_REQUIRED"
	ErrTypePhoneVerify   = "PHONE_VERIFICATION"
	ErrTypeAccountLocked = "ACCOUNT_LOCKED"
	ErrTypeNetworkError  = "NETWORK_ERROR"
	ErrTypeUnknown       = "UNKNOWN_ERROR"
)

// AuthManager handles LinkedIn authentication.
type AuthManager struct {
	browser *browser.BrowserManager
	config  *config.Config
	log     *logger.Logger
	stealth *stealth.StealthManager
}

// NewAuthManager creates a new authentication manager.
func NewAuthManager(bm *browser.BrowserManager, cfg *config.Config, log *logger.Logger, sm *stealth.StealthManager) *AuthManager {
	return &AuthManager{
		browser: bm,
		config:  cfg,
		log:     log,
		stealth: sm,
	}
}

// IsLoggedIn checks if there's an active LinkedIn session.
func (am *AuthManager) IsLoggedIn() bool {
	am.log.Debug("Checking login status", nil)

	// Try to navigate to LinkedIn home
	if err := am.browser.Navigate(LinkedInHomeURL); err != nil {
		am.log.Debug("Navigation failed", map[string]interface{}{
			"error": err.Error(),
		})
		return false
	}

	// Wait for page to stabilize (increased for slow connections)
	time.Sleep(4 * time.Second)

	// Get current URL
	currentURL, err := am.browser.GetCurrentURL()
	if err != nil {
		return false
	}

	// Check if redirected to login
	if strings.Contains(currentURL, "/login") || strings.Contains(currentURL, "/authwall") {
		am.log.Debug("Not logged in - redirected to login page", nil)
		return false
	}

	// Check for feed URL (indicates logged in)
	if strings.Contains(currentURL, "/feed") {
		am.log.Info("Active session found", nil)
		return true
	}

	// Check for nav bar (indicates logged in state)
	if am.browser.HasElement(SelectorNavBar) {
		am.log.Info("Active session found", nil)
		return true
	}

	return false
}

// Login performs LinkedIn login with credentials.
func (am *AuthManager) Login() error {
	endOp := am.log.StartOperation("LinkedIn login")

	// First, try to use existing session
	if err := am.browser.LoadCookies(); err == nil {
		if am.IsLoggedIn() {
			endOp(nil)
			return nil
		}
	}

	am.log.Info("Starting fresh login", map[string]interface{}{
		"email": am.maskEmail(am.config.LinkedIn.Email),
	})

	// Navigate to login page
	if err := am.browser.Navigate(LinkedInLoginURL); err != nil {
		err = &AuthError{Type: ErrTypeNetworkError, Message: "Failed to load login page", Err: err}
		endOp(err)
		return err
	}

	// Wait for page to load
	time.Sleep(2 * time.Second)

	// Check for any security challenges before login
	if err := am.checkSecurityChallenge(); err != nil {
		endOp(err)
		return err
	}

	// Enter email
	am.log.Debug("Entering email", nil)
	am.stealth.ThinkDelay()
	if err := am.browser.Type(SelectorEmailInput, am.config.LinkedIn.Email); err != nil {
		err = &AuthError{Type: ErrTypeUnknown, Message: "Failed to enter email", Err: err}
		endOp(err)
		return err
	}

	// Random pause between fields
	am.stealth.RandomDelay()

	// Enter password
	am.log.Debug("Entering password", nil)
	if err := am.browser.Type(SelectorPasswordInput, am.config.LinkedIn.Password); err != nil {
		err = &AuthError{Type: ErrTypeUnknown, Message: "Failed to enter password", Err: err}
		endOp(err)
		return err
	}

	// Random pause before clicking
	am.stealth.ThinkDelay()

	// Click login button
	am.log.Debug("Clicking login button", nil)
	if err := am.browser.Click(SelectorLoginButton); err != nil {
		err = &AuthError{Type: ErrTypeUnknown, Message: "Failed to click login button", Err: err}
		endOp(err)
		return err
	}

	// Wait for response
	time.Sleep(3 * time.Second)

	// Check for security challenges
	if err := am.checkSecurityChallenge(); err != nil {
		endOp(err)
		return err
	}

	// Verify login success
	if err := am.verifyLoginSuccess(); err != nil {
		endOp(err)
		return err
	}

	// Save cookies for session persistence
	if err := am.browser.SaveCookies(); err != nil {
		am.log.Warn("Failed to save cookies", map[string]interface{}{
			"error": err.Error(),
		})
	}

	endOp(nil)
	return nil
}

// checkSecurityChallenge checks for and handles security challenges.
func (am *AuthManager) checkSecurityChallenge() error {
	currentURL, _ := am.browser.GetCurrentURL()

	// Check for CAPTCHA
	if strings.Contains(currentURL, "challenge") || am.browser.HasElement(SelectorCaptcha) {
		am.log.Warn("CAPTCHA detected", map[string]interface{}{
			"url": currentURL,
		})
		// Take screenshot for manual review
		am.browser.Screenshot("./data/captcha_challenge.png")
		return &AuthError{
			Type:    ErrTypeCaptcha,
			Message: "CAPTCHA verification required. Please solve manually and retry.",
		}
	}

	// Check for 2FA
	if am.browser.HasElement(Selector2FA) {
		am.log.Warn("2FA required", nil)
		return &AuthError{
			Type:    ErrType2FA,
			Message: "Two-factor authentication required. Please complete manually.",
		}
	}

	// Check for phone verification
	if strings.Contains(currentURL, "add-phone") || am.browser.HasElement(SelectorPhoneVerify) {
		am.log.Warn("Phone verification required", nil)
		return &AuthError{
			Type:    ErrTypePhoneVerify,
			Message: "Phone verification required. Please complete manually.",
		}
	}

	// Check for general security checkpoint
	if strings.Contains(currentURL, "checkpoint") || am.browser.HasElement(SelectorSecurityCheck) {
		am.log.Warn("Security checkpoint detected", nil)
		am.browser.Screenshot("./data/security_checkpoint.png")
		return &AuthError{
			Type:    ErrTypeUnknown,
			Message: "Security checkpoint detected. Please review manually.",
		}
	}

	return nil
}

// verifyLoginSuccess confirms successful login.
func (am *AuthManager) verifyLoginSuccess() error {
	// Wait for page transition
	time.Sleep(3 * time.Second)

	currentURL, err := am.browser.GetCurrentURL()
	if err != nil {
		return &AuthError{Type: ErrTypeNetworkError, Message: "Failed to get current URL", Err: err}
	}

	// Check if still on login page (indicates failure)
	if strings.Contains(currentURL, "/login") {
		// Check for error messages
		if am.browser.HasElement(".form__label--error") {
			return &AuthError{
				Type:    ErrTypeCredentials,
				Message: "Invalid credentials. Please check email and password.",
			}
		}
		return &AuthError{
			Type:    ErrTypeUnknown,
			Message: "Login failed. Still on login page.",
		}
	}

	// Check for nav bar presence (indicates success)
	am.browser.WaitStable()
	if am.browser.HasElement(SelectorNavBar) {
		am.log.Info("Login successful", nil)
		return nil
	}

	// Additional check - look for feed content
	if strings.Contains(currentURL, "/feed") {
		am.log.Info("Login successful - on feed page", nil)
		return nil
	}

	return &AuthError{
		Type:    ErrTypeUnknown,
		Message: fmt.Sprintf("Unable to verify login success. Current URL: %s", currentURL),
	}
}

// Logout performs LinkedIn logout.
func (am *AuthManager) Logout() error {
	am.log.Info("Logging out", nil)

	// Navigate to logout URL
	if err := am.browser.Navigate("https://www.linkedin.com/m/logout/"); err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	// Clear saved cookies
	// Note: We don't delete cookies from storage to allow re-login

	return nil
}

// RefreshSession refreshes the LinkedIn session.
func (am *AuthManager) RefreshSession() error {
	am.log.Info("Refreshing session", nil)

	// Navigate to home to trigger any session refresh
	if err := am.browser.Navigate(LinkedInHomeURL); err != nil {
		return err
	}

	// Check if still logged in
	if !am.IsLoggedIn() {
		return am.Login()
	}

	// Save updated cookies
	return am.browser.SaveCookies()
}

// WaitForManualVerification waits for user to complete manual verification.
func (am *AuthManager) WaitForManualVerification(timeout time.Duration) error {
	am.log.Info("Waiting for manual verification", map[string]interface{}{
		"timeout_seconds": timeout.Seconds(),
	})

	startTime := time.Now()
	checkInterval := 5 * time.Second

	for time.Since(startTime) < timeout {
		// Check if verification completed
		if err := am.checkSecurityChallenge(); err == nil {
			if am.IsLoggedIn() {
				am.log.Info("Manual verification completed", nil)
				return am.browser.SaveCookies()
			}
		}
		time.Sleep(checkInterval)
	}

	return errors.New("manual verification timeout")
}

// GetSessionInfo returns information about the current session.
func (am *AuthManager) GetSessionInfo() map[string]interface{} {
	loggedIn := am.IsLoggedIn()
	info := map[string]interface{}{
		"logged_in": loggedIn,
	}

	if loggedIn {
		currentURL, _ := am.browser.GetCurrentURL()
		info["current_url"] = currentURL

		// Try to get profile name
		if am.browser.HasElement(".feed-identity-module__actor-meta") {
			if el, err := am.browser.GetElement(".feed-identity-module__actor-meta"); err == nil {
				if text, err := el.Text(); err == nil {
					info["profile_name"] = strings.TrimSpace(text)
				}
			}
		}
	}

	return info
}

// maskEmail masks email for logging.
func (am *AuthManager) maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***"
	}
	name := parts[0]
	domain := parts[1]
	if len(name) <= 2 {
		return name[:1] + "***@" + domain
	}
	return name[:2] + "***@" + domain
}

// DetectLoginState returns the current login state.
type LoginState int

const (
	StateUnknown LoginState = iota
	StateLoggedOut
	StateLoggedIn
	StateCaptchaRequired
	State2FARequired
	StatePhoneVerifyRequired
	StateSecurityCheck
)

// GetLoginState returns the current login state.
func (am *AuthManager) GetLoginState() LoginState {
	currentURL, err := am.browser.GetCurrentURL()
	if err != nil {
		return StateUnknown
	}

	// Check for security challenges
	if strings.Contains(currentURL, "challenge") || am.browser.HasElement(SelectorCaptcha) {
		return StateCaptchaRequired
	}

	if am.browser.HasElement(Selector2FA) {
		return State2FARequired
	}

	if strings.Contains(currentURL, "add-phone") || am.browser.HasElement(SelectorPhoneVerify) {
		return StatePhoneVerifyRequired
	}

	if strings.Contains(currentURL, "checkpoint") {
		return StateSecurityCheck
	}

	if strings.Contains(currentURL, "/login") {
		return StateLoggedOut
	}

	if am.browser.HasElement(SelectorNavBar) {
		return StateLoggedIn
	}

	return StateUnknown
}

// HandleSecurityChallenge attempts to handle or wait for security challenges.
func (am *AuthManager) HandleSecurityChallenge() error {
	state := am.GetLoginState()

	switch state {
	case StateCaptchaRequired:
		am.log.Warn("CAPTCHA required - please solve manually", nil)
		return am.WaitForManualVerification(5 * time.Minute)
	case State2FARequired:
		am.log.Warn("2FA required - please complete verification", nil)
		return am.WaitForManualVerification(2 * time.Minute)
	case StatePhoneVerifyRequired:
		am.log.Warn("Phone verification required - please complete", nil)
		return am.WaitForManualVerification(2 * time.Minute)
	case StateSecurityCheck:
		am.log.Warn("Security check required - please complete", nil)
		return am.WaitForManualVerification(5 * time.Minute)
	case StateLoggedIn:
		return nil
	default:
		return errors.New("unknown authentication state")
	}
}
