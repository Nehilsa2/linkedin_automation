<div align="center">

# 🔗 LinkedIn Automation Tool

```
╔══════════════════════════════════════════════════════════════╗
║                                                              ║
║   ██╗     ██╗███╗   ██╗██╗  ██╗███████╗██████╗ ██████╗      ║
║   ██║     ██║████╗  ██║██║ ██╔╝██╔════╝██╔══██╗██╔══██╗     ║
║   ██║     ██║██╔██╗ ██║█████╔╝ █████╗  ██║  ██║██████╔╝     ║
║   ██║     ██║██║╚██╗██║██╔═██╗ ██╔══╝  ██║  ██║██╔══██╗     ║
║   ███████╗██║██║ ╚████║██║  ██╗███████╗██████╔╝██████╔╝     ║
║   ╚══════╝╚═╝╚═╝  ╚═══╝╚═╝  ╚═╝╚══════╝╚═════╝ ╚═════╝      ║
║                                                              ║
║         🤖 Automated Networking Made Easy 🤖                 ║
╚══════════════════════════════════════════════════════════════╝
```

[![Go Version](https://img.shields.io/badge/Go-1.25.5+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Educational-red?style=for-the-badge)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Active-success?style=for-the-badge)]()

**A sophisticated LinkedIn automation tool written in Go that helps you automate networking activities while maintaining a human-like behavior pattern to avoid detection.**

✨ *Search profiles • Send connections • Follow-up messages • All automated* ✨

</div>

---

<div align="center">

## ⚠️ DISCLAIMER ⚠️

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  ⚠️  This tool is for EDUCATIONAL PURPOSES ONLY  ⚠️        │
│                                                             │
│  Using automation tools on LinkedIn may violate            │
│  LinkedIn's Terms of Service.                              │
│                                                             │
│  Use at your own risk. The authors are not responsible      │
│  for any account restrictions or bans.                     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

</div>

---



## ✨ Features

<table>
<tr>
<td width="50%">

### 🎯 Core Features

- **🔍 Profile Search**  
  Automatically search for people and companies based on keywords

- **🔗 Connection Requests**  
  Send personalized connection requests with custom notes

- **📬 Follow-up Messaging**  
  Automatically send follow-up messages to accepted connections

- **📊 Statistics Tracking**  
  Track daily activity, connection acceptance rates, and message statistics

</td>
<td width="50%">

### 🛡️ Stealth & Safety

- **🛡️ Stealth Mode**  
  Human-like behavior patterns:
  - ✨ Natural typing speeds
  - 🖱️ Random delays and mouse movements
  - 🌐 Organic browsing between actions
  - 🔒 Browser fingerprint masking

- **⏱️ Rate Limiting**  
  Configurable rate limits to avoid triggering LinkedIn's anti-bot measures

- **📅 Scheduling**  
  Work hour enforcement and break management

- **💾 Persistent Storage**  
  SQLite database for tracking connections, messages, and search results

- **🔄 Workflow Resumption**  
  Resume paused workflows after interruptions

- **🧪 Dry Run Mode**  
  Test workflows without performing actual actions

</td>
</tr>
</table>


## 📋 Prerequisites

- **Go 1.25.5** or higher
- **Google Chrome** browser installed (default path: `C://Program Files//Google//Chrome//Application//chrome.exe`)
- **LinkedIn account** credentials

## 🚀 Installation

<div align="center">

### Quick Start Guide

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│  1️⃣  Clone Repository                                  │
│  2️⃣  Install Dependencies                              │
│  3️⃣  Configure Environment                             │
│  4️⃣  Build & Run                                        │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

</div>

### Step 1: Clone the Repository 📥

```bash
git clone https://github.com/Nehilsa2/linkedin_automation.git
cd linkedin_automation
```

### Step 2: Install Dependencies 📦

```bash
go mod download
```

### Step 3: Configure Environment ⚙️

Create a `.env` file in the project root:

```env
LINKEDIN_EMAIL=your_email@example.com
LINKEDIN_PASSWORD=your_password
```

### Step 4: Build the Project 🔨

```bash
go build -o linkedin_automation.exe
```

<div align="center">

**🎉 You're all set! Ready to automate! 🎉**

</div>

## ⚙️ Configuration

<div align="center">

### 🎛️ Customize Your Setup

</div>

### 🔐 Environment Variables

<div align="center">

**Create a `.env` file with your LinkedIn credentials:**

</div>

| Variable | Description | Example |
|:---------|:------------|:--------|
| `LINKEDIN_EMAIL` | Your LinkedIn email address | `user@example.com` |
| `LINKEDIN_PASSWORD` | Your LinkedIn password | `your_secure_password` |

### 🚦 Rate Limiting Configuration

<div align="center">

**Edit `rate_config.json` to adjust rate limits and delays**

</div>

```json
{
  "safety_level": "conservative",
  "connection_daily_limit": 10,
  "connection_hourly_limit": 3,
  "connection_delay_min_sec": 30,
  "connection_delay_max_sec": 90,
  "message_daily_limit": 3,
  "message_hourly_limit": 1,
  "message_delay_min_sec": 30,
  "message_delay_max_sec": 60,
  "search_daily_limit": 15,
  "search_hourly_limit": 5,
  "search_delay_min_sec": 5,
  "search_delay_max_sec": 20
}
```

<div align="center">

**🛡️ Safety Levels:**

</div>

| Level | Description | Risk |
|:------|:------------|:-----|
| 🟢 **ultra_conservative** | Safest, lowest limits (for accounts at risk) | ⭐ Lowest |
| 🟡 **conservative** | Recommended for main accounts (default) | ⭐⭐ Low |
| 🟠 **moderate** | For established accounts | ⭐⭐⭐ Medium |
| 🔴 **aggressive** | High risk, not recommended | ⭐⭐⭐⭐ High |

### Message Templates

Edit `message_templates.json` to customize your follow-up messages:

```json
{
  "templates": [
    {
      "name": "follow_up_simple",
      "content": "Hi {name}! Thanks for accepting my connection request. Looking forward to staying in touch!",
      "description": "Simple thank you message",
      "variables": ["{name}"]
    }
  ]
}
```

### Main Configuration

Edit constants in `main.go` to customize behavior:

```go
const (
    DryRunMode = true              // Set to false to perform real actions
    EnforceSchedule = false        // Enable work hour enforcement
    SearchKeywordPeople = "software engineer"
    SearchKeywordCompanies = "E-commerce"
    SearchMaxPages = 2
    EnableOrganicBrowsing = true   // Browse profiles/feed between connections
    MessageTemplate = "follow_up_simple"
    MaxFollowUpMessages = 1
)
```

## 📖 Usage

<div align="center">

### 🎬 Available Workflows

```
╔═══════════════════════════════════════════════════════╗
║                                                       ║
║  1️⃣  SEARCH    → Find profiles & companies          ║
║  2️⃣  CONNECT   → Send connection requests           ║
║  3️⃣  FOLLOWUP  → Send follow-up messages           ║
║                                                       ║
╚═══════════════════════════════════════════════════════╝
```

</div>

---

### 1️⃣ Search Workflow 🔍

<div align="center">

**Search for people and companies on LinkedIn**

</div>

```bash
linkedin_automation.exe -workflow search
```

**What it does:**
```
┌─────────────────────────────────────────────┐
│  🔍 Searching for people...                 │
│  🏢 Searching for companies...              │
│  💾 Saving results to database...           │
│  ✅ Search complete!                        │
└─────────────────────────────────────────────┘
```

- 🔎 Search for people matching your keyword
- 🏢 Search for companies matching your keyword
- 💾 Save results to the database for later use

---

### 2️⃣ Connection Workflow 🔗

<div align="center">

**Send connection requests to discovered profiles**

</div>

```bash
linkedin_automation.exe -workflow connect
```

**What it does:**
```
┌─────────────────────────────────────────────┐
│  📋 Loading profiles from database...       │
│  🌐 Organic browsing (stealth mode)...      │
│  ✉️  Sending connection requests...        │
│  📊 Tracking in database...                │
│  ✅ Connections sent!                      │
└─────────────────────────────────────────────┘
```

- 📋 Load unprocessed profiles from the database
- 🌐 Browse profiles organically (if enabled)
- ✉️ Send connection requests with personalized notes
- 📊 Track sent requests in the database

---

### 3️⃣ Messaging Workflow 📬

<div align="center">

**Send follow-up messages to accepted connections**

</div>

```bash
linkedin_automation.exe -workflow followup
```

**What it does:**
```
┌─────────────────────────────────────────────┐
│  🔍 Detecting accepted connections...       │
│  📝 Loading message templates...            │
│  ✉️  Sending follow-up messages...          │
│  📊 Tracking message history...            │
│  ✅ Messages sent!                         │
└─────────────────────────────────────────────┘
```

- 🔍 Detect newly accepted connections
- 📝 Send follow-up messages using configured templates
- 📊 Track message history

## 🗂️ Project Structure

<div align="center">

```
📦 linkedin_automation/
│
├── 🔐 auth/                    # Authentication & Login
│   ├── checker.go             # ✅ Auth status checking
│   ├── cookies.go             # 🍪 Cookie management
│   └── login.go               # 🔑 Login functionality
│
├── 🔗 connect/                 # Connection Requests
│   └── connect.go             # 📤 Send connections
│
├── 💬 message/                 # Messaging System
│   ├── detector.go            # 🔍 Connection detection
│   ├── sender.go              # 📨 Message sending
│   ├── service.go             # ⚙️  Messaging service
│   ├── templates.go           # 📝 Template management
│   ├── tracker.go             # 📊 Message tracking
│   └── types.go               # 🏷️  Message types
│
├── 💾 persistence/             # Database Layer
│   ├── connections.go         # 🔗 Connection storage
│   ├── messages.go            # 💬 Message storage
│   ├── migration.go           # 🔄 Database migrations
│   ├── resumption.go          # ▶️  Workflow resumption
│   ├── store.go               # 🗄️  Main store interface
│   └── workflow.go             # 📋 Workflow state
│
├── 🔍 search/                  # Search Engine
│   ├── companies.go           # 🏢 Company search
│   ├── extractor.go           # 🎯 Profile extraction
│   ├── pagination.go          # 📄 Search pagination
│   ├── people.go              # 👤 People search
│   ├── scroll_helpers.go      # 📜 Scroll helpers
│   └── search.go              # 🔎 Main search logic
│
├── 🥷 stealth/                 # Stealth Features
│   ├── browsing.go            # 🌐 Organic browsing
│   ├── delays.go              # ⏱️  Random delays
│   ├── detection.go           # 🚨 Error detection
│   ├── fingerprint.go         # 🔒 Browser fingerprinting
│   ├── mouse.go               # 🖱️  Mouse simulation
│   ├── ratelimit.go           # 🚦 Rate limiting
│   ├── scheduler.go           # 📅 Schedule management
│   ├── scrolling.go           # 📜 Natural scrolling
│   └── typing.go              # ⌨️  Human-like typing
│
├── 🚀 main.go                  # 🎯 Entry point
├── ⚙️  workflows.go             # 🔄 Workflow implementations
├── 📦 go.mod                   # 📚 Go dependencies
├── ⚡ rate_config.json          # 🚦 Rate limiting config
└── 📝 message_templates.json   # 💬 Message templates
```

</div>

## 🗄️ Database

<div align="center">

**The tool uses SQLite to persist all data**

</div>

```
┌─────────────────────────────────────────────┐
│  💾 SQLite Database Storage                │
│                                             │
│  📋 Search results (people & companies)    │
│  🔗 Connection requests & status            │
│  💬 Messages sent & received                │
│  🔄 Workflow states for resumption          │
│  📊 Daily statistics                        │
│                                             │
│  📁 Database: linkedin_automation.db        │
└─────────────────────────────────────────────┘
```

## 🔒 Security & Privacy

<div align="center">

### 🛡️ Best Practices

</div>

| 🔐 Security Aspect | ✅ Best Practice |
|:------------------|:----------------|
| **🔑 Credentials** | Never commit your `.env` file to version control |
| **🍪 Cookies** | Cookie files are automatically ignored by `.gitignore` |
| **💾 Database** | Contains sensitive LinkedIn data - keep it secure |
| **🚦 Rate Limiting** | Always use conservative settings to protect your account |

<div align="center">

**🔒 Your security is our priority! 🔒**

</div>

## 📊 Statistics

<div align="center">

### 📈 What Gets Tracked

```
╔═══════════════════════════════════════════════╗
║                                               ║
║  📊 Daily Connection Requests Sent           ║
║  ✅ Connection Acceptance Rates               ║
║  💬 Messages Sent (Initial & Follow-ups)      ║
║  🔍 Profiles Discovered Through Search        ║
║  🔄 Workflow Progress & Resumption Points     ║
║                                               ║
╚═══════════════════════════════════════════════╝
```

**View statistics after each workflow run in the session summary!** 📊

</div>

## 🛠️ Troubleshooting

<div align="center">

### 🔧 Common Issues & Solutions

</div>

### 🔐 Authentication Issues

```
┌─────────────────────────────────────────────┐
│  ❌ Problem: Login fails                    │
│                                             │
│  ✅ Solutions:                              │
│  • Check .env file has correct credentials  │
│  • Verify LinkedIn 2FA (may need manual)    │
│  • Ensure cookies are being saved          │
└─────────────────────────────────────────────┘
```

### 🚦 Rate Limiting

```
┌─────────────────────────────────────────────┐
│  ❌ Problem: Hit rate limits                │
│                                             │
│  ✅ Solutions:                              │
│  • Increase delays in rate_config.json      │
│  • Switch to conservative safety level      │
│  • Reduce daily/hourly limits               │
└─────────────────────────────────────────────┘
```

### 🌐 Browser Issues

```
┌─────────────────────────────────────────────┐
│  ❌ Problem: Chrome not launching            │
│                                             │
│  ✅ Solutions:                              │
│  • Ensure Chrome at default path            │
│  • Update Chrome to latest version          │
│  • Close other Chrome instances             │
└─────────────────────────────────────────────┘
```

### 🔄 Workflow Resumption

```
┌─────────────────────────────────────────────┐
│  ❌ Problem: Workflow interrupted            │
│                                             │
│  ✅ Solutions:                              │
│  • Workflow auto-saves on interruption      │
│  • Run same workflow to resume              │
│  • Check database for workflow states       │
└─────────────────────────────────────────────┘
```

## 🧪 Testing

<div align="center">

### 🎭 Dry Run Mode

**Test without sending actual requests!**

</div>

```
┌─────────────────────────────────────────────┐
│  🧪 DRY RUN MODE (Default)                  │
│                                             │
│  ✅ Set DryRunMode = true in main.go        │
│  ✅ Workflows simulate actions              │
│  ✅ Statistics still tracked                │
│  ✅ Perfect for testing!                    │
└─────────────────────────────────────────────┘
```

<div align="center">

**💡 Always test in dry-run mode first! 💡**

</div>

## 📝 Notes

- The tool opens Chrome in non-headless mode so you can monitor activity
- Organic browsing is enabled by default to mimic human behavior
- Workflows can be paused and resumed automatically
- All actions respect rate limits and scheduling constraints

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is provided as-is for educational purposes.

## ⚠️ Important Warnings

<div align="center">

```
╔═══════════════════════════════════════════════════════╗
║                                                       ║
║  ⚠️  CRITICAL WARNINGS - READ CAREFULLY  ⚠️          ║
║                                                       ║
╚═══════════════════════════════════════════════════════╝
```

</div>

| ⚠️ Warning | 📝 Description |
|:---------:|:--------------|
| 🚫 **LinkedIn ToS** | Using automation tools may violate LinkedIn's Terms of Service |
| 🔒 **Account Risk** | Your account may be restricted or banned |
| ⏱️ **Rate Limits** | Always use conservative settings to protect your account |
| 🧪 **Testing** | Always test in dry-run mode first |
| ⚖️ **Legal** | Ensure compliance with applicable laws and regulations |

<div align="center">

**⚠️ Use at your own risk! ⚠️**

</div>

## 🔗 Dependencies

<div align="center">

### 📦 Key Libraries

</div>

| Library | Purpose | Link |
|:-------|:--------|:-----|
| 🎭 **go-rod** | Browser automation | [GitHub](https://github.com/go-rod/rod) |
| 🔐 **godotenv** | Environment variable management | [GitHub](https://github.com/joho/godotenv) |
| 💾 **sqlite** | SQLite database driver | [Website](https://modernc.org/sqlite) |

---

<div align="center">

```
╔═══════════════════════════════════════════════════════╗
║                                                       ║
║           🙏 Use Responsibly & At Your Own Risk 🙏    ║
║                                                       ║
║              Made with ❤️  and ☕                      ║
║                                                       ║
╚═══════════════════════════════════════════════════════╝
```

**⭐ Star this repo if you find it useful! ⭐**

</div>

