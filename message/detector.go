package message

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

// DetectNewConnections scans the connections page for newly accepted connections
func DetectNewConnections(page *rod.Page, tracker *Tracker, maxToScan int) ([]Connection, error) {
	fmt.Println("🔍 Scanning for newly accepted connections...")

	// Navigate to connections page
	connectionsURL := "https://www.linkedin.com/mynetwork/invite-connect/connections/"
	fmt.Printf("📍 Navigating to: %s\n", connectionsURL)

	timeoutPage := page.Timeout(15 * time.Second)
	err := timeoutPage.Navigate(connectionsURL)
	if err != nil {
		timeoutPage.CancelTimeout()
		return nil, fmt.Errorf("failed to navigate to connections: %w", err)
	}

	err = timeoutPage.WaitStable(time.Second)
	timeoutPage.CancelTimeout()
	if err != nil {
		fmt.Println("⚠️ Page stability wait timed out, continuing...")
	}

	time.Sleep(3 * time.Second)

	// Debug: log page URL
	fmt.Printf("📍 Current URL: %s\n", page.MustInfo().URL)

	// Extract connections from page
	result := page.MustEval(`() => {
		const connections = [];
		const maxResults = ` + fmt.Sprintf("%d", maxToScan) + `;
		
		// Debug: log what we find
		console.log('Scanning for connections...');
		
		// LinkedIn connections list selectors (updated for 2024/2025 layout)
		const cardSelectors = [
			'li.mn-connection-card',
			'.mn-connection-card',
			'[data-view-name="connections-list-item"]',
			'.scaffold-finite-scroll__content li',
			'ul.reusable-search__entity-result-list > li',
			'.reusable-search__result-container',
			'div[data-chameleon-result-urn]',
			'li.reusable-search__result-container',
		];
		
		let cards = [];
		for (const selector of cardSelectors) {
			cards = document.querySelectorAll(selector);
			console.log('Selector:', selector, 'Found:', cards.length);
			if (cards.length > 0) break;
		}

		// If still no cards, try finding any element with profile links
		if (cards.length === 0) {
			// Find all profile links and get their parent containers
			const profileLinks = document.querySelectorAll('a[href*="/in/"]');
			console.log('Found profile links:', profileLinks.length);
			
			const seenURLs = new Set();
			for (const link of profileLinks) {
				const url = link.href.split('?')[0];
				if (seenURLs.has(url)) continue;
				seenURLs.add(url);
				
				// Try to find name - check the link itself first, then container
				const container = link.closest('li') || link.closest('div[class*="entity"]') || link.parentElement?.parentElement;
				
				// Name could be in the link text or in a nearby span
				let name = 'Unknown';
				// First try: direct text in link
				const linkText = link.innerText?.trim();
				if (linkText && linkText.length > 0 && linkText.length < 100) {
					name = linkText;
				}
				// Second try: span inside link
				if (name === 'Unknown') {
					const spanInLink = link.querySelector('span[aria-hidden="true"]') || link.querySelector('span');
					if (spanInLink) name = spanInLink.innerText.trim();
				}
				// Third try: search in container
				if (name === 'Unknown' && container) {
					const nameEl = container.querySelector('.mn-connection-card__name') ||
					               container.querySelector('span[aria-hidden="true"]') ||
					               container.querySelector('[class*="name"]');
					if (nameEl) name = nameEl.innerText.trim();
				}
				
				// Try to find headline
				const headlineEl = container?.querySelector('[class*="subtitle"]') ||
				                   container?.querySelector('[class*="occupation"]') ||
				                   container?.querySelector('.mn-connection-card__occupation') ||
				                   container?.querySelector('span.t-14');
				const headline = headlineEl ? headlineEl.innerText.trim() : '';
				
				// Try to find connected time
				const timeEl = container?.querySelector('[class*="time"]') ||
				               container?.querySelector('time') ||
				               container?.querySelector('span.t-12');
				const connectedTime = timeEl ? timeEl.innerText.trim() : '';
				
				if (connections.length < maxResults) {
					connections.push({
						profileURL: url,
						name: name,
						headline: headline,
						connectedTime: connectedTime
					});
				}
			}
			return connections;
		}
		
		for (let i = 0; i < Math.min(cards.length, maxResults); i++) {
			const card = cards[i];
			
			// Extract profile link
			const linkEl = card.querySelector('a[href*="/in/"]');
			const profileURL = linkEl ? linkEl.href.split('?')[0] : null;
			
			// Extract name - try multiple selectors
			const nameEl = card.querySelector('.mn-connection-card__name') || 
			               card.querySelector('[class*="entity-result__title"]') ||
			               card.querySelector('span[aria-hidden="true"]') ||
			               card.querySelector('[class*="name"]');
			const name = nameEl ? nameEl.innerText.trim() : 'Unknown';
			
			// Extract headline/occupation
			const headlineEl = card.querySelector('.mn-connection-card__occupation') ||
			                   card.querySelector('[class*="entity-result__primary-subtitle"]') ||
			                   card.querySelector('[class*="subtitle"]');
			const headline = headlineEl ? headlineEl.innerText.trim() : '';
			
			// Extract connected time
			const timeEl = card.querySelector('.time-badge') ||
			               card.querySelector('[class*="time-ago"]') ||
			               card.querySelector('.mn-connection-card__connected-time') ||
			               card.querySelector('time');
			const connectedTime = timeEl ? timeEl.innerText.trim() : '';
			
			if (profileURL) {
				connections.push({
					profileURL: profileURL,
					name: name,
					headline: headline,
					connectedTime: connectedTime
				});
			}
		}
		
		return connections;
	}`)

	// Parse results
	var newConnections []Connection

	// Use gson.JSON array iteration
	arr := result.Arr()
	for _, item := range arr {
		profileURL := item.Get("profileURL").Str()
		name := item.Get("name").Str()
		headline := item.Get("headline").Str()
		connectedTime := item.Get("connectedTime").Str()

		// If name is Unknown, try to extract from profile URL
		if name == "Unknown" || name == "" {
			name = extractNameFromURL(profileURL)
		}

		// Check if this is a new connection (not already tracked)
		existing := tracker.GetConnection(profileURL)
		if existing == nil && profileURL != "" {
			conn := Connection{
				ProfileURL:  profileURL,
				Name:        name,
				Headline:    headline,
				Company:     extractCompany(headline),
				ConnectedAt: parseConnectedTime(connectedTime),
				HasMessaged: false,
			}
			newConnections = append(newConnections, conn)
			fmt.Printf("   ✨ New: %s (%s)\n", name, profileURL)
		}
	}

	fmt.Printf("📊 Found %d new connections\n", len(newConnections))
	return newConnections, nil
}

