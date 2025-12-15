// Package stealth implements anti-detection techniques for browser automation.
// This package implements sophisticated techniques to avoid bot detection including:
// - Human-like mouse movements using Bézier curves
// - Realistic typing simulation with typos and corrections
// - Random scrolling behavior
// - Browser fingerprint masking
// - Activity scheduling
package stealth

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/singh-yash129/link/internal/config"
	"github.com/singh-yash129/link/internal/logger"
)

// StealthManager handles all anti-detection and human simulation operations.
type StealthManager struct {
	config     *config.Config
	log        *logger.Logger
	lastAction time.Time
	lastBreak  time.Time
}

// NewStealthManager creates a new stealth manager instance.
func NewStealthManager(cfg *config.Config, log *logger.Logger) *StealthManager {
	return &StealthManager{
		config:     cfg,
		log:        log,
		lastAction: time.Now(),
		lastBreak:  time.Now(),
	}
}

// Point represents a 2D coordinate.
type Point struct {
	X float64
	Y float64
}

// ============================================================================
// TECHNIQUE 1: Human-like Mouse Movement (Bézier Curves)
// ============================================================================

// MoveMouse simulates human-like mouse movement from current position to target.
// Uses Bézier curves with variable speed, natural overshoot, and micro-corrections.
func (sm *StealthManager) MoveMouse(page *rod.Page, targetX, targetY float64) error {
	if !sm.config.Stealth.EnableMouseSimulation {
		// Direct movement if simulation is disabled
		return page.Mouse.MoveTo(proto.Point{X: targetX, Y: targetY})
	}

	sm.log.Debug("Moving mouse to target position", map[string]interface{}{
		"targetX": targetX,
		"targetY": targetY,
	})

	// Get current mouse position (estimate from page center if not available)
	currentX, currentY := sm.estimateCurrentPosition(page)

	// Generate Bézier curve path with control points
	path := sm.generateBezierPath(currentX, currentY, targetX, targetY)

	// Add micro-corrections and jitter to the path
	path = sm.addMicroCorrections(path)

	// Execute movement along the path with variable speed
	for i, point := range path {
		// Calculate speed variation (slower at start and end)
		progress := float64(i) / float64(len(path))
		delay := sm.calculateMovementDelay(progress)

		if err := page.Mouse.MoveTo(proto.Point{X: point.X, Y: point.Y}); err != nil {
			return fmt.Errorf("failed to move mouse: %w", err)
		}
		time.Sleep(delay)
	}

	// Simulate occasional overshoot and correction
	if sm.shouldOvershoot() {
		sm.log.Debug("Simulating mouse overshoot", nil)
		overshoot := sm.calculateOvershoot(targetX, targetY)
		if err := page.Mouse.MoveTo(proto.Point{X: overshoot.X, Y: overshoot.Y}); err != nil {
			return err
		}
		time.Sleep(sm.randomDuration(50, 150))
		// Correct back to target
		if err := page.Mouse.MoveTo(proto.Point{X: targetX, Y: targetY}); err != nil {
			return err
		}
	}

	return nil
}

// generateBezierPath creates a curved path using cubic Bézier curves.
func (sm *StealthManager) generateBezierPath(startX, startY, endX, endY float64) []Point {
	// Calculate distance and number of steps
	distance := math.Sqrt(math.Pow(endX-startX, 2) + math.Pow(endY-startY, 2))
	steps := int(math.Max(20, distance/10))

	// Generate random control points for natural curve
	// Control points are offset perpendicular to the direct line
	midX := (startX + endX) / 2
	midY := (startY + endY) / 2

	// Random perpendicular offset for control points
	offsetRange := distance * 0.2
	offset1 := sm.randomFloat(-offsetRange, offsetRange)
	offset2 := sm.randomFloat(-offsetRange, offsetRange)

	// Calculate perpendicular direction
	dx := endX - startX
	dy := endY - startY
	length := math.Sqrt(dx*dx + dy*dy)
	if length == 0 {
		length = 1
	}
	perpX := -dy / length
	perpY := dx / length

	// Control points
	cp1 := Point{
		X: midX + perpX*offset1 - dx*0.2,
		Y: midY + perpY*offset1 - dy*0.2,
	}
	cp2 := Point{
		X: midX + perpX*offset2 + dx*0.2,
		Y: midY + perpY*offset2 + dy*0.2,
	}

	// Generate path points using cubic Bézier formula
	path := make([]Point, steps)
	for i := 0; i < steps; i++ {
		t := float64(i) / float64(steps-1)
		path[i] = sm.cubicBezier(
			Point{X: startX, Y: startY},
			cp1,
			cp2,
			Point{X: endX, Y: endY},
			t,
		)
	}

	return path
}

