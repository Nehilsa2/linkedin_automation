package stealth

import (
	"math"
	"math/rand"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// MouseConfig holds configuration for human-like mouse movement
type MouseConfig struct {
	// Movement speed (base duration in ms for a "standard" move)
	BaseSpeedMs int

	// Number of steps for the curve (more = smoother but slower)
	// Keep this reasonable: 8-15 is human-like, 50+ is suspicious
	MinSteps int
	MaxSteps int

	// Overshoot settings
	OvershootChance   float64 // Probability of overshooting (0.0-1.0)
	OvershootDistance float64 // Max overshoot as fraction of total distance

	// Curve randomization (how much the control points deviate)
	CurveVariance float64 // 0.0 = straight line, 0.3 = natural curve

	// Micro-jitter (tiny random movements)
	JitterEnabled bool
	JitterAmount  float64 // pixels
}

// DefaultMouseConfig returns balanced settings for human-like movement
func DefaultMouseConfig() *MouseConfig {
	return &MouseConfig{
		BaseSpeedMs:       150,
		MinSteps:          8,
		MaxSteps:          14,
		OvershootChance:   0.15, // 15% chance to overshoot
		OvershootDistance: 0.08, // Max 8% of distance
		CurveVariance:     0.25, // Moderate curve
		JitterEnabled:     true,
		JitterAmount:      1.5,
	}
}

// Global mouse config
var MouseCfg = DefaultMouseConfig()

// MoveAndClick moves to element and clicks with human-like behavior
func MoveAndClick(page *rod.Page, el *rod.Element) error {
	return MoveAndClickWithConfig(page, el, MouseCfg)
}

// MoveAndClickWithConfig moves and clicks with custom configuration
func MoveAndClickWithConfig(page *rod.Page, el *rod.Element, cfg *MouseConfig) error {
	// Get element center position
	box, err := el.Shape()
	if err != nil {
		// Fallback to simple click
		return el.Click(proto.InputMouseButtonLeft, 1)
	}

	if box == nil || len(box.Quads) == 0 {
		return el.Click(proto.InputMouseButtonLeft, 1)
	}

	// Calculate element center
	quad := box.Quads[0]
	targetX := (quad[0] + quad[2] + quad[4] + quad[6]) / 4
	targetY := (quad[1] + quad[3] + quad[5] + quad[7]) / 4

	// Add small random offset within element (don't always click dead center)
	width := math.Abs(quad[2] - quad[0])
	height := math.Abs(quad[5] - quad[1])
	targetX += (rand.Float64() - 0.5) * width * 0.3
	targetY += (rand.Float64() - 0.5) * height * 0.3

	// Get current mouse position
	currentPos := page.Mouse.Position()

	// If mouse hasn't moved yet, start from random viewport position
	if currentPos.X == 0 && currentPos.Y == 0 {
		currentPos = getRandomViewportPos(page)
		page.Mouse.MustMoveTo(currentPos.X, currentPos.Y)
	}

	// Move to target
	err = moveMouseTo(page, currentPos, proto.Point{X: targetX, Y: targetY}, cfg)
	if err != nil {
		return err
	}

	// Small delay before click (human reaction time)
	time.Sleep(time.Duration(30+rand.Intn(70)) * time.Millisecond)

	// Click
	return page.Mouse.Click(proto.InputMouseButtonLeft, 1)
}

// moveMouseTo performs the Bézier curve movement
func moveMouseTo(page *rod.Page, from, to proto.Point, cfg *MouseConfig) error {
	distance := math.Sqrt(math.Pow(to.X-from.X, 2) + math.Pow(to.Y-from.Y, 2))

	// Skip movement for very short distances
	if distance < 5 {
		return page.Mouse.MoveTo(to)
	}

	// Calculate number of steps based on distance
	steps := cfg.MinSteps + int(distance/100)
	if steps > cfg.MaxSteps {
		steps = cfg.MaxSteps
	}

	// Generate Bézier control points
	ctrl1, ctrl2 := generateControlPoints(from, to, cfg.CurveVariance)

	// Calculate movement duration based on distance
	duration := time.Duration(cfg.BaseSpeedMs) * time.Millisecond
	duration = time.Duration(float64(duration) * (0.8 + distance/500)) // Scale with distance

	stepDelay := duration / time.Duration(steps)

	// Move along the curve
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)

		// Apply easing (slow start, fast middle, slow end)
		t = easeInOutQuad(t)

		// Calculate point on Bézier curve
		pos := cubicBezier(from, ctrl1, ctrl2, to, t)

		// Add micro-jitter (except on last step)
		if cfg.JitterEnabled && i < steps {
			pos.X += (rand.Float64() - 0.5) * cfg.JitterAmount
			pos.Y += (rand.Float64() - 0.5) * cfg.JitterAmount
		}

		// Move to this point
		page.Mouse.MustMoveTo(pos.X, pos.Y)

		// Variable delay between steps
		jitteredDelay := stepDelay + time.Duration(rand.Intn(10)-5)*time.Millisecond
		if jitteredDelay < time.Millisecond {
			jitteredDelay = time.Millisecond
		}
		time.Sleep(jitteredDelay)
	}

	// Overshoot and correct (occasional)
	if rand.Float64() < cfg.OvershootChance {
		overshootAndCorrect(page, to, distance, cfg)
	}

	return nil
}