// extractCompany tries to extract company name from headline
func extractCompany(headline string) string {
	// Common patterns: "Title at Company", "Title @ Company", "Title | Company"
	separators := []string{" at ", " @ ", " | ", " - "}
	for _, sep := range separators {
		parts := strings.SplitN(headline, sep, 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[1])
		}
	}
	return ""
}

// extractNameFromURL extracts a readable name from LinkedIn profile URL
// e.g., "raunak-kumar-676a6a230" -> "Raunak Kumar"
func extractNameFromURL(profileURL string) string {
	// Get the username part from URL
	parts := strings.Split(profileURL, "/in/")
	if len(parts) < 2 {
		return "Unknown"
	}

	username := strings.TrimSuffix(parts[1], "/")

	// Remove trailing numbers (LinkedIn IDs)
	// e.g., "raunak-kumar-676a6a230" -> "raunak-kumar"
	nameParts := strings.Split(username, "-")
	var cleanParts []string
	for _, part := range nameParts {
		// Skip if it looks like a LinkedIn ID (mostly numbers)
		if len(part) > 3 && isNumeric(part) {
			continue
		}
		// Skip very short parts that might be IDs
		if len(part) <= 1 {
			continue
		}
		cleanParts = append(cleanParts, part)
	}

	if len(cleanParts) == 0 {
		return "Unknown"
	}

	// Capitalize each part
	for i, part := range cleanParts {
		if len(part) > 0 {
			cleanParts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}

	return strings.Join(cleanParts, " ")
}

// isNumeric checks if a string is mostly numeric
func isNumeric(s string) bool {
	digitCount := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			digitCount++
		}
	}
	return digitCount > len(s)/2
}

// parseConnectedTime attempts to parse connection time
func parseConnectedTime(timeStr string) time.Time {
	timeStr = strings.ToLower(timeStr)
	now := time.Now()

	if strings.Contains(timeStr, "just now") || strings.Contains(timeStr, "today") {
		return now
	}
	if strings.Contains(timeStr, "yesterday") {
		return now.AddDate(0, 0, -1)
	}
	if strings.Contains(timeStr, "day") {
		// "X days ago"
		return now.AddDate(0, 0, -3) // Approximate
	}
	if strings.Contains(timeStr, "week") {
		return now.AddDate(0, 0, -7)
	}
	if strings.Contains(timeStr, "month") {
		return now.AddDate(0, -1, 0)
	}

	// Default to now if can't parse
	return now
}

// GetRecentConnections returns connections from the last N days that haven't been messaged
func GetRecentConnections(tracker *Tracker, days int) []Connection {
	cutoff := time.Now().AddDate(0, 0, -days)
	var recent []Connection

	for _, conn := range tracker.Connections {
		if conn.ConnectedAt.After(cutoff) && !conn.HasMessaged {
			recent = append(recent, conn)
		}
	}

	return recent
}

// SyncNewConnections detects and adds new connections to tracker
func SyncNewConnections(page *rod.Page, tracker *Tracker, maxToScan int) (int, error) {
	newConns, err := DetectNewConnections(page, tracker, maxToScan)
	if err != nil {
		return 0, err
	}

	for _, conn := range newConns {
		tracker.AddConnection(conn)
	}

	if len(newConns) > 0 && !tracker.DryRun {
		if err := tracker.Save(); err != nil {
			fmt.Printf("⚠️ Failed to save tracker: %v\n", err)
		}
	}

	return len(newConns), nil
}