// cubicBezier calculates a point on a cubic Bézier curve at parameter t.
func (sm *StealthManager) cubicBezier(p0, p1, p2, p3 Point, t float64) Point {
	u := 1 - t
	tt := t * t
	uu := u * u
	uuu := uu * u
	ttt := tt * t

	return Point{
		X: uuu*p0.X + 3*uu*t*p1.X + 3*u*tt*p2.X + ttt*p3.X,
		Y: uuu*p0.Y + 3*uu*t*p1.Y + 3*u*tt*p2.Y + ttt*p3.Y,
	}
}

// addMicroCorrections adds small random jitter to simulate hand tremor.
func (sm *StealthManager) addMicroCorrections(path []Point) []Point {
	for i := range path {
		// Skip first and last points for accuracy
		if i == 0 || i == len(path)-1 {
			continue
		}
		// Add small random offset
		path[i].X += sm.randomFloat(-1, 1)
		path[i].Y += sm.randomFloat(-1, 1)
	}
	return path
}

// calculateMovementDelay returns delay for movement, slower at start/end.
func (sm *StealthManager) calculateMovementDelay(progress float64) time.Duration {
	baseDelay := 5.0 // milliseconds
	speedFactor := sm.config.Stealth.MouseSpeed
	if speedFactor == 0 {
		speedFactor = 1.0
	}

	// Ease-in-out curve: slower at start and end
	easing := math.Sin(progress * math.Pi)
	delay := baseDelay / (0.5 + easing*speedFactor)
	return time.Duration(delay) * time.Millisecond
}

// shouldOvershoot determines if mouse should overshoot target.
func (sm *StealthManager) shouldOvershoot() bool {
	return sm.randomFloat(0, 1) < 0.15 // 15% chance of overshoot
}

// calculateOvershoot returns an overshoot position past the target.
func (sm *StealthManager) calculateOvershoot(targetX, targetY float64) Point {
	overshootDist := sm.randomFloat(3, 8)
	angle := sm.randomFloat(0, 2*math.Pi)
	return Point{
		X: targetX + math.Cos(angle)*overshootDist,
		Y: targetY + math.Sin(angle)*overshootDist,
	}
}

// estimateCurrentPosition estimates current mouse position.
func (sm *StealthManager) estimateCurrentPosition(page *rod.Page) (float64, float64) {
	// Get viewport size from evaluation
	result, err := page.Eval(`({width: window.innerWidth, height: window.innerHeight})`)
	if err != nil {
		return 500, 400 // Default fallback
	}

	width := result.Value.Get("width").Num()
	height := result.Value.Get("height").Num()

	// Start from a random position in the viewport
	x := sm.randomFloat(100, width-100)
	y := sm.randomFloat(100, height-100)
	return x, y
}

// ============================================================================
// TECHNIQUE 2: Randomized Timing Patterns
// ============================================================================

// RandomDelay adds a randomized delay between actions.
func (sm *StealthManager) RandomDelay() {
	delay := sm.randomDuration(
		sm.config.RateLimits.MinActionDelay,
		sm.config.RateLimits.MaxActionDelay,
	)
	sm.log.Debug("Adding random delay", map[string]interface{}{
		"delay_ms": delay.Milliseconds(),
	})
	time.Sleep(delay)
	sm.lastAction = time.Now()
}

// PageDelay adds a delay appropriate for page navigation.
func (sm *StealthManager) PageDelay() {
	delay := sm.randomDuration(
		sm.config.RateLimits.MinPageDelay,
		sm.config.RateLimits.MaxPageDelay,
	)
	sm.log.Debug("Adding page navigation delay", map[string]interface{}{
		"delay_ms": delay.Milliseconds(),
	})
	time.Sleep(delay)
}

// ThinkDelay simulates human "thinking" time before an action.
func (sm *StealthManager) ThinkDelay() {
	// Variable think time: 500ms to 3000ms
	delay := sm.randomDuration(500, 3000)
	sm.log.Debug("Simulating think time", map[string]interface{}{
		"delay_ms": delay.Milliseconds(),
	})
	time.Sleep(delay)
}

