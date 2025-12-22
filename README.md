# LinkedIn Automation Tool

A comprehensive Go-based LinkedIn automation tool demonstrating advanced browser automation, anti-detection techniques, and clean architecture. Built with the Rod browser automation library.

> ⚠️ **EDUCATIONAL PURPOSE ONLY** - This tool is designed exclusively for technical evaluation and educational purposes. Automating LinkedIn violates their Terms of Service and may result in account bans. Do not use in production.

## 🎯 Features

### Core Functionality
- **Authentication System** - Login with credentials, session persistence, security checkpoint detection
- **Search & Targeting** - Search by job title, company, location, keywords with pagination
- **Connection Requests** - Send personalized connection requests with note templates
- **Messaging System** - Automated follow-up messages with template support

### Anti-Detection Techniques (8+ Implemented)

| # | Technique | Description |
|---|-----------|-------------|
| 1 | **Human-like Mouse Movement** | Bézier curves with variable speed, overshoot, and micro-corrections |
| 2 | **Randomized Timing Patterns** | Variable delays between actions mimicking human cognitive processing |
| 3 | **Browser Fingerprint Masking** | WebDriver flag removal, plugin spoofing, navigator property masking |
| 4 | **Random Scrolling Behavior** | Variable speeds, natural acceleration, occasional scroll-back |
| 5 | **Realistic Typing Simulation** | Variable keystroke intervals, occasional typos with corrections |
| 6 | **Mouse Hovering & Movement** | Random hover events and natural cursor wandering |
| 7 | **Activity Scheduling** | Business hours operation, break patterns, work day simulation |
| 8 | **Rate Limiting & Throttling** | Connection quotas, message limits, batch cooldowns |
| 9 | **Canvas Fingerprint Noise** | Subtle modifications to prevent canvas fingerprinting |
| 10 | **WebGL Fingerprint Masking** | GPU/renderer information spoofing |

## 📁 Project Structure

```
link/
├── cmd/
│   └── linkedin/
│       └── main.go           # Application entry point
├── internal/
│   ├── auth/
│   │   └── auth.go           # LinkedIn authentication
│   ├── browser/
│   │   └── browser.go        # Browser management
│   ├── config/
│   │   └── config.go         # Configuration management
│   ├── connection/
│   │   └── connection.go     # Connection request handling
│   ├── logger/
│   │   └── logger.go         # Structured logging
│   ├── messaging/
│   │   └── messaging.go      # Messaging system
│   ├── search/
│   │   └── search.go         # Search functionality
│   ├── stealth/
│   │   └── stealth.go        # Anti-detection techniques
│   └── storage/
│       └── storage.go        # Data persistence (SQLite/JSON)
├── data/                      # Runtime data directory
├── logs/                      # Log files
├── config.yaml.example        # Configuration template
├── .env.example               # Environment variables template
├── go.mod                     # Go module definition
└── README.md                  # This file
```

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or higher
- Google Chrome or Chromium browser
- LinkedIn account (for testing only)

### Installation

```bash
# Clone the repository
git clone https://github.com/singh-yash129/linkedin-automation-agent.git
cd link

# Install dependencies
go mod download

# Build the application
go build -o linkedin-tool ./cmd/linkedin
```

### Configuration

1. **Copy configuration templates:**
```bash
cp config.yaml.example config.yaml
cp .env.example .env
```

2. **Edit `.env` with your credentials:**
```env
LINKEDIN_EMAIL=your_email@example.com
LINKEDIN_PASSWORD=your_password_here
```

3. **Customize `config.yaml`** for your targeting preferences.

### Running

```bash
# Run full automation (search → connect → message)
./linkedin-tool --mode=full

# Run specific modes
./linkedin-tool --mode=search   # Search only
./linkedin-tool --mode=connect  # Send connections only
./linkedin-tool --mode=message  # Send messages only
./linkedin-tool --mode=check    # Check stats only

# Additional options
./linkedin-tool --headless      # Run in headless mode
./linkedin-tool --debug         # Enable debug logging
./linkedin-tool --config=custom.yaml  # Use custom config
```

## 📋 Configuration Options

### LinkedIn Credentials

