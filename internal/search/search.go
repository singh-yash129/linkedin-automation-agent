// Package search provides LinkedIn search functionality.
package search

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/singh-yash129/linkedin-automation-agent/internal/browser"
	"github.com/singh-yash129/linkedin-automation-agent/internal/config"
	"github.com/singh-yash129/linkedin-automation-agent/internal/logger"
	"github.com/singh-yash129/linkedin-automation-agent/internal/stealth"
	"github.com/singh-yash129/linkedin-automation-agent/internal/storage"
)

// SearchResult represents a single search result.
type SearchResult struct {
	ProfileURL    string
	Name          string
	Title         string
	Company       string
	Location      string
	ConnectionDeg string
	IsConnection  bool
}

// SearchManager handles LinkedIn search operations.
type SearchManager struct {
	browser *browser.BrowserManager
	config  *config.Config
	log     *logger.Logger
	stealth *stealth.StealthManager
	storage storage.Storage
}

// NewSearchManager creates a new search manager.
func NewSearchManager(bm *browser.BrowserManager, cfg *config.Config, log *logger.Logger, sm *stealth.StealthManager, store storage.Storage) *SearchManager {
	return &SearchManager{
		browser: bm,
		config:  cfg,
		log:     log,
		stealth: sm,
		storage: store,
	}
}

// SearchFilters contains search filter options.
type SearchFilters struct {
	Keywords        string
	Location        string
	Company         string
	Title           string
	Industry        string
	ConnectionLevel string // 1st, 2nd, 3rd
	MaxResults      int
	SkipDuplicates  bool // Skip profiles already in storage
}

// Search performs a LinkedIn people search with the given filters.
func (sm *SearchManager) Search(filters SearchFilters) ([]*SearchResult, error) {
	endOp := sm.log.StartOperation("LinkedIn search")

	// Build search URL
	searchURL := sm.buildSearchURL(filters)

	sm.log.Info("Performing search", map[string]interface{}{
		"keywords":   filters.Keywords,
		"location":   filters.Location,
		"company":    filters.Company,
		"title":      filters.Title,
		"maxResults": filters.MaxResults,
	})

	// Track seen URLs to avoid duplicates within search
	seenURLs := make(map[string]bool)

	// Navigate to search
	if err := sm.browser.Navigate(searchURL); err != nil {
		endOp(err)
		return nil, fmt.Errorf("failed to navigate to search: %w", err)
	}

	sm.stealth.PageDelay()

	// Wait for search page to load
	if err := sm.browser.WaitForElement(".search-results-container", 10*time.Second); err != nil {
		sm.log.Warn("Search results container not found, waiting longer", nil)
		time.Sleep(3 * time.Second)
	}

	// Apply filters using LinkedIn's filter UI (Location, Company, Title)
	if filters.Location != "" || filters.Company != "" || filters.Title != "" {
		sm.log.Info("Applying search filters via UI", nil)
		if err := sm.applySearchFilters(filters); err != nil {
			sm.log.Warn("Some filters may not have been applied", map[string]interface{}{
				"error": err.Error(),
			})
		}
		// Wait for filtered results to load
		time.Sleep(2 * time.Second)
	}

	var allResults []*SearchResult
	page := 1
	maxResults := filters.MaxResults
	if maxResults == 0 {
		maxResults = 100
	}

	for len(allResults) < maxResults {
		sm.log.Debug("Processing search page", map[string]interface{}{
			"page":          page,
			"resultsFound":  len(allResults),
			"targetResults": maxResults,
		})

		// Wait for results container to load
		if err := sm.browser.WaitForElement(".search-results-container", 10*time.Second); err != nil {
			sm.log.Warn("Search results not found, may be end of results", nil)
			break
		}

		// Scroll down to trigger lazy loading of profile links
		sm.log.Debug("Scrolling to load lazy content", nil)
		rodPage := sm.browser.GetPage()
		for i := 0; i < 3; i++ {
			rodPage.Mouse.Scroll(0, 500, 1)
			time.Sleep(500 * time.Millisecond)
		}
		// Scroll back up
		rodPage.Mouse.Scroll(0, -1000, 1)
		time.Sleep(1 * time.Second)

		// Extract results from current page
		results, err := sm.extractSearchResults()
		if err != nil {
			sm.log.Warn("Failed to extract results", map[string]interface{}{
				"error": err.Error(),
			})
			break
		}

		if len(results) == 0 {
			sm.log.Debug("No more results found", nil)
			break
		}

		// Filter duplicates from this page
		for _, r := range results {
			// Skip if URL already seen in this search
			if seenURLs[r.ProfileURL] {
				sm.log.Debug("Skipping duplicate profile", map[string]interface{}{
					"url": r.ProfileURL,
				})
				continue
			}
			seenURLs[r.ProfileURL] = true

			// Skip if already in storage (if enabled)
			if filters.SkipDuplicates && sm.storage.ProfileExists(r.ProfileURL) {
				sm.log.Debug("Skipping existing profile", map[string]interface{}{
					"url": r.ProfileURL,
				})
				continue
			}

			// Skip if connection already sent
			if filters.SkipDuplicates && sm.storage.WasConnectionSent(r.ProfileURL) {
				sm.log.Debug("Skipping - connection already sent", map[string]interface{}{
					"url": r.ProfileURL,
				})
				continue
			}

			allResults = append(allResults, r)
		}

		// Check if we have enough results
		if len(allResults) >= maxResults {
			break
		}

		// Random scroll behavior
		sm.stealth.RandomScroll(sm.browser.GetPage())
		sm.stealth.RandomDelay()

		// Try to go to next page
		if !sm.goToNextPage() {
			sm.log.Debug("No next page available", nil)
			break
		}

		page++
		sm.stealth.PageDelay()
	}

	// Trim to max results
	if len(allResults) > maxResults {
		allResults = allResults[:maxResults]
	}

	sm.log.Info("Search completed", map[string]interface{}{
		"totalResults": len(allResults),
		"pages":        page,
	})

	endOp(nil)
	return allResults, nil
}

