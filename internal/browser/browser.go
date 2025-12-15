// Package browser provides browser management and initialization with stealth features.
package browser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/singh-yash129/link/internal/config"
	"github.com/singh-yash129/link/internal/logger"
)

// BrowserManager handles browser lifecycle and operations.
type BrowserManager struct {
	browser    *rod.Browser
	page       *rod.Page
	config     *config.Config
	log        *logger.Logger
	cookiePath string
}

// NewBrowserManager creates a new browser manager instance.
func NewBrowserManager(cfg *config.Config, log *logger.Logger) *BrowserManager {
	return &BrowserManager{
		config:     cfg,
		log:        log,
		cookiePath: cfg.Storage.CookiesPath,
	}
}

// Launch starts the browser with stealth configuration.
func (bm *BrowserManager) Launch() error {
	bm.log.Info("Launching browser", map[string]interface{}{
		"headless": bm.config.Browser.Headless,
	})

	// Ensure data directory exists
	dataDir := filepath.Dir(bm.config.Storage.DatabasePath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Try to find existing browser (Chrome, Edge, or Chromium)
	browserPath, found := launcher.LookPath()
	
	var l *launcher.Launcher
	if found {
		bm.log.Info("Using existing browser", map[string]interface{}{
			"path": browserPath,
		})
		l = launcher.New().Bin(browserPath)
	} else {
		bm.log.Info("No existing browser found, will download Chromium", nil)
		l = launcher.New()
	}

	// Configure launcher
	l = l.Leakless(false). // Disable leakless to avoid Windows Defender false positive
		Headless(bm.config.Browser.Headless).
		Set("disable-blink-features", "AutomationControlled").
		Set("disable-infobars").
		Set("disable-dev-shm-usage").
		Set("no-first-run").
		Set("no-default-browser-check")

	// Set user data directory for session persistence
	userDataDir := bm.config.Browser.UserDataDir
	if userDataDir == "" {
		userDataDir = filepath.Join(dataDir, "chrome-data")
	}
	l = l.UserDataDir(userDataDir)

	// Set proxy if configured
	if bm.config.Browser.ProxyURL != "" {
		l = l.Proxy(bm.config.Browser.ProxyURL)
	}

	// Launch browser
	url, err := l.Launch()
	if err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}

	// Connect to browser
	browser := rod.New().ControlURL(url)
	if err := browser.Connect(); err != nil {
		return fmt.Errorf("failed to connect to browser: %w", err)
	}

	bm.browser = browser

	// Create stealth page
	page, err := stealth.Page(browser)
	if err != nil {
		return fmt.Errorf("failed to create stealth page: %w", err)
	}

	bm.page = page

	// Set viewport
	if err := bm.SetViewport(bm.config.Browser.ViewportWidth, bm.config.Browser.ViewportHeight); err != nil {
		bm.log.Warn("Failed to set viewport", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Set user agent from list if available
	if len(bm.config.Browser.UserAgents) > 0 {
		if err := bm.SetUserAgent(bm.config.Browser.UserAgents[0]); err != nil {
			bm.log.Warn("Failed to set user agent", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	bm.log.Info("Browser launched successfully", nil)
	return nil
}

// Close closes the browser.
func (bm *BrowserManager) Close() error {
	if bm.browser != nil {
		bm.log.Info("Closing browser", nil)
		return bm.browser.Close()
	}
	return nil
}

// Navigate navigates to a URL.
func (bm *BrowserManager) Navigate(url string) error {
	bm.log.Debug("Navigating to URL", map[string]interface{}{
		"url": url,
	})
	return bm.page.Navigate(url)
}

// GetCurrentURL returns the current page URL.
func (bm *BrowserManager) GetCurrentURL() (string, error) {
	info, err := bm.page.Info()
	if err != nil {
		return "", err
	}
	return info.URL, nil
}

// Type types text into an element.
func (bm *BrowserManager) Type(selector, text string) error {
	el, err := bm.page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %s", selector)
	}
	return el.Input(text)
}

// Click clicks on an element.
func (bm *BrowserManager) Click(selector string) error {
	el, err := bm.page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %s", selector)
	}
	return el.Click(proto.InputMouseButtonLeft, 1)
}

// HasElement checks if an element exists on the page.
func (bm *BrowserManager) HasElement(selector string) bool {
	_, err := bm.page.Timeout(2 * time.Second).Element(selector)
	return err == nil
}

// WaitForElement waits for an element to appear.
func (bm *BrowserManager) WaitForElement(selector string, timeout time.Duration) error {
	_, err := bm.page.Timeout(timeout).Element(selector)
	return err
}

// GetElement returns an element by selector.
func (bm *BrowserManager) GetElement(selector string) (*rod.Element, error) {
	return bm.page.Element(selector)
}

// GetElements returns all matching elements.
func (bm *BrowserManager) GetElements(selector string) (rod.Elements, error) {
	return bm.page.Elements(selector)
}

// GetText returns text content of an element.
func (bm *BrowserManager) GetText(selector string) (string, error) {
	el, err := bm.page.Element(selector)
	if err != nil {
		return "", err
	}
	return el.Text()
}

// Screenshot takes a screenshot of the current page.
func (bm *BrowserManager) Screenshot(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := bm.page.Screenshot(true, nil)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// SetViewport sets the browser viewport size.
func (bm *BrowserManager) SetViewport(width, height int) error {
	return bm.page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  width,
		Height: height,
	})
}

// SetUserAgent sets the browser user agent.
func (bm *BrowserManager) SetUserAgent(userAgent string) error {
	return bm.page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent: userAgent,
	})
}

