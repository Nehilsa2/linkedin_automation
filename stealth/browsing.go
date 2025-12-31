package stealth

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/go-rod/rod"
)

// BrowsingConfig holds configuration for organic browsing behavior
type BrowsingConfig struct {
	// Profile viewing
	ProfileViewMin int // seconds to spend viewing a profile
	ProfileViewMax int

	// Feed scrolling
	FeedScrollMin int // seconds to spend on feed
	FeedScrollMax int
	FeedScrolls   int // number of scroll actions on feed

	// Probabilities
	ViewAboutChance   float64 // chance to click "see more" on about
	ViewPostsChance   float64 // chance to scroll to posts section
	LikePostChance    float64 // chance to like a post on feed (keep low!)
	CheckNotifyChance float64 // chance to check notifications

	// Delays
	BetweenActionsMin int // seconds between browse actions
	BetweenActionsMax int
}

// DefaultBrowsingConfig returns safe defaults for organic browsing
func DefaultBrowsingConfig() *BrowsingConfig {
	return &BrowsingConfig{
		ProfileViewMin:    8,
		ProfileViewMax:    15,
		FeedScrollMin:     4,
		FeedScrollMax:     8,
		FeedScrolls:       3,
		ViewAboutChance:   0.3,  // 30% chance to expand about
		ViewPostsChance:   0.2,  // 20% chance to scroll to posts
		LikePostChance:    0.05, // 5% - very low to avoid patterns
		CheckNotifyChance: 0.15, // 15% chance to check notifications
		BetweenActionsMin: 2,
		BetweenActionsMax: 5,
	}
}

// Global browsing config
var BrowseCfg = DefaultBrowsingConfig()

// OrganicBrowser handles human-like browsing behavior
type OrganicBrowser struct {
	config *BrowsingConfig
	page   *rod.Page
}

// NewOrganicBrowser creates a new organic browser
func NewOrganicBrowser(page *rod.Page) *OrganicBrowser {
	return &OrganicBrowser{
		config: BrowseCfg,
		page:   page,
	}
}

// NewOrganicBrowserWithConfig creates browser with custom config
func NewOrganicBrowserWithConfig(page *rod.Page, cfg *BrowsingConfig) *OrganicBrowser {
	return &OrganicBrowser{
		config: cfg,
		page:   page,
	}
}

// BrowseProfile visits a profile and spends time viewing it naturally
// Returns error if page fails to load
func (ob *OrganicBrowser) BrowseProfile(profileURL string) error {
	fmt.Printf("👀 Browsing profile: %s\n", truncateURL(profileURL))

	// Navigate to profile
	err := ob.page.Navigate(profileURL)
	if err != nil {
		return fmt.Errorf("failed to navigate to profile: %w", err)
	}

	// Wait for page to load
	ob.page.MustWaitLoad()
	SleepMillis(500, 1000)

	// Check for LinkedIn errors
	if result := CheckPage(ob.page); result.HasError {
		return result.Error
	}

	// Random view duration
	viewDuration := rand.Intn(ob.config.ProfileViewMax-ob.config.ProfileViewMin+1) + ob.config.ProfileViewMin
	fmt.Printf("   📖 Reading profile for %d seconds...\n", viewDuration)

	// Split view time into scroll segments
	segments := 3 + rand.Intn(3) // 3-5 segments
	segmentTime := viewDuration / segments

	for i := 0; i < segments; i++ {
		// Scroll down a bit
		ScrollDown(ob.page)

		// Wait (simulating reading)
		time.Sleep(time.Duration(segmentTime) * time.Second)

		// Small random variation
		SleepMillis(200, 800)
	}

	// Maybe expand "About" section
	if rand.Float64() < ob.config.ViewAboutChance {
		ob.tryExpandAbout()
	}

	// Maybe scroll to posts/activity
	if rand.Float64() < ob.config.ViewPostsChance {
		ob.scrollToActivity()
	}

	fmt.Printf("   ✅ Done browsing profile\n")
	return nil
}

// BrowseProfileQuick does a shorter profile view (for target before connect)
func (ob *OrganicBrowser) BrowseProfileQuick(profileURL string) error {
	fmt.Printf("👀 Quick view: %s\n", truncateURL(profileURL))

	// Navigate to profile
	err := ob.page.Navigate(profileURL)
	if err != nil {
		return fmt.Errorf("failed to navigate to profile: %w", err)
	}

	// Wait for page to load
	ob.page.MustWaitLoad()
	SleepMillis(500, 1000)

	// Check for LinkedIn errors
	if result := CheckPage(ob.page); result.HasError {
		return result.Error
	}

	// Shorter view time (3-6 seconds)
	viewTime := 3 + rand.Intn(4)
	fmt.Printf("   📖 Quick scan for %d seconds...\n", viewTime)

	// One or two scrolls
	ScrollDown(ob.page)
	time.Sleep(time.Duration(viewTime) * time.Second)

	if rand.Float64() < 0.5 {
		ScrollDown(ob.page)
		SleepMillis(500, 1500)
	}

	return nil
}