// buildSearchURL constructs the LinkedIn search URL with filters.
// LinkedIn uses a complex URL structure with encoded filters.
func (sm *SearchManager) buildSearchURL(filters SearchFilters) string {
	baseURL := "https://www.linkedin.com/search/results/people/?"
	params := url.Values{}

	// Add keywords if provided
	if filters.Keywords != "" {
		params.Add("keywords", filters.Keywords)
	}

	// Add network filter (connection level)
	if filters.ConnectionLevel != "" {
		network := sm.connectionLevelToNetwork(filters.ConnectionLevel)
		if network != "" {
			params.Add("network", network)
		}
	}

	// Origin
	params.Add("origin", "GLOBAL_SEARCH_HEADER")

	return baseURL + params.Encode()
}

// applySearchFilters applies location, company, and title filters using LinkedIn's filter UI
func (sm *SearchManager) applySearchFilters(filters SearchFilters) error {
	page := sm.browser.GetPage()

	// Apply Location filter
	if filters.Location != "" {
		sm.log.Info("Applying location filter", map[string]interface{}{"location": filters.Location})
		if err := sm.applyFilter("Locations", filters.Location); err != nil {
			sm.log.Warn("Could not apply location filter via UI, will use keyword", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Apply Company filter
	if filters.Company != "" {
		sm.log.Info("Applying company filter", map[string]interface{}{"company": filters.Company})
		if err := sm.applyFilter("Current company", filters.Company); err != nil {
			sm.log.Warn("Could not apply company filter via UI, will use keyword", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Apply Title filter
	if filters.Title != "" {
		sm.log.Info("Applying title filter", map[string]interface{}{"title": filters.Title})
		if err := sm.applyFilter("Title", filters.Title); err != nil {
			sm.log.Warn("Could not apply title filter via UI, will use keyword", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Wait for results to update
	time.Sleep(2 * time.Second)
	_ = page
	return nil
}

// applyFilter clicks on a filter button and enters the value
func (sm *SearchManager) applyFilter(filterName, value string) error {
	page := sm.browser.GetPage()

	// Click "All filters" button to open filter panel
	allFiltersSelectors := []string{
		"button[aria-label='Show all filters']",
		"button:has-text('All filters')",
		".search-reusables__all-filters-pill-button",
	}

	filterPanelOpened := false
	for _, sel := range allFiltersSelectors {
		if el, err := page.Element(sel); err == nil {
			el.Click(proto.InputMouseButtonLeft, 1)
			time.Sleep(1 * time.Second)
			filterPanelOpened = true
			break
		}
	}

	if !filterPanelOpened {
		// Try clicking on specific filter button directly
		filterBtnSelectors := []string{
			fmt.Sprintf("button[aria-label*='%s']", filterName),
			fmt.Sprintf("button:has-text('%s')", filterName),
		}
		for _, sel := range filterBtnSelectors {
			if el, err := page.Element(sel); err == nil {
				el.Click(proto.InputMouseButtonLeft, 1)
				time.Sleep(1 * time.Second)
				filterPanelOpened = true
				break
			}
		}
	}

	if !filterPanelOpened {
		return fmt.Errorf("could not open filter panel")
	}

	// Find the filter section and input
	// Look for input field in the filter section
	inputSelectors := []string{
		fmt.Sprintf("input[placeholder*='%s']", filterName),
		fmt.Sprintf("input[aria-label*='%s']", filterName),
		"input[placeholder='Add a location']",
		"input[placeholder='Add a company']",
		"input[placeholder='Add a title']",
	}

	var inputEl *rod.Element
	for _, sel := range inputSelectors {
		if el, err := page.Element(sel); err == nil {
			inputEl = el
			break
		}
	}

	if inputEl == nil {
		// Close filter panel
		if closeBtn, err := page.Element("button[aria-label='Dismiss']"); err == nil {
			closeBtn.Click(proto.InputMouseButtonLeft, 1)
		}
		return fmt.Errorf("filter input not found for %s", filterName)
	}

	// Type the value
	inputEl.Click(proto.InputMouseButtonLeft, 1)
	time.Sleep(500 * time.Millisecond)
	inputEl.Input(value)
	time.Sleep(1 * time.Second)

	// Select first suggestion from dropdown
	suggestionSelectors := []string{
		".basic-typeahead__selectable:first-child",
		".search-typeahead-v2__hit:first-child",
		"[role='option']:first-child",
	}

	for _, sel := range suggestionSelectors {
		if suggEl, err := page.Element(sel); err == nil {
			suggEl.Click(proto.InputMouseButtonLeft, 1)
			time.Sleep(500 * time.Millisecond)
			break
		}
	}

	// Click "Show results" button
	showResultsSelectors := []string{
		"button[aria-label='Apply current filters to show results']",
		"button:has-text('Show results')",
		".search-reusables__secondary-filters-show-results-button",
	}

	for _, sel := range showResultsSelectors {
		if btn, err := page.Element(sel); err == nil {
			btn.Click(proto.InputMouseButtonLeft, 1)
			time.Sleep(2 * time.Second)
			break
		}
	}

	return nil
}

// connectionLevelToNetwork converts connection level to LinkedIn network parameter.
func (sm *SearchManager) connectionLevelToNetwork(level string) string {
	switch strings.ToLower(level) {
	case "1st", "1":
		return "[\"F\"]"
	case "2nd", "2":
		return "[\"S\"]"
	case "3rd", "3":
		return "[\"O\"]"
	case "2nd+3rd", "2+3":
		return "[\"S\",\"O\"]"
	default:
		return ""
	}
}

// extractSearchResults extracts profile information from search results.
func (sm *SearchManager) extractSearchResults() ([]*SearchResult, error) {
	var results []*SearchResult

	// Wait a bit for results to fully render
	time.Sleep(2 * time.Second)

	page := sm.browser.GetPage()

	// Scroll to load all content
	for i := 0; i < 5; i++ {
		page.Mouse.Scroll(0, 400, 1)
		time.Sleep(300 * time.Millisecond)
	}
	page.Mouse.Scroll(0, -2000, 1)
	time.Sleep(1 * time.Second)

	// Try multiple selectors for result cards
	selectors := []string{
		"li.reusable-search__result-container",
		"div.entity-result",
		"div[data-chameleon-result-urn]",
	}

	var elements rod.Elements
	var err error
	for _, selector := range selectors {
		elements, err = sm.browser.GetElements(selector)
		if err == nil && len(elements) > 0 {
			sm.log.Debug("Found result containers", map[string]interface{}{
				"selector": selector,
				"count":    len(elements),
			})
			break
		}
	}

	if len(elements) == 0 {
		return nil, nil
	}

	// Save original URL to return to
	originalURL, _ := sm.browser.GetCurrentURL()

	for i, el := range elements {
		result := &SearchResult{}

		// Extract name from various selectors
		nameSelectors := []string{
			"span.entity-result__title-text a span[aria-hidden='true']",
			".entity-result__title-text a",
			"a.app-aware-link span[aria-hidden='true']",
			".t-16 a span",
			"span[dir='ltr'] > span[aria-hidden='true']",
		}
		for _, sel := range nameSelectors {
			nameEl, err := el.Element(sel)
			if err == nil {
				text, _ := nameEl.Text()
				text = strings.TrimSpace(text)
				if text != "" && len(text) < 100 && text != "View" && !strings.HasPrefix(text, "View ") {
					result.Name = text
					break
				}
			}
		}

		// Extract title/headline
		titleSelectors := []string{
			".entity-result__primary-subtitle",
			"div.entity-result__primary-subtitle",
			".t-14.t-black.t-normal",
		}
		for _, sel := range titleSelectors {
			titleEl, err := el.Element(sel)
			if err == nil {
				text, _ := titleEl.Text()
				result.Title = strings.TrimSpace(text)
				if result.Title != "" {
					break
				}
			}
		}

		// Extract location
		locSelectors := []string{
			".entity-result__secondary-subtitle",
			"div.entity-result__secondary-subtitle",
			".t-14.t-normal.t-black--light",
		}
		for _, sel := range locSelectors {
			locEl, err := el.Element(sel)
			if err == nil {
				text, _ := locEl.Text()
				result.Location = strings.TrimSpace(text)
				if result.Location != "" {
					break
				}
			}
		}

		// Try to get profile URL by clicking on the name link
		linkSelectors := []string{
			"a.app-aware-link[href*='/in/']",
			".entity-result__title-text a[href*='/in/']",
			"a[href*='/in/']",
		}

		profileURL := ""
		for _, sel := range linkSelectors {
			linkEl, err := el.Element(sel)
			if err == nil {
				href, _ := linkEl.Attribute("href")
				if href != nil && strings.Contains(*href, "/in/") {
					profileURL = sm.cleanProfileURL(*href)
					break
				}
			}
		}

		// If still no URL, try clicking to navigate
		if profileURL == "" || profileURL == "https://www.linkedin.com" {
			sm.log.Debug("No direct URL found, attempting to click", map[string]interface{}{
				"index": i + 1,
				"name":  result.Name,
			})

			// Click on the name/title link
			clickSelectors := []string{
				"a.app-aware-link",
				".entity-result__title-text a",
				"a[href*='/in/']",
			}

			for _, sel := range clickSelectors {
				linkEl, err := el.Element(sel)
				if err != nil {
					continue
				}

				err = linkEl.Click(proto.InputMouseButtonLeft, 1)
				if err != nil {
					continue
				}

				time.Sleep(2 * time.Second)

				currentURL, _ := sm.browser.GetCurrentURL()
				if strings.Contains(currentURL, "/in/") {
					profileURL = sm.cleanProfileURL(currentURL)
					sm.log.Debug("Got URL by clicking", map[string]interface{}{
						"url": profileURL,
					})

					// Go back to search
					sm.browser.Navigate(originalURL)
					time.Sleep(2 * time.Second)

					// Re-fetch elements
					for _, selector := range selectors {
						elements, _ = sm.browser.GetElements(selector)
						if len(elements) > 0 {
							break
						}
					}
					break
				} else {
					// Go back if we navigated somewhere else
					sm.browser.Navigate(originalURL)
					time.Sleep(1 * time.Second)
				}
				break
			}
		}

		result.ProfileURL = profileURL

		// Skip if we got nothing useful
		if result.Name == "" && result.Title == "" {
			continue
		}

		// If name is hidden, use title as identifier
		if result.Name == "" || result.Name == "LinkedIn Member" {
			if result.Title != "" {
				result.Name = result.Title
			}
		}

		results = append(results, result)
		sm.log.Debug("Extracted profile", map[string]interface{}{
			"name":     result.Name,
			"title":    result.Title,
			"location": result.Location,
			"url":      result.ProfileURL,
		})
	}

	return results, nil
}

// cleanProfileURL cleans and normalizes a LinkedIn profile URL.
func (sm *SearchManager) cleanProfileURL(rawURL string) string {
	// Parse the URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	// Get just the path
	path := u.Path

	// Remove query parameters
	if idx := strings.Index(path, "?"); idx > 0 {
		path = path[:idx]
	}

	// Ensure it starts with /in/
	if !strings.HasPrefix(path, "/in/") {
		return ""
	}

	return "https://www.linkedin.com" + path
}

// goToNextPage attempts to navigate to the next search results page.
func (sm *SearchManager) goToNextPage() bool {
	// Look for next button
	nextSelectors := []string{
		"button[aria-label='Next']",
		".artdeco-pagination__button--next",
		"button.artdeco-pagination__button--next",
	}

	for _, selector := range nextSelectors {
		if sm.browser.HasElement(selector) {
			if err := sm.browser.Click(selector); err == nil {
				sm.stealth.PageDelay()
				return true
			}
		}
	}

	return false
}

// SaveResults saves search results to storage.
func (sm *SearchManager) SaveResults(results []*SearchResult) error {
	sm.log.Info("Saving search results", map[string]interface{}{
		"count": len(results),
	})

	for _, r := range results {
		profile := &storage.Profile{
			ID:            sm.generateProfileID(r.ProfileURL),
			ProfileURL:    r.ProfileURL,
			Name:          r.Name,
			Title:         r.Title,
			Location:      r.Location,
			ConnectionDeg: r.ConnectionDeg,
			IsConnected:   r.IsConnection,
		}

		if err := sm.storage.SaveProfile(profile); err != nil {
			sm.log.Warn("Failed to save profile", map[string]interface{}{
				"url":   r.ProfileURL,
				"error": err.Error(),
			})
		}
	}

	return nil
}

// generateProfileID generates a unique ID from profile URL.
func (sm *SearchManager) generateProfileID(profileURL string) string {
	// Extract username from URL
	parts := strings.Split(profileURL, "/in/")
	if len(parts) < 2 {
		return fmt.Sprintf("profile_%d", time.Now().UnixNano())
	}

	username := strings.TrimSuffix(parts[1], "/")
	return username
}

// FilterResults filters search results based on criteria.
func (sm *SearchManager) FilterResults(results []*SearchResult, skipExisting bool, skipConnections bool) []*SearchResult {
	var filtered []*SearchResult

	for _, r := range results {
		// Skip existing profiles
		if skipExisting && sm.storage.ProfileExists(r.ProfileURL) {
			sm.log.Debug("Skipping existing profile", map[string]interface{}{
				"url": r.ProfileURL,
			})
			continue
		}

		// Skip existing connections
		if skipConnections && r.IsConnection {
			sm.log.Debug("Skipping existing connection", map[string]interface{}{
				"url": r.ProfileURL,
			})
			continue
		}

		// Skip if connection already sent
		if sm.storage.WasConnectionSent(r.ProfileURL) {
			sm.log.Debug("Skipping - connection already sent", map[string]interface{}{
				"url": r.ProfileURL,
			})
			continue
		}

		filtered = append(filtered, r)
	}

	sm.log.Info("Filtered results", map[string]interface{}{
		"original": len(results),
		"filtered": len(filtered),
	})

	return filtered
}

// GetProfileDetails fetches detailed information about a profile.
func (sm *SearchManager) GetProfileDetails(profileURL string) (*SearchResult, error) {
	sm.log.Debug("Fetching profile details", map[string]interface{}{
		"url": profileURL,
	})

	if err := sm.browser.Navigate(profileURL); err != nil {
		return nil, err
	}

	sm.stealth.PageDelay()

	result := &SearchResult{
		ProfileURL: profileURL,
	}

	// Extract name
	if text, err := sm.browser.GetText("h1.text-heading-xlarge"); err == nil {
		result.Name = strings.TrimSpace(text)
	}

	// Extract headline/title
	if text, err := sm.browser.GetText(".text-body-medium.break-words"); err == nil {
		result.Title = strings.TrimSpace(text)
	}

	// Extract location
	if text, err := sm.browser.GetText(".text-body-small.inline.t-black--light.break-words"); err == nil {
		result.Location = strings.TrimSpace(text)
	}

	// Check connection status
	if sm.browser.HasElement("button[aria-label*='Message']") {
		result.IsConnection = true
		result.ConnectionDeg = "1st"
	}

	return result, nil
}
