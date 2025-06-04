package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// RenderParams holds all the parameters needed for rendering
type RenderParams struct {
	Scale        float64
	OffsetX      float64
	OffsetY      float64
	TrayX        float64
	TrayWidth    int
	ScreenWidth  int
	ScreenHeight int
}

// Renderer handles map and general rendering logic
type Renderer struct{}

// NewRenderer creates a new renderer
func NewRenderer() *Renderer {
	return &Renderer{}
}

func (r *Renderer) CalculateRenderParams(screen *ebiten.Image, level *TilemapJSON) RenderParams {
	const tileSize = 64
	const trayWidth = 80

	// Get screen dimensions
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Calculate available space for the map (minus the tray)
	mapAreaWidth := screenWidth - trayWidth

	// Calculate map dimensions
	mapWidth := float64(level.Layers[0].Width * tileSize)
	mapHeight := float64(level.Layers[0].Height * tileSize)

	// Ensure map has minimum dimensions to prevent division by zero
	if mapWidth <= 0 {
		mapWidth = float64(tileSize)
	}
	if mapHeight <= 0 {
		mapHeight = float64(tileSize)
	}

	// Calculate scale - only scale if map is larger than available area
	var scale float64 = 1.0
	if mapWidth > float64(mapAreaWidth) || mapHeight > float64(screenHeight) {
		scaleX := float64(mapAreaWidth) / mapWidth
		scaleY := float64(screenHeight) / mapHeight
		scale = scaleX
		if scaleY < scaleX {
			scale = scaleY
		}
	}

	// Ensure minimum scale to prevent crashes
	const minScale = 0.1
	if scale < minScale {
		scale = minScale
	}

	// Calculate offsets to center the map
	scaledMapWidth := mapWidth * scale
	scaledMapHeight := mapHeight * scale
	offsetX := (float64(mapAreaWidth) - scaledMapWidth) / 2
	offsetY := (float64(screenHeight) - scaledMapHeight) / 2

	// Calculate tray position (right next to the actual map)
	trayX := offsetX + scaledMapWidth

	return RenderParams{
		Scale:        scale,
		OffsetX:      offsetX,
		OffsetY:      offsetY,
		TrayX:        trayX,
		TrayWidth:    trayWidth,
		ScreenWidth:  screenWidth,
		ScreenHeight: screenHeight,
	}
}