// generateControlPoints creates Bézier control points for a natural curve
func generateControlPoints(from, to proto.Point, variance float64) (proto.Point, proto.Point) {
	dx := to.X - from.X
	dy := to.Y - from.Y
	distance := math.Sqrt(dx*dx + dy*dy)

	// Perpendicular offset for curve
	perpX := -dy / distance
	perpY := dx / distance

	// Control point 1: ~1/3 along the path with perpendicular offset
	offset1 := (rand.Float64() - 0.5) * 2 * variance * distance
	ctrl1 := proto.Point{
		X: from.X + dx*0.3 + perpX*offset1,
		Y: from.Y + dy*0.3 + perpY*offset1,
	}

	// Control point 2: ~2/3 along the path with perpendicular offset
	offset2 := (rand.Float64() - 0.5) * 2 * variance * distance
	ctrl2 := proto.Point{
		X: from.X + dx*0.7 + perpX*offset2,
		Y: from.Y + dy*0.7 + perpY*offset2,
	}

	return ctrl1, ctrl2
}

// cubicBezier calculates a point on a cubic Bézier curve
func cubicBezier(p0, p1, p2, p3 proto.Point, t float64) proto.Point {
	// B(t) = (1-t)³P0 + 3(1-t)²tP1 + 3(1-t)t²P2 + t³P3
	mt := 1 - t
	mt2 := mt * mt
	mt3 := mt2 * mt
	t2 := t * t
	t3 := t2 * t

	return proto.Point{
		X: mt3*p0.X + 3*mt2*t*p1.X + 3*mt*t2*p2.X + t3*p3.X,
		Y: mt3*p0.Y + 3*mt2*t*p1.Y + 3*mt*t2*p2.Y + t3*p3.Y,
	}
}

// easeInOutQuad provides smooth acceleration/deceleration
func easeInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return 1 - math.Pow(-2*t+2, 2)/2
}

// overshootAndCorrect simulates overshooting the target and correcting
func overshootAndCorrect(page *rod.Page, target proto.Point, distance float64, cfg *MouseConfig) {
	// Calculate overshoot amount
	overshootDist := distance * cfg.OvershootDistance * (0.5 + rand.Float64()*0.5)

	// Random direction for overshoot
	angle := rand.Float64() * 2 * math.Pi
	overshootPos := proto.Point{
		X: target.X + math.Cos(angle)*overshootDist,
		Y: target.Y + math.Sin(angle)*overshootDist,
	}

	// Move to overshoot position (quick)
	page.Mouse.MustMoveTo(overshootPos.X, overshootPos.Y)
	time.Sleep(time.Duration(15+rand.Intn(25)) * time.Millisecond)

	// Correct back to target (2-3 quick steps)
	correctionSteps := 2 + rand.Intn(2)
	for i := 1; i <= correctionSteps; i++ {
		t := float64(i) / float64(correctionSteps)
		x := overshootPos.X + (target.X-overshootPos.X)*t
		y := overshootPos.Y + (target.Y-overshootPos.Y)*t
		page.Mouse.MustMoveTo(x, y)
		time.Sleep(time.Duration(10+rand.Intn(15)) * time.Millisecond)
	}
}

// getRandomViewportPos returns a random position within the viewport
func getRandomViewportPos(page *rod.Page) proto.Point {
	result := page.MustEval(`() => ({
		width: window.innerWidth,
		height: window.innerHeight
	})`)

	width := result.Get("width").Num()
	height := result.Get("height").Num()

	return proto.Point{
		X: width * (0.3 + rand.Float64()*0.4),  // 30-70% of width
		Y: height * (0.3 + rand.Float64()*0.4), // 30-70% of height
	}
}

// ClickElement is a convenience wrapper for MoveAndClick
func ClickElement(page *rod.Page, el *rod.Element) error {
	return MoveAndClick(page, el)
}

// ClickSelector finds element by selector and clicks with human-like movement
func ClickSelector(page *rod.Page, selector string) error {
	el, err := page.Element(selector)
	if err != nil {
		return err
	}
	return MoveAndClick(page, el)
}

// HoverElement moves mouse to element without clicking (for hover states)
func HoverElement(page *rod.Page, el *rod.Element) error {
	box, err := el.Shape()
	if err != nil {
		return el.Hover()
	}

	if box == nil || len(box.Quads) == 0 {
		return el.Hover()
	}

	quad := box.Quads[0]
	targetX := (quad[0] + quad[2] + quad[4] + quad[6]) / 4
	targetY := (quad[1] + quad[3] + quad[5] + quad[7]) / 4

	currentPos := page.Mouse.Position()
	if currentPos.X == 0 && currentPos.Y == 0 {
		currentPos = getRandomViewportPos(page)
	}

	return moveMouseTo(page, currentPos, proto.Point{X: targetX, Y: targetY}, MouseCfg)
}
