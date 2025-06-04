package main

import (
	"bytes"
	"fmt"
	"image/color"
	"towerDefense/assets"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gobold"
)

var (
	boldFontFace *text.GoTextFace
)

func init() {
	// Initialize the bold font face
	boldFontSource, err := text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF))
	if err != nil {
		panic(err)
	}
	boldFontFace = &text.GoTextFace{
		Source: boldFontSource,
		Size:   20,
	}
}

// UIManager handles all UI rendering
type UIManager struct{}

// NewUIManager creates a new UI manager
func NewUIManager() *UIManager {
	return &UIManager{}
}

// DrawHealthBar renders the health bar and "Health:" label
func (ui *UIManager) DrawHealthBar(screen *ebiten.Image, params RenderParams, currentHealth, maxHealth int) {
	const healthBarX = 192.0
	const healthBarY = 64.0
	const labelOffset = 100.0

	// Scale the positioning and add map offset
	scaledX := healthBarX*params.Scale + params.OffsetX
	scaledY := healthBarY*params.Scale + params.OffsetY

	// Draw "Health:" label
	ui.drawHealthLabel(screen, scaledX, scaledY, labelOffset, params.Scale)

	// Draw health bar segments
	ui.drawHealthBarSegments(screen, scaledX, scaledY, params.Scale, currentHealth, maxHealth)
}

// drawHealthLabel renders the "Health:" text label
func (ui *UIManager) drawHealthLabel(screen *ebiten.Image, scaledX, scaledY, labelOffset, scale float64) {
	labelX := scaledX - labelOffset*scale
	labelY := scaledY - 4*scale

	// Create scaled font
	scaledFontFace := ui.createScaledFont(scale)

	// Create text draw options
	opts := &text.DrawOptions{}
	opts.GeoM.Translate(labelX, labelY)
	opts.ColorScale.ScaleWithColor(color.White)

	// Draw the text
	text.Draw(screen, "Health:", scaledFontFace, opts)
}

// drawHealthBarSegments renders the actual health bar with segments
func (ui *UIManager) drawHealthBarSegments(screen *ebiten.Image, scaledX, scaledY, scale float64, currentHealth, maxHealth int) {
	// Calculate what percentage of health remains (0.0 to 1.0)
	healthPercentage := float64(currentHealth) / float64(maxHealth)

	// Convert percentage to number of filled segments out of 10 total segments
	// Example: 75% health = 7.5, which becomes 7 filled segments when cast to int
	filledSegments := int(healthPercentage * 10) // 10 segments total

	// Track the current X position as we draw each piece from left to right
	currentX := scaledX

	// Draw the left end cap of the health bar (rounded left edge)
	if assets.HealthLeft != nil {
		// Create drawing options for the left cap
		leftOpts := &ebiten.DrawImageOptions{}

		// Scale the image to match the current zoom level
		leftOpts.GeoM.Scale(scale, scale)

		// Position the left cap at the starting coordinates
		leftOpts.GeoM.Translate(currentX, scaledY)

		// Actually draw the left cap to the screen
		screen.DrawImage(assets.HealthLeft, leftOpts)

		// Get the dimensions of the left cap image
		leftBounds := assets.HealthLeft.Bounds()

		// Move our X position forward by the width of the left cap (scaled)
		// This ensures the next piece will be drawn right after this one
		currentX += float64(leftBounds.Dx()) * scale
	}

	// Draw 10 individual health segments that make up the main bar
	for i := 0; i < 10; i++ {
		if assets.HealthFill != nil {
			// Create drawing options for this health segment
			fillOpts := &ebiten.DrawImageOptions{}

			// Scale the segment to match current zoom level
			fillOpts.GeoM.Scale(scale, scale)

			// Position this segment at the current X position
			fillOpts.GeoM.Translate(currentX, scaledY)

			// Determine if this segment should be filled or empty based on current health
			if i < filledSegments {
				// This segment represents health that the player still has
				// Apply color tinting based on how much health remains
				if currentHealth <= 30 {
					// Critical health: bright red (boost red, reduce green/blue)
					fillOpts.ColorScale.Scale(1.2, 0.3, 0.3, 1.0) // Bright red
				} else if currentHealth <= 50 {
					// Low health: orange warning color
					fillOpts.ColorScale.Scale(1.0, 0.6, 0.2, 1.0) // Orange
				}
				// If health > 50, use the default red color (no color scaling applied)

				// Draw the filled segment with appropriate color
				screen.DrawImage(assets.HealthFill, fillOpts)
			} else {
				// This segment represents health that has been lost
				// Make it dark gray and semi-transparent to show it's empty
				fillOpts.ColorScale.Scale(0.2, 0.2, 0.2, 0.6) // Dark gray, 60% opacity
				screen.DrawImage(assets.HealthFill, fillOpts)
			}

			// Get the dimensions of the health fill segment
			fillBounds := assets.HealthFill.Bounds()

			// Move our X position forward by the width of this segment (scaled)
			// This positions us for drawing the next segment
			currentX += float64(fillBounds.Dx()) * scale
		}
	}

	// Draw the right end cap of the health bar (rounded right edge)
	if assets.HealthRight != nil {
		// Create drawing options for the right cap
		rightOpts := &ebiten.DrawImageOptions{}

		// Scale the right cap to match current zoom level
		rightOpts.GeoM.Scale(scale, scale)

		// Position the right cap at the final X position (after all segments)
		rightOpts.GeoM.Translate(currentX, scaledY)

		// Draw the right cap to complete the health bar
		screen.DrawImage(assets.HealthRight, rightOpts)
	}
}

