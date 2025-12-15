// Package search provides LinkedIn search functionality.
package search

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/singh-yash129/link/internal/browser"
	"github.com/singh-yash129/link/internal/config"
	"github.com/singh-yash129/link/internal/logger"
	"github.com/singh-yash129/link/internal/stealth"
	"github.com/singh-yash129/link/internal/storage"
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
		"maxResults": filters.MaxResults,
	})

	// Navigate to search
	if err := sm.browser.Navigate(searchURL); err != nil {
		endOp(err)
		return nil, fmt.Errorf("failed to navigate to search: %w", err)
	}

	sm.stealth.PageDelay()

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

		// Wait for results to load
		if err := sm.browser.WaitForElement(".search-results-container", 10*time.Second); err != nil {
			sm.log.Warn("Search results not found, may be end of results", nil)
			break
		}

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

		allResults = append(allResults, results...)

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
func (sm *SearchManager) buildSearchURL(filters SearchFilters) string {
	baseURL := "https://www.linkedin.com/search/results/people/?"
	params := url.Values{}

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

	// Geographic region (would need mapping to LinkedIn geo IDs)
	if filters.Location != "" {
		params.Add("geoUrn", filters.Location)
	}

	// Company filter
	if filters.Company != "" {
		params.Add("company", filters.Company)
	}

	// Origin
	params.Add("origin", "GLOBAL_SEARCH_HEADER")

	return baseURL + params.Encode()
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

	// Get all result cards
	elements, err := sm.browser.GetElements(".entity-result")
	if err != nil {
		// Try alternative selector
		elements, err = sm.browser.GetElements(".reusable-search__result-container")
		if err != nil {
			return nil, err
		}
	}

	for _, el := range elements {
		result := &SearchResult{}

		// Extract profile URL
		linkEl, err := el.Element("a.app-aware-link")
		if err == nil {
			href, _ := linkEl.Attribute("href")
			if href != nil {
				result.ProfileURL = sm.cleanProfileURL(*href)
			}
		}

		// Skip if no valid profile URL
		if result.ProfileURL == "" {
			continue
		}

		// Extract name
		nameEl, err := el.Element(".entity-result__title-text a span[aria-hidden='true']")
		if err == nil {
			result.Name, _ = nameEl.Text()
			result.Name = strings.TrimSpace(result.Name)
		}

		// Extract title/headline
		titleEl, err := el.Element(".entity-result__primary-subtitle")
		if err == nil {
			result.Title, _ = titleEl.Text()
			result.Title = strings.TrimSpace(result.Title)
		}

		// Extract location
		locEl, err := el.Element(".entity-result__secondary-subtitle")
		if err == nil {
			result.Location, _ = locEl.Text()
			result.Location = strings.TrimSpace(result.Location)
		}

		// Extract connection degree
		degreeEl, err := el.Element(".entity-result__badge-text")
		if err == nil {
			degree, _ := degreeEl.Text()
			result.ConnectionDeg = strings.TrimSpace(degree)
			result.IsConnection = strings.Contains(degree, "1st")
		}

		if result.Name != "" {
			results = append(results, result)

			// Random hover for human-like behavior
			if sm.stealth.IsWithinSchedule() {
				sm.stealth.RandomHover(sm.browser.GetPage())
			}
		}
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