Set via environment variables (recommended for security):
```env
LINKEDIN_EMAIL=your_email@example.com
LINKEDIN_PASSWORD=your_password_here
```

### Search Targeting

```yaml
search:
  keywords: ["technology", "startup"]
  job_titles: ["Software Engineer", "Product Manager"]
  companies: ["Google", "Microsoft"]
  locations: ["San Francisco Bay Area"]
  max_results: 100
  filter_by_2nd_degree: true
```

### Connection Settings

```yaml
connection:
  daily_limit: 20
  hourly_limit: 5
  send_note: true
  note_templates:
    - "Hi {{.FirstName}}, I'd love to connect!"
```

### Activity Schedule

```yaml
schedule:
  enabled: true
  start_hour: 9
  end_hour: 18
  work_days_only: true
  timezone: "America/New_York"
```

## 🛡️ Anti-Detection Deep Dive

### 1. Human-like Mouse Movement

```go
// Uses cubic Bézier curves for natural trajectory
func (sm *StealthManager) generateBezierPath(startX, startY, endX, endY float64) []Point {
    // Generate control points with random perpendicular offset
    // Implements acceleration/deceleration curves
    // Adds micro-corrections for hand tremor simulation
}
```

### 2. Browser Fingerprint Masking

The tool injects JavaScript to:
- Remove `navigator.webdriver` flag
- Spoof `navigator.plugins` array
- Mask automation-related Chrome properties
- Override permission queries

### 3. Typing Simulation

```go
// Simulates realistic typing with:
// - Variable keystroke intervals based on WPM
// - Occasional typos with backspace corrections
// - Pauses at punctuation marks
// - Random "thinking" pauses
```

### 4. Rate Limiting

Multiple layers of protection:
- Hourly connection limits
- Daily connection limits
- Batch processing with cooldowns
- Randomized delays between actions

## 📊 Data Storage

### SQLite Schema (Default)

```sql
-- Profiles table
CREATE TABLE profiles (
    id INTEGER PRIMARY KEY,
    linkedin_url TEXT UNIQUE,
    first_name TEXT,
    last_name TEXT,
    title TEXT,
    company TEXT,
    status TEXT DEFAULT 'new'
);

-- Connection requests
CREATE TABLE connection_requests (
    id INTEGER PRIMARY KEY,
    profile_url TEXT,
    note TEXT,
    status TEXT DEFAULT 'sent',
    sent_at DATETIME
);

-- Messages
CREATE TABLE messages (
    id INTEGER PRIMARY KEY,
    profile_url TEXT,
    content TEXT,
    direction TEXT,
    type TEXT,
    sent_at DATETIME
);
```

### JSON Storage (Alternative)

Files stored in `./data/`:
- `profiles.json` - Collected profiles
- `requests.json` - Connection requests
- `messages.json` - Sent messages
- `cookies.json` - Session cookies

## 🔧 Development

### Building from Source

```bash
# Development build
go build -o linkedin-tool ./cmd/linkedin

# Production build with optimizations
go build -ldflags="-s -w" -o linkedin-tool ./cmd/linkedin
```

### Running Tests

```bash
go test ./...
```

### Code Architecture

The application follows clean architecture principles:

- **Separation of Concerns**: Each package handles a specific domain
- **Dependency Injection**: Components receive dependencies through constructors
- **Interface Abstraction**: Storage uses interfaces for SQLite/JSON flexibility
- **Error Handling**: Comprehensive error types with context

## 📝 Template Variables

Available in note and message templates:

| Variable | Description |
|----------|-------------|
| `{{.FirstName}}` | Contact's first name |
| `{{.LastName}}` | Contact's last name |
| `{{.Name}}` | Full name |
| `{{.Title}}` | Job title |
| `{{.Company}}` | Current company |
| `{{.Location}}` | Location |
| `{{.Industry}}` | Industry |

## ⚠️ Important Disclaimers

1. **Terms of Service**: Using this tool on LinkedIn violates their ToS
2. **Account Risk**: Your account may be banned permanently
3. **Legal**: Unauthorized automation may have legal implications
4. **Educational Only**: This code is for learning purposes only


## 📄 License

This project is for educational purposes only. See LICENSE file for details.

**Remember**: This tool demonstrates automation concepts and should never be used on production LinkedIn accounts.