// DrawGoldDisplay renders the gold amount display
func (ui *UIManager) DrawGoldDisplay(screen *ebiten.Image, params RenderParams, currentGold int) {
	const healthBarX = 192.0
	const healthBarY = 64.0
	const labelOffset = 100.0

	// Position below health bar
	scaledX := healthBarX*params.Scale + params.OffsetX
	scaledY := healthBarY*params.Scale + params.OffsetY + 40*params.Scale
	labelX := scaledX - labelOffset*params.Scale

	// Create scaled font
	scaledFontFace := ui.createScaledFont(params.Scale)

	// Create text draw options
	goldOpts := &text.DrawOptions{}
	goldOpts.GeoM.Translate(labelX, scaledY)
	goldOpts.ColorScale.ScaleWithColor(color.White)

	// Draw the gold text
	goldText := fmt.Sprintf("Gold: %d", currentGold)
	text.Draw(screen, goldText, scaledFontFace, goldOpts)
}

// createScaledFont creates a font face scaled appropriately for the current scale
func (ui *UIManager) createScaledFont(scale float64) *text.GoTextFace {
	scaledFontSize := 20.0 * scale
	if scaledFontSize < 8 {
		scaledFontSize = 8 // Minimum readable size
	}

	return &text.GoTextFace{
		Source: boldFontFace.Source,
		Size:   scaledFontSize,
	}
}

// DrawWaveDisplay renders the current wave number and next wave countdown
func (ui *UIManager) DrawWaveDisplay(screen *ebiten.Image, params RenderParams, waveNumber int, waitingForWave bool, waveTimer float64) {
	const healthBarX = 192.0
	const healthBarY = 64.0
	const labelOffset = 100.0

	// Position below gold display
	scaledX := healthBarX*params.Scale + params.OffsetX
	scaledY := healthBarY*params.Scale + params.OffsetY + 80*params.Scale
	labelX := scaledX - labelOffset*params.Scale

	// Create scaled font
	scaledFontFace := ui.createScaledFont(params.Scale)

	// Create text draw options
	waveOpts := &text.DrawOptions{}
	waveOpts.GeoM.Translate(labelX, scaledY)
	waveOpts.ColorScale.ScaleWithColor(color.White)

	// Draw the wave text
	var waveText string
	if waitingForWave {
		waveText = fmt.Sprintf("Wave: %d (Next in %.1fs)", waveNumber, waveTimer)
	} else {
		waveText = fmt.Sprintf("Wave: %d", waveNumber)
	}
	text.Draw(screen, waveText, scaledFontFace, waveOpts)
}