// BrowseFeed navigates to feed and scrolls naturally
func (ob *OrganicBrowser) BrowseFeed() error {
	fmt.Println("📰 Browsing feed...")

	// Navigate to feed
	err := ob.page.Navigate("https://www.linkedin.com/feed/")
	if err != nil {
		return fmt.Errorf("failed to navigate to feed: %w", err)
	}

	ob.page.MustWaitLoad()
	SleepMillis(1000, 2000)

	// Check for errors
	if result := CheckPage(ob.page); result.HasError {
		return result.Error
	}

	// Random time on feed
	feedTime := rand.Intn(ob.config.FeedScrollMax-ob.config.FeedScrollMin+1) + ob.config.FeedScrollMin
	fmt.Printf("   📜 Scrolling feed for %d seconds...\n", feedTime)

	scrollCount := ob.config.FeedScrolls + rand.Intn(2) // 3-4 scrolls
	scrollInterval := feedTime / scrollCount

	for i := 0; i < scrollCount; i++ {
		ScrollDown(ob.page)
		time.Sleep(time.Duration(scrollInterval) * time.Second)

		// Random pause (reading a post)
		if rand.Float64() < 0.4 {
			SleepMillis(500, 1500)
		}
	}

	// Very rare: like a post (keep this LOW)
	if rand.Float64() < ob.config.LikePostChance {
		ob.tryLikePost()
	}

	fmt.Println("   ✅ Done browsing feed")
	return nil
}

// CheckNotifications visits the notifications page briefly
func (ob *OrganicBrowser) CheckNotifications() error {
	if rand.Float64() > ob.config.CheckNotifyChance {
		return nil // Skip this time
	}

	fmt.Println("🔔 Checking notifications...")

	err := ob.page.Navigate("https://www.linkedin.com/notifications/")
	if err != nil {
		return err
	}

	ob.page.MustWaitLoad()

	// Brief look (2-4 seconds)
	time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)

	// Maybe scroll once
	if rand.Float64() < 0.5 {
		ScrollDown(ob.page)
		SleepMillis(500, 1500)
	}

	fmt.Println("   ✅ Done checking notifications")
	return nil
}

// tryExpandAbout attempts to click "see more" on profile about section
func (ob *OrganicBrowser) tryExpandAbout() {
	// Try to find and click "see more" in about section
	seeMore := ob.page.MustElement("body").MustElements("button")

	for _, btn := range seeMore {
		text, _ := btn.Text()
		if text == "see more" || text == "…see more" {
			// Found it - click with human-like behavior
			MoveAndClick(ob.page, btn)
			SleepMillis(800, 1500)
			break
		}
	}
}

// scrollToActivity scrolls down to the activity/posts section
func (ob *OrganicBrowser) scrollToActivity() {
	fmt.Println("   📜 Scrolling to activity section...")

	// Scroll down several times to reach activity
	for i := 0; i < 3; i++ {
		ScrollDown(ob.page)
		SleepMillis(300, 700)
	}

	// Pause to "read" activity
	time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)
}

// tryLikePost attempts to like a post on the feed (very rare action)
func (ob *OrganicBrowser) tryLikePost() {
	// Find like buttons - but only do this VERY rarely
	// This is risky behavior, so we keep probability very low
	fmt.Println("   👍 Considering liking a post...")

	// Just a placeholder - actual implementation would find like buttons
	// But we keep this minimal to avoid detection
	SleepMillis(500, 1000)
}

// RandomDelay adds a random delay between browse actions
func (ob *OrganicBrowser) RandomDelay() {
	min := ob.config.BetweenActionsMin
	max := ob.config.BetweenActionsMax
	delay := rand.Intn(max-min+1) + min
	time.Sleep(time.Duration(delay) * time.Second)
}

// PerformOrganicCycle does one cycle of organic browsing before an action
// Pattern: Browse random profile -> Feed -> (ready for target)
func (ob *OrganicBrowser) PerformOrganicCycle(browseProfileURL string) error {
	// Step 1: Browse a random profile (longer view)
	if browseProfileURL != "" {
		err := ob.BrowseProfile(browseProfileURL)
		if err != nil {
			fmt.Printf("   ⚠️ Browse failed: %v (continuing)\n", err)
			// Non-fatal - continue with workflow
		}
		ob.RandomDelay()
	}

	// Step 2: Check feed
	err := ob.BrowseFeed()
	if err != nil {
		fmt.Printf("   ⚠️ Feed browse failed: %v (continuing)\n", err)
	}
	ob.RandomDelay()

	// Step 3: Maybe check notifications (random)
	ob.CheckNotifications()

	return nil
}

// truncateURL shortens a URL for display
func truncateURL(url string) string {
	if len(url) > 60 {
		return url[:57] + "..."
	}
	return url
}