// ============================================================================
// TECHNIQUE 3: Browser Fingerprint Masking
// ============================================================================

// ApplyFingerprintMasking applies browser fingerprint masking scripts.
func (sm *StealthManager) ApplyFingerprintMasking(page *rod.Page) error {
	if !sm.config.Stealth.EnableWebDriverMasking {
		return nil
	}

	sm.log.Info("Applying browser fingerprint masking", nil)

	// Comprehensive anti-detection JavaScript
	script := `
		// Remove webdriver flag
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined,
		});

		// Override navigator.plugins to look more realistic
		Object.defineProperty(navigator, 'plugins', {
			get: () => {
				const plugins = [
					{ name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer' },
					{ name: 'Chrome PDF Viewer', filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai' },
					{ name: 'Native Client', filename: 'internal-nacl-plugin' },
				];
				const pluginArray = Object.create(PluginArray.prototype);
				plugins.forEach((p, i) => {
					const plugin = Object.create(Plugin.prototype);
					Object.defineProperty(plugin, 'name', { get: () => p.name });
					Object.defineProperty(plugin, 'filename', { get: () => p.filename });
					Object.defineProperty(plugin, 'description', { get: () => p.name });
					Object.defineProperty(plugin, 'length', { get: () => 1 });
					pluginArray[i] = plugin;
				});
				Object.defineProperty(pluginArray, 'length', { get: () => plugins.length });
				return pluginArray;
			},
		});

		// Override languages
		Object.defineProperty(navigator, 'languages', {
			get: () => ['en-US', 'en'],
		});

		// Override platform
		Object.defineProperty(navigator, 'platform', {
			get: () => 'Win32',
		});

		// Override hardwareConcurrency
		Object.defineProperty(navigator, 'hardwareConcurrency', {
			get: () => 8,
		});

		// Override deviceMemory
		Object.defineProperty(navigator, 'deviceMemory', {
			get: () => 8,
		});

		// Override connection info
		if (navigator.connection) {
			Object.defineProperty(navigator.connection, 'rtt', {
				get: () => 50,
			});
		}

		// Hide automation-related Chrome properties
		window.chrome = {
			runtime: {},
			loadTimes: function() {},
			csi: function() {},
			app: {},
		};

		// Override permissions query
		const originalQuery = window.navigator.permissions.query;
		window.navigator.permissions.query = (parameters) => {
			if (parameters.name === 'notifications') {
				return Promise.resolve({ state: Notification.permission });
			}
			return originalQuery(parameters);
		};

		// Prevent detection via toString
		const originalToString = Function.prototype.toString;
		Function.prototype.toString = function() {
			if (this === navigator.permissions.query) {
				return 'function query() { [native code] }';
			}
			return originalToString.call(this);
		};
	`

	_, err := page.Eval(script)
	if err != nil {
		return fmt.Errorf("failed to apply fingerprint masking: %w", err)
	}

	// Apply canvas fingerprint noise if enabled
	if sm.config.Stealth.EnableCanvasNoise {
		if err := sm.applyCanvasNoise(page); err != nil {
			sm.log.Warn("Failed to apply canvas noise", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Apply WebGL noise if enabled
	if sm.config.Stealth.EnableWebGLNoise {
		if err := sm.applyWebGLNoise(page); err != nil {
			sm.log.Warn("Failed to apply WebGL noise", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	return nil
}

// applyCanvasNoise adds slight noise to canvas fingerprinting.
func (sm *StealthManager) applyCanvasNoise(page *rod.Page) error {
	script := `
		const originalGetContext = HTMLCanvasElement.prototype.getContext;
		HTMLCanvasElement.prototype.getContext = function(type, attributes) {
			const context = originalGetContext.call(this, type, attributes);
			if (type === '2d' && context) {
				const originalGetImageData = context.getImageData;
				context.getImageData = function(x, y, w, h) {
					const imageData = originalGetImageData.call(this, x, y, w, h);
					// Add subtle noise to prevent fingerprinting
					for (let i = 0; i < imageData.data.length; i += 4) {
						// Only modify some pixels slightly
						if (Math.random() < 0.01) {
							imageData.data[i] = Math.max(0, Math.min(255, imageData.data[i] + (Math.random() * 2 - 1)));
						}
					}
					return imageData;
				};
			}
			return context;
		};
	`
	_, err := page.Eval(script)
	return err
}

// applyWebGLNoise adds noise to WebGL fingerprinting.
func (sm *StealthManager) applyWebGLNoise(page *rod.Page) error {
	script := `
		const getParameterProxyHandler = {
			apply: function(target, thisArg, argumentsList) {
				const param = argumentsList[0];
				const result = target.apply(thisArg, argumentsList);
				// Slightly modify certain parameters
				if (param === 37445) { // UNMASKED_VENDOR_WEBGL
					return 'Google Inc. (NVIDIA)';
				}
				if (param === 37446) { // UNMASKED_RENDERER_WEBGL
					return 'ANGLE (NVIDIA, NVIDIA GeForce GTX 1080 Direct3D11 vs_5_0 ps_5_0)';
				}
				return result;
			}
		};

		const getExtensionProxyHandler = {
			apply: function(target, thisArg, argumentsList) {
				return target.apply(thisArg, argumentsList);
			}
		};

		// Proxy WebGLRenderingContext
		if (typeof WebGLRenderingContext !== 'undefined') {
			WebGLRenderingContext.prototype.getParameter = new Proxy(
				WebGLRenderingContext.prototype.getParameter, 
				getParameterProxyHandler
			);
		}

		// Proxy WebGL2RenderingContext
		if (typeof WebGL2RenderingContext !== 'undefined') {
			WebGL2RenderingContext.prototype.getParameter = new Proxy(
				WebGL2RenderingContext.prototype.getParameter, 
				getParameterProxyHandler
			);
		}
	`
	_, err := page.Eval(script)
	return err
}

// ============================================================================
// TECHNIQUE 4: Random Scrolling Behavior
// ============================================================================

// HumanScroll performs natural-looking scrolling behavior.
func (sm *StealthManager) HumanScroll(page *rod.Page, targetY float64) error {
	if !sm.config.Stealth.EnableScrollSimulation {
		return page.Mouse.Scroll(0, targetY, 1)
	}

	sm.log.Debug("Performing human-like scroll", map[string]interface{}{
		"targetY": targetY,
	})

	// Get current scroll position
	currentY, err := page.Eval(`window.scrollY`)
	if err != nil {
		return fmt.Errorf("failed to get scroll position: %w", err)
	}

	current := currentY.Value.Num()
	// Calculate scroll distance
	distance := targetY - current

	if math.Abs(distance) < 10 {
		return nil // Already at target
	}

	// Determine number of scroll steps
	steps := int(math.Max(5, math.Abs(distance)/100))

	// Add variation to scroll behavior
	variation := sm.config.Stealth.ScrollVariation

	for i := 0; i < steps; i++ {
		progress := float64(i+1) / float64(steps)

		// Ease-out scrolling (fast start, slow end)
		easing := 1 - math.Pow(1-progress, 3)

		// Calculate next position with variation
		nextY := current + (distance * easing)

		scrollAmount := (nextY - current) * (1 + sm.randomFloat(-variation, variation))

		if err := page.Mouse.Scroll(0, scrollAmount, 1); err != nil {
			return err
		}

		// Variable delay between scroll steps
		delay := sm.randomDuration(30, 100)
		time.Sleep(delay)

		// Occasionally pause mid-scroll (simulating reading)
		if sm.randomFloat(0, 1) < 0.1 {
			time.Sleep(sm.randomDuration(200, 800))
		}
	}

	// Occasionally scroll back slightly (natural behavior)
	if sm.randomFloat(0, 1) < 0.2 {
		scrollBack := sm.randomFloat(10, 50)
		if err := page.Mouse.Scroll(0, -scrollBack, 1); err != nil {
			return err
		}
		time.Sleep(sm.randomDuration(100, 300))
	}

	return nil
}

// RandomScroll performs random scrolling to simulate page exploration.
func (sm *StealthManager) RandomScroll(page *rod.Page) error {
	pageHeight, err := page.Eval(`document.body.scrollHeight`)
	if err != nil {
		return err
	}

	height := pageHeight.Value.Num()
	scrollAmount := sm.randomFloat(100, height*0.3)
	direction := 1.0
	if sm.randomFloat(0, 1) < 0.3 {
		direction = -1.0 // Occasionally scroll up
	}

	return sm.HumanScroll(page, scrollAmount*direction)
}

// ============================================================================
// TECHNIQUE 5: Realistic Typing Simulation
// ============================================================================

// HumanType simulates realistic human typing with variable speed and typos.
func (sm *StealthManager) HumanType(page *rod.Page, text string) error {
	if !sm.config.Stealth.EnableTypingSimulation {
		// Convert string to keys
		keys := make([]input.Key, len(text))
		for i, c := range text {
			keys[i] = input.Key(c)
		}
		return page.Keyboard.Type(keys...)
	}

	sm.log.Debug("Simulating human typing", map[string]interface{}{
		"textLength": len(text),
	})

	// Calculate base delay from WPM
	// Average word is 5 characters, so chars per minute = WPM * 5
	wpm := sm.config.Stealth.TypingSpeedWPM
	if wpm == 0 {
		wpm = 60
	}
	// Delay per character in milliseconds
	baseDelay := 60000.0 / (float64(wpm) * 5)

	for i, char := range text {
		// Variable typing speed
		delay := baseDelay * sm.randomFloat(0.7, 1.5)

		// Simulate occasional typo
		if sm.config.Stealth.TypoFrequency > 0 && sm.randomFloat(0, 1) < sm.config.Stealth.TypoFrequency {
			wrongChar := sm.getRandomChar()
			// Type a wrong character
			if err := page.Keyboard.Type(input.Key(wrongChar)); err != nil {
				return err
			}
			time.Sleep(time.Duration(delay*2) * time.Millisecond)
			// Pause as if realizing mistake
			time.Sleep(sm.randomDuration(100, 300))
			// Delete the wrong character
			if err := page.Keyboard.Press(input.Backspace); err != nil {
				return err
			}
			time.Sleep(sm.randomDuration(50, 150))
		}

		// Type the correct character
		if err := page.Keyboard.Type(input.Key(char)); err != nil {
			return err
		}
		time.Sleep(time.Duration(delay) * time.Millisecond)

		// Occasional longer pauses (thinking)
		if i > 0 && i%20 == 0 && sm.randomFloat(0, 1) < 0.3 {
			time.Sleep(sm.randomDuration(300, 800))
		}

		// Pause at punctuation
		if char == '.' || char == ',' || char == '!' || char == '?' {
			time.Sleep(sm.randomDuration(100, 400))
		}
	}

	return nil
}

// getRandomChar returns a random character for typo simulation.
func (sm *StealthManager) getRandomChar() rune {
	chars := "abcdefghijklmnopqrstuvwxyz"
	idx := sm.randomInt(0, len(chars)-1)
	return rune(chars[idx])
}

// ============================================================================
// TECHNIQUE 6: Mouse Hovering & Movement
// ============================================================================

// HoverElement hovers over an element naturally.
func (sm *StealthManager) HoverElement(page *rod.Page, selector string) error {
	if !sm.config.Stealth.EnableHoverEvents {
		return nil
	}

	el, err := page.Element(selector)
	if err != nil {
		return err
	}

	box, err := el.Shape()
	if err != nil {
		return err
	}

	if len(box.Quads) == 0 {
		return fmt.Errorf("element has no shape")
	}

	// Get center of element
	quad := box.Quads[0]
	centerX := (quad[0] + quad[2] + quad[4] + quad[6]) / 4
	centerY := (quad[1] + quad[3] + quad[5] + quad[7]) / 4

	// Move to element with human-like movement
	if err := sm.MoveMouse(page, centerX, centerY); err != nil {
		return err
	}

	// Hover for a random duration
	hoverTime := sm.randomDuration(200, 800)
	time.Sleep(hoverTime)

	return nil
}

// RandomHover performs random hovering on page elements.
func (sm *StealthManager) RandomHover(page *rod.Page) error {
	if !sm.config.Stealth.EnableHoverEvents {
		return nil
	}

	// Common elements to hover over randomly
	selectors := []string{
		"a",
		"button",
		"img",
		"[role='button']",
		".feed-shared-actor__name",
		".feed-shared-text",
	}

	// Pick a random selector
	selector := selectors[sm.randomInt(0, len(selectors)-1)]

	elements, err := page.Elements(selector)
	if err != nil || len(elements) == 0 {
		return nil // Silently fail, not critical
	}

	// Pick a random element
	el := elements[sm.randomInt(0, len(elements)-1)]

	box, err := el.Shape()
	if err != nil {
		return nil
	}

	if len(box.Quads) == 0 {
		return nil
	}

	quad := box.Quads[0]
	centerX := (quad[0] + quad[2] + quad[4] + quad[6]) / 4
	centerY := (quad[1] + quad[3] + quad[5] + quad[7]) / 4

	return sm.MoveMouse(page, centerX, centerY)
}

// ============================================================================
// TECHNIQUE 7: Activity Scheduling
// ============================================================================

// IsWithinSchedule checks if current time is within allowed activity hours.
func (sm *StealthManager) IsWithinSchedule() bool {
	if !sm.config.Schedule.Enabled {
		return true
	}

	loc, err := time.LoadLocation(sm.config.Schedule.Timezone)
	if err != nil {
		loc = time.Local
	}

	now := time.Now().In(loc)
	hour := now.Hour()

	// Check work hours
	if hour < sm.config.Schedule.StartHour || hour >= sm.config.Schedule.EndHour {
		return false
	}

	// Check work days
	if sm.config.Schedule.WorkDaysOnly {
		weekday := now.Weekday()
		if weekday == time.Saturday || weekday == time.Sunday {
			return false
		}
	}

	return true
}

// WaitForSchedule waits until the activity schedule allows operations.
func (sm *StealthManager) WaitForSchedule() {
	for !sm.IsWithinSchedule() {
		sm.log.Info("Outside scheduled activity hours, waiting...", map[string]interface{}{
			"timezone":  sm.config.Schedule.Timezone,
			"startHour": sm.config.Schedule.StartHour,
			"endHour":   sm.config.Schedule.EndHour,
		})
		time.Sleep(5 * time.Minute)
	}
}

// ShouldTakeBreak determines if it's time for a simulated break.
func (sm *StealthManager) ShouldTakeBreak() bool {
	if !sm.config.Schedule.EnableBreaks {
		return false
	}

	elapsed := time.Since(sm.lastBreak)
	breakFreq := time.Duration(sm.config.Schedule.BreakFrequency) * time.Minute
	return elapsed >= breakFreq
}

// TakeBreak simulates a human taking a break.
func (sm *StealthManager) TakeBreak() {
	breakDuration := sm.randomDuration(
		sm.config.Schedule.MinBreakMinutes*60*1000,
		sm.config.Schedule.MaxBreakMinutes*60*1000,
	)
	sm.log.Info("Taking a simulated break", map[string]interface{}{
		"duration_minutes": breakDuration.Minutes(),
	})
	time.Sleep(breakDuration)
	sm.lastBreak = time.Now()
}

// ============================================================================
// TECHNIQUE 8: Rate Limiting & Throttling
// ============================================================================

// RateLimiter tracks and enforces rate limits.
type RateLimiter struct {
	hourlyCount   int
	dailyCount    int
	lastHourReset time.Time
	lastDayReset  time.Time
	batchCount    int
	config        *config.Config
	log           *logger.Logger
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(cfg *config.Config, log *logger.Logger) *RateLimiter {
	now := time.Now()
	return &RateLimiter{
		hourlyCount:   0,
		dailyCount:    0,
		lastHourReset: now,
		lastDayReset:  now,
		batchCount:    0,
		config:        cfg,
		log:           log,
	}
}

// CanProceed checks if an action can proceed based on rate limits.
func (rl *RateLimiter) CanProceed(actionType string) bool {
	rl.resetIfNeeded()

	switch actionType {
	case "connection":
		if rl.hourlyCount >= rl.config.Connection.HourlyLimit {
			rl.log.Warn("Hourly connection limit reached", map[string]interface{}{
				"limit": rl.config.Connection.HourlyLimit,
				"count": rl.hourlyCount,
			})
			return false
		}
		if rl.dailyCount >= rl.config.Connection.DailyLimit {
			rl.log.Warn("Daily connection limit reached", map[string]interface{}{
				"limit": rl.config.Connection.DailyLimit,
				"count": rl.dailyCount,
			})
			return false
		}
	case "message":
		if rl.dailyCount >= rl.config.Messaging.DailyMessageLimit {
			rl.log.Warn("Daily message limit reached", map[string]interface{}{
				"limit": rl.config.Messaging.DailyMessageLimit,
				"count": rl.dailyCount,
			})
			return false
		}
	}

	return true
}

// RecordAction records that an action was taken.
func (rl *RateLimiter) RecordAction(actionType string) {
	rl.hourlyCount++
	rl.dailyCount++
	rl.batchCount++

	rl.log.Debug("Action recorded", map[string]interface{}{
		"type":        actionType,
		"hourlyCount": rl.hourlyCount,
		"dailyCount":  rl.dailyCount,
		"batchCount":  rl.batchCount,
	})

	// Check if batch cooldown is needed
	if rl.batchCount >= rl.config.RateLimits.BatchSize {
		rl.log.Info("Batch complete, taking cooldown", map[string]interface{}{
			"batchSize":       rl.config.RateLimits.BatchSize,
			"cooldownMinutes": rl.config.RateLimits.CooldownAfterBatch,
		})
		time.Sleep(time.Duration(rl.config.RateLimits.CooldownAfterBatch) * time.Minute)
		rl.batchCount = 0
	}
}

// resetIfNeeded resets counters if time periods have elapsed.
func (rl *RateLimiter) resetIfNeeded() {
	now := time.Now()

	// Reset hourly counter
	if now.Sub(rl.lastHourReset) >= time.Hour {
		rl.hourlyCount = 0
		rl.lastHourReset = now
		rl.log.Debug("Hourly counter reset", nil)
	}

	// Reset daily counter
	if now.Sub(rl.lastDayReset) >= 24*time.Hour {
		rl.dailyCount = 0
		rl.lastDayReset = now
		rl.log.Debug("Daily counter reset", nil)
	}
}

// GetRemainingHourly returns remaining hourly allowance.
func (rl *RateLimiter) GetRemainingHourly() int {
	rl.resetIfNeeded()
	return rl.config.Connection.HourlyLimit - rl.hourlyCount
}

// GetRemainingDaily returns remaining daily allowance.
func (rl *RateLimiter) GetRemainingDaily() int {
	rl.resetIfNeeded()
	return rl.config.Connection.DailyLimit - rl.dailyCount
}

// ============================================================================
// Helper Functions
// ============================================================================

// randomFloat returns a random float64 between min and max.
func (sm *StealthManager) randomFloat(min, max float64) float64 {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return min
	}
	return min + (max-min)*float64(n.Int64())/1000000.0
}

// randomInt returns a random int between min and max (inclusive).
func (sm *StealthManager) randomInt(min, max int) int {
	if min >= max {
		return min
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	if err != nil {
		return min
	}
	return min + int(n.Int64())
}

// randomDuration returns a random duration between min and max milliseconds.
func (sm *StealthManager) randomDuration(minMs, maxMs int) time.Duration {
	ms := sm.randomInt(minMs, maxMs)
	return time.Duration(ms) * time.Millisecond
}

// ClickElement clicks an element with human-like behavior.
func (sm *StealthManager) ClickElement(page *rod.Page, selector string) error {
	el, err := page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %s", selector)
	}

	// Get element position
	box, err := el.Shape()
	if err != nil {
		return fmt.Errorf("failed to get element shape: %w", err)
	}

	if len(box.Quads) == 0 {
		return fmt.Errorf("element has no visible shape")
	}

	// Calculate click position with slight randomization within element
	quad := box.Quads[0]
	centerX := (quad[0] + quad[2] + quad[4] + quad[6]) / 4
	centerY := (quad[1] + quad[3] + quad[5] + quad[7]) / 4

	// Add slight offset from center for natural clicking
	width := math.Abs(quad[2] - quad[0])
	height := math.Abs(quad[5] - quad[1])
	offsetX := sm.randomFloat(-width*0.2, width*0.2)
	offsetY := sm.randomFloat(-height*0.2, height*0.2)

	// Move to element
	if err := sm.MoveMouse(page, centerX+offsetX, centerY+offsetY); err != nil {
		return err
	}

	// Small delay before click
	time.Sleep(sm.randomDuration(50, 150))

	// Click
	if err := page.Mouse.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("failed to click: %w", err)
	}

	return nil
}

// SetupBrowserStealth configures browser with stealth settings.
func (sm *StealthManager) SetupBrowserStealth(browser *rod.Browser) error {
	sm.log.Info("Setting up browser stealth features", nil)

	// Get the first page or create one
	pages, err := browser.Pages()
	if err != nil {
		return err
	}

	var page *rod.Page
	if len(pages) > 0 {
		page = pages[0]
	} else {
		page, err = browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
		if err != nil {
			return err
		}
	}

	// Apply fingerprint masking
	return sm.ApplyFingerprintMasking(page)
}
