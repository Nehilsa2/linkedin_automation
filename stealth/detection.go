package stealth

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

// LinkedInError represents a detected LinkedIn error/warning
type LinkedInError struct {
	Type        ErrorType
	Message     string
	Recoverable bool
	Action      RecoveryAction
}

func (e *LinkedInError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// ErrorType categorizes LinkedIn errors
type ErrorType string

const (
	// Authentication errors
	ErrorCheckpoint     ErrorType = "CHECKPOINT"
	ErrorSessionExpired ErrorType = "SESSION_EXPIRED"
	ErrorPhoneVerify    ErrorType = "PHONE_VERIFICATION"
	ErrorEmailVerify    ErrorType = "EMAIL_VERIFICATION"
	ErrorCaptcha        ErrorType = "CAPTCHA"

	// Rate limiting errors
	ErrorWeeklyInviteLimit  ErrorType = "WEEKLY_INVITE_LIMIT"
	ErrorDailyInviteLimit   ErrorType = "DAILY_INVITE_LIMIT"
	ErrorMonthlySearchLimit ErrorType = "MONTHLY_SEARCH_LIMIT"
	ErrorMessageLimit       ErrorType = "MESSAGE_LIMIT"
	ErrorTooManyRequests    ErrorType = "TOO_MANY_REQUESTS"

	// Connection errors
	ErrorAlreadyConnected ErrorType = "ALREADY_CONNECTED"
	ErrorPendingInvite    ErrorType = "PENDING_INVITE"
	ErrorInviteDeclined   ErrorType = "INVITE_DECLINED"
	ErrorCannotConnect    ErrorType = "CANNOT_CONNECT"

	// Profile errors
	ErrorProfileNotFound    ErrorType = "PROFILE_NOT_FOUND"
	ErrorProfileRestricted  ErrorType = "PROFILE_RESTRICTED"
	ErrorProfileUnavailable ErrorType = "PROFILE_UNAVAILABLE"

	// Account errors
	ErrorAccountRestricted ErrorType = "ACCOUNT_RESTRICTED"
	ErrorAccountSuspended  ErrorType = "ACCOUNT_SUSPENDED"
	ErrorAccountWarning    ErrorType = "ACCOUNT_WARNING"

	// Message errors
	ErrorMessageBlocked ErrorType = "MESSAGE_BLOCKED"
	ErrorCannotMessage  ErrorType = "CANNOT_MESSAGE"
	ErrorInMailRequired ErrorType = "INMAIL_REQUIRED"

	// General errors
	ErrorPageNotLoaded ErrorType = "PAGE_NOT_LOADED"
	ErrorUnknown       ErrorType = "UNKNOWN"
)

// RecoveryAction suggests what to do when an error is detected
type RecoveryAction string

const (
	ActionStop     RecoveryAction = "STOP"     // Stop all automation immediately
	ActionWait     RecoveryAction = "WAIT"     // Wait and retry later
	ActionSkip     RecoveryAction = "SKIP"     // Skip this item, continue with next
	ActionManual   RecoveryAction = "MANUAL"   // Requires manual intervention
	ActionReauth   RecoveryAction = "REAUTH"   // Re-authenticate
	ActionCooldown RecoveryAction = "COOLDOWN" // Take a longer break
	ActionContinue RecoveryAction = "CONTINUE" // Safe to continue
)

// DetectionResult holds the result of a page check
type DetectionResult struct {
	HasError  bool
	Error     *LinkedInError
	PageURL   string
	CheckedAt time.Time
}

// ErrorPatterns defines text patterns to look for on the page
var errorPatterns = map[ErrorType][]string{
	ErrorWeeklyInviteLimit: {
		"you've reached the weekly invitation limit",
		"weekly invitation limit",
		"too many pending invitations",
		"you have too many outstanding invitations",
	},
	ErrorDailyInviteLimit: {
		"daily limit",
		"try again tomorrow",
		"limit for today",
	},
	ErrorMonthlySearchLimit: {
		"reached the monthly limit",
		"reached your monthly limit",
		"monthly search limit",
		"upgrade to premium",
		"unlimited search",
	},
	ErrorMessageLimit: {
		"message limit",
		"you can't send more messages",
		"messaging limit reached",
	},
	ErrorAccountRestricted: {
		"your account has been restricted",
		"account is restricted",
		"temporarily restricted",
		"unusual activity",
		"we've restricted your account",
	},
	ErrorAccountWarning: {
		"we noticed some activity",
		"verify it's you",
		"security check",
		"confirm your identity",
	},
	ErrorPhoneVerify: {
		"add a phone number",
		"verify your phone",
		"phone verification",
		"confirm your phone",
	},
	ErrorEmailVerify: {
		"verify your email",
		"confirm your email",
		"email verification",
	},
	ErrorCaptcha: {
		"verify you're not a robot",
		"security verification",
		"complete the security check",
		"prove you're human",
	},
	ErrorProfileNotFound: {
		"this page doesn't exist",
		"page not found",
		"profile not found",
		"couldn't find",
	},
	ErrorProfileUnavailable: {
		"this profile is not available",
		"profile is unavailable",
		"content is unavailable",
	},
	ErrorCannotConnect: {
		"unable to send invitation",
		"can't send invitation",
		"connection request failed",
	},
	ErrorCannotMessage: {
		"unable to send message",
		"can't send message",
		"message could not be sent",
	},
	ErrorInMailRequired: {
		"inmail required",
		"send an inmail",
		"requires inmail",
		"only accepts inmail",
	},
	ErrorTooManyRequests: {
		"too many requests",
		"slow down",
		"you're doing that too fast",
	},
	ErrorSessionExpired: {
		"session has expired",
		"please sign in again",
		"you've been signed out",
	},
}

// URL patterns that indicate specific states
var urlPatterns = map[ErrorType][]string{
	ErrorCheckpoint: {
		"/checkpoint/",
		"/checkpoint?",
	},
	ErrorCaptcha: {
		"/checkpoint/challenge",
		"/authwall",
	},
	ErrorSessionExpired: {
		"/login",
		"/uas/login",
	},
}

// CheckPage performs a comprehensive check of the current page for errors
func CheckPage(page *rod.Page) *DetectionResult {
	result := &DetectionResult{
		HasError:  false,
		CheckedAt: time.Now(),
	}

	// Get current URL
	info, err := page.Info()
	if err != nil {
		result.HasError = true
		result.Error = &LinkedInError{
			Type:        ErrorPageNotLoaded,
			Message:     "Failed to get page info",
			Recoverable: true,
			Action:      ActionWait,
		}
		return result
	}
	result.PageURL = info.URL

	// Check URL patterns first (faster)
	if urlErr := checkURLPatterns(info.URL); urlErr != nil {
		result.HasError = true
		result.Error = urlErr
		return result
	}

	// Check page content for error patterns
	if contentErr := checkPageContent(page); contentErr != nil {
		result.HasError = true
		result.Error = contentErr
		return result
	}

	// Check for specific DOM elements that indicate errors
	if domErr := checkDOMElements(page); domErr != nil {
		result.HasError = true
		result.Error = domErr
		return result
	}

	return result
}

// checkURLPatterns checks the URL for known error patterns
func checkURLPatterns(url string) *LinkedInError {
	urlLower := strings.ToLower(url)

	for errType, patterns := range urlPatterns {
		for _, pattern := range patterns {
			if strings.Contains(urlLower, pattern) {
				return createError(errType)
			}
		}
	}

	return nil
}

// checkPageContent checks page text for error messages
func checkPageContent(page *rod.Page) *LinkedInError {
	// Get page text content (with timeout)
	page = page.Timeout(5 * time.Second)
	defer page.CancelTimeout()

	textContent, err := page.Eval(`() => {
		return document.body ? document.body.innerText.toLowerCase() : '';
	}`)
	if err != nil {
		return nil // Don't fail on eval error, just skip this check
	}

	pageText := textContent.Value.String()

	// Check each error type's patterns
	for errType, patterns := range errorPatterns {
		for _, pattern := range patterns {
			if strings.Contains(pageText, strings.ToLower(pattern)) {
				return createError(errType)
			}
		}
	}

	return nil
}

// checkDOMElements checks for specific DOM elements indicating errors
func checkDOMElements(page *rod.Page) *LinkedInError {
	page = page.Timeout(3 * time.Second)
	defer page.CancelTimeout()

	// Check for common error modal/dialog elements
	result, err := page.Eval(`() => {
		const checks = {
			// Captcha iframe
			captcha: !!document.querySelector('iframe[src*="captcha"], iframe[src*="recaptcha"], #captcha-box'),
			
			// Restriction modal
			restricted: !!document.querySelector('[data-test-modal-id="restriction-modal"], .restriction-modal'),
			
			// Verification prompt
			verify: !!document.querySelector('[data-test-modal-id="verification-modal"], .verification-modal, [class*="challenge"]'),
			
			// Weekly limit banner
			weeklyLimit: !!document.querySelector('[class*="weekly-limit"], [class*="invitation-limit"]'),
			
			// Error toast/banner
			errorBanner: !!document.querySelector('.artdeco-toast--error, [class*="error-banner"], [class*="alert-error"]'),
			
			// Connection limit modal
			connectionLimit: !!document.querySelector('[class*="connection-limit"], [class*="invite-limit-modal"]'),
			
			// Profile unavailable
			profileUnavailable: !!document.querySelector('[class*="profile-unavailable"], .profile-not-found'),
			
			// Sign out/session modal
			sessionExpired: !!document.querySelector('[class*="session-expired"], [class*="sign-out-modal"]'),
		};
		
		return checks;
	}`)

	if err != nil {
		return nil
	}

	checks := result.Value.Map()

	if val, ok := checks["captcha"]; ok && val.Bool() {
		return createError(ErrorCaptcha)
	}
	if val, ok := checks["restricted"]; ok && val.Bool() {
		return createError(ErrorAccountRestricted)
	}
	if val, ok := checks["verify"]; ok && val.Bool() {
		return createError(ErrorPhoneVerify)
	}
	if val, ok := checks["weeklyLimit"]; ok && val.Bool() {
		return createError(ErrorWeeklyInviteLimit)
	}
	if val, ok := checks["connectionLimit"]; ok && val.Bool() {
		return createError(ErrorWeeklyInviteLimit)
	}
	if val, ok := checks["profileUnavailable"]; ok && val.Bool() {
		return createError(ErrorProfileUnavailable)
	}
	if val, ok := checks["sessionExpired"]; ok && val.Bool() {
		return createError(ErrorSessionExpired)
	}

	return nil
}

// createError creates a LinkedInError with appropriate metadata
func createError(errType ErrorType) *LinkedInError {
	err := &LinkedInError{
		Type: errType,
	}

	switch errType {
	case ErrorCheckpoint:
		err.Message = "LinkedIn checkpoint detected (2FA or verification required)"
		err.Recoverable = false
		err.Action = ActionManual

	case ErrorSessionExpired:
		err.Message = "Session expired - re-authentication required"
		err.Recoverable = true
		err.Action = ActionReauth

	case ErrorPhoneVerify:
		err.Message = "Phone verification required"
		err.Recoverable = false
		err.Action = ActionManual

	case ErrorEmailVerify:
		err.Message = "Email verification required"
		err.Recoverable = false
		err.Action = ActionManual

	case ErrorCaptcha:
		err.Message = "CAPTCHA challenge detected"
		err.Recoverable = false
		err.Action = ActionManual

	case ErrorWeeklyInviteLimit:
		err.Message = "Weekly invitation limit reached"
		err.Recoverable = true
		err.Action = ActionStop

	case ErrorDailyInviteLimit:
		err.Message = "Daily invitation limit reached"
		err.Recoverable = true
		err.Action = ActionCooldown

	case ErrorMonthlySearchLimit:
		err.Message = "Monthly search limit reached (Premium required)"
		err.Recoverable = false
		err.Action = ActionStop

	case ErrorMessageLimit:
		err.Message = "Message limit reached"
		err.Recoverable = true
		err.Action = ActionCooldown

	case ErrorAccountRestricted:
		err.Message = "Account has been restricted"
		err.Recoverable = false
		err.Action = ActionStop

	case ErrorAccountSuspended:
		err.Message = "Account has been suspended"
		err.Recoverable = false
		err.Action = ActionStop

	case ErrorAccountWarning:
		err.Message = "LinkedIn detected unusual activity"
		err.Recoverable = true
		err.Action = ActionCooldown

	case ErrorAlreadyConnected:
		err.Message = "Already connected with this user"
		err.Recoverable = true
		err.Action = ActionSkip

	case ErrorPendingInvite:
		err.Message = "Invitation already pending"
		err.Recoverable = true
		err.Action = ActionSkip

	case ErrorInviteDeclined:
		err.Message = "Previous invitation was declined"
		err.Recoverable = true
		err.Action = ActionSkip

	case ErrorCannotConnect:
		err.Message = "Unable to send connection request"
		err.Recoverable = true
		err.Action = ActionSkip

	case ErrorProfileNotFound:
		err.Message = "Profile not found"
		err.Recoverable = true
		err.Action = ActionSkip

	case ErrorProfileRestricted:
		err.Message = "Profile is restricted"
		err.Recoverable = true
		err.Action = ActionSkip

	case ErrorProfileUnavailable:
		err.Message = "Profile is unavailable"
		err.Recoverable = true
		err.Action = ActionSkip

	case ErrorMessageBlocked:
		err.Message = "Messaging blocked by user"
		err.Recoverable = true
		err.Action = ActionSkip

	case ErrorCannotMessage:
		err.Message = "Unable to send message"
		err.Recoverable = true
		err.Action = ActionSkip

	case ErrorInMailRequired:
		err.Message = "InMail required to contact this user"
		err.Recoverable = true
		err.Action = ActionSkip

	case ErrorTooManyRequests:
		err.Message = "Too many requests - slow down"
		err.Recoverable = true
		err.Action = ActionCooldown

	case ErrorPageNotLoaded:
		err.Message = "Page failed to load"
		err.Recoverable = true
		err.Action = ActionWait

	default:
		err.Message = "Unknown error detected"
		err.Recoverable = true
		err.Action = ActionWait
	}

	return err
}

// QuickCheck performs a fast URL-only check (no page content scan)
func QuickCheck(page *rod.Page) *DetectionResult {
	result := &DetectionResult{
		HasError:  false,
		CheckedAt: time.Now(),
	}

	info, err := page.Info()
	if err != nil {
		result.HasError = true
		result.Error = &LinkedInError{
			Type:        ErrorPageNotLoaded,
			Message:     "Failed to get page info",
			Recoverable: true,
			Action:      ActionWait,
		}
		return result
	}
	result.PageURL = info.URL

	if urlErr := checkURLPatterns(info.URL); urlErr != nil {
		result.HasError = true
		result.Error = urlErr
	}

	return result
}

// CheckAndHandle checks for errors and returns appropriate action
// Returns: shouldContinue (bool), error
func CheckAndHandle(page *rod.Page) (bool, error) {
	result := CheckPage(page)

	if !result.HasError {
		return true, nil
	}

	// Log the error
	fmt.Printf("⚠️ LinkedIn Error Detected: %s\n", result.Error.Error())
	fmt.Printf("   Suggested Action: %s\n", result.Error.Action)

	switch result.Error.Action {
	case ActionStop:
		fmt.Println("🛑 Stopping automation...")
		return false, result.Error

	case ActionManual:
		fmt.Println("👤 Manual intervention required. Please check browser.")
		return false, result.Error

	case ActionReauth:
		fmt.Println("🔐 Re-authentication required.")
		return false, result.Error

	case ActionCooldown:
		cooldownTime := 30 * time.Minute
		fmt.Printf("⏸️ Taking cooldown break for %v...\n", cooldownTime)
		time.Sleep(cooldownTime)
		return true, nil

	case ActionWait:
		waitTime := 5 * time.Second
		fmt.Printf("⏳ Waiting %v before retry...\n", waitTime)
		time.Sleep(waitTime)
		return true, nil

	case ActionSkip:
		fmt.Println("⏭️ Skipping current item...")
		return true, result.Error

	default:
		return true, nil
	}
}

// IsRecoverable checks if an error allows automation to continue
func IsRecoverable(err error) bool {
	if linkedInErr, ok := err.(*LinkedInError); ok {
		return linkedInErr.Recoverable
	}
	return true // Unknown errors are assumed recoverable
}

// IsCritical checks if an error requires immediate stop
func IsCritical(err error) bool {
	if linkedInErr, ok := err.(*LinkedInError); ok {
		return linkedInErr.Action == ActionStop ||
			linkedInErr.Action == ActionManual
	}
	return false
}

// WaitForPageStable waits for page to stabilize and checks for errors
func WaitForPageStable(page *rod.Page) *DetectionResult {
	// Wait for network to be idle
	page.MustWaitLoad()
	Sleep(1, 2)

	// Now check for any errors
	return CheckPage(page)
}

// MonitorPage continuously monitors a page for errors (use in goroutine)
func MonitorPage(page *rod.Page, errorChan chan<- *LinkedInError, stopChan <-chan struct{}) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			result := QuickCheck(page)
			if result.HasError {
				select {
				case errorChan <- result.Error:
				default:
					// Channel full, skip
				}
			}
		}
	}
}

// PrintDetectionStatus prints a summary of detection status
func PrintDetectionStatus(result *DetectionResult) {
	if !result.HasError {
		fmt.Println("✅ Page Status: OK")
		return
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("⚠️ ERROR DETECTED\n")
	fmt.Printf("   Type: %s\n", result.Error.Type)
	fmt.Printf("   Message: %s\n", result.Error.Message)
	fmt.Printf("   Recoverable: %v\n", result.Error.Recoverable)
	fmt.Printf("   Action: %s\n", result.Error.Action)
	fmt.Printf("   URL: %s\n", result.PageURL)
	fmt.Printf("   Time: %s\n", result.CheckedAt.Format("15:04:05"))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