// WaitStable waits for the page to become stable.
func (bm *BrowserManager) WaitStable() {
	bm.page.MustWaitStable()
}

// WaitLoad waits for the page to finish loading.
func (bm *BrowserManager) WaitLoad() error {
	return bm.page.WaitLoad()
}

// SaveCookies saves current cookies to file.
func (bm *BrowserManager) SaveCookies() error {
	cookies, err := bm.page.Cookies(nil)
	if err != nil {
		return fmt.Errorf("failed to get cookies: %w", err)
	}

	data, err := json.MarshalIndent(cookies, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cookies: %w", err)
	}

	if err := os.WriteFile(bm.cookiePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write cookies: %w", err)
	}

	bm.log.Debug("Cookies saved", map[string]interface{}{
		"path":  bm.cookiePath,
		"count": len(cookies),
	})

	return nil
}

// LoadCookies loads cookies from file.
func (bm *BrowserManager) LoadCookies() error {
	data, err := os.ReadFile(bm.cookiePath)
	if err != nil {
		if os.IsNotExist(err) {
			bm.log.Debug("No saved cookies found", nil)
			return nil
		}
		return fmt.Errorf("failed to read cookies: %w", err)
	}

	var cookies []*proto.NetworkCookie
	if err := json.Unmarshal(data, &cookies); err != nil {
		return fmt.Errorf("failed to unmarshal cookies: %w", err)
	}

	// Convert to CookieParam for setting
	var params []*proto.NetworkCookieParam
	for _, c := range cookies {
		params = append(params, &proto.NetworkCookieParam{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HTTPOnly: c.HTTPOnly,
			SameSite: c.SameSite,
			Expires:  proto.TimeSinceEpoch(c.Expires),
		})
	}

	if err := bm.page.SetCookies(params); err != nil {
		return fmt.Errorf("failed to set cookies: %w", err)
	}

	bm.log.Debug("Cookies loaded", map[string]interface{}{
		"count": len(params),
	})

	return nil
}

// GetPage returns the current page.
func (bm *BrowserManager) GetPage() *rod.Page {
	return bm.page
}

// GetBrowser returns the browser instance.
func (bm *BrowserManager) GetBrowser() *rod.Browser {
	return bm.browser
}

// Eval evaluates JavaScript on the page.
func (bm *BrowserManager) Eval(js string) (*proto.RuntimeRemoteObject, error) {
	return bm.page.Eval(js)
}

// ScrollTo scrolls to a specific position.
func (bm *BrowserManager) ScrollTo(x, y float64) error {
	return bm.page.Mouse.Scroll(x, y, 1)
}

// MoveMouse moves the mouse to a position.
func (bm *BrowserManager) MoveMouse(x, y float64) error {
	return bm.page.Mouse.MoveTo(proto.Point{X: x, Y: y})
}

// Reload reloads the current page.
func (bm *BrowserManager) Reload() error {
	return bm.page.Reload()
}

// WaitVisible waits for an element to become visible.
func (bm *BrowserManager) WaitVisible(selector string, timeout time.Duration) error {
	el, err := bm.page.Timeout(timeout).Element(selector)
	if err != nil {
		return err
	}
	return el.WaitVisible()
}

// GetAttribute gets an attribute value from an element.
func (bm *BrowserManager) GetAttribute(selector, attr string) (string, error) {
	el, err := bm.page.Element(selector)
	if err != nil {
		return "", err
	}
	val, err := el.Attribute(attr)
	if err != nil {
		return "", err
	}
	if val == nil {
		return "", nil
	}
	return *val, nil
}

// GetPageHTML returns the full page HTML.
func (bm *BrowserManager) GetPageHTML() (string, error) {
	return bm.page.HTML()
}
