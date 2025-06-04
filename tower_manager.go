package main

import (
	"math"
	"towerDefense/assets"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	BallistaTowerID    = 1
	MagicTowerID       = 2
	StageBuilding      = 0
	StageTransitioning = 1
)
const weaponRotationSpeed = 3.0      // Radians per second for weapon rotation
const weaponRotationSmoothness = 8.0 // Higher values = smoother but slower rotation
const towerRange = 5.0               // Range in tiles for tower attacks
const fireDelay = 1.5                // Seconds between shots
const fireAnimationDuration = 0.5    // Duration of firing animation (50% faster)

type TowerManager struct {
	towers             []*ebiten.Image
	placedTowers       []PlacedTower
	buildingAnimations []*BuildingAnimationState
	projectileManager  *ProjectileManager
}

// BuildingAnimationState holds the state for a tower being built
type BuildingAnimationState struct {
	X, Y             int
	TowerIDToPlace   int
	CurrentAnimation *AnimatedSprite
	Stage            int // 0 for build, 1 for transition
}

// PlacedTower represents a tower that has been placed on the map
type PlacedTower struct {
	Image           *ebiten.Image
	X               int // Grid position X
	Y               int // Grid position Y
	TowerID         int // Which tower type (index in towers array)
	Damage          float64
	WeaponImage     *ebiten.Image   // Image for the tower's weapon, if any
	WeaponAngle     float64         // Current angle of the weapon in radians. 0 = East, -PI/2 = North.
	FireTimer       float64         // Time remaining before weapon can fire again
	FiringAnimation *AnimatedSprite // Current firing animation, if any
	IdleAnimation   *AnimatedSprite // Idle animation for weapons that have one
	WeaponFired     bool            // Flag to indicate if weapon has been fired and needs to spawn a projectile
	TargetX         float64         // X position of target when weapon was fired
	TargetY         float64         // Y position of target when weapon was fired
}

const (
	waterFirstTileIDLocal = 257
	flipMask              = 0x80000000 | 0x40000000 | 0x20000000
	towerCost             = 75 // Cost to place a tower
)

func NewTowerManager() *TowerManager {
	return &TowerManager{
		towers: []*ebiten.Image{
			assets.NoneIndicator,
			assets.BallistaTower,
			assets.MagicTower,
		},
		placedTowers:       make([]PlacedTower, 0),
		buildingAnimations: make([]*BuildingAnimationState, 0),
		projectileManager:  &ProjectileManager{},
	}
}

// drawTrayBackground renders the tray background image
func (tm *TowerManager) drawTrayBackground(screen *ebiten.Image, params RenderParams) {
	trayOpts := &ebiten.DrawImageOptions{}

	// Scale the tray image to fit the available space
	trayImageBounds := assets.TrayBackground.Bounds()
	trayImageWidth := float64(trayImageBounds.Dx())
	trayImageHeight := float64(trayImageBounds.Dy())

	// Calculate scale to fit the tray area
	trayScaleX := float64(params.TrayWidth) / trayImageWidth
	trayScaleY := float64(params.ScreenHeight) / trayImageHeight

	trayOpts.GeoM.Scale(trayScaleX, trayScaleY)
	trayOpts.GeoM.Translate(params.TrayX, 0)
	screen.DrawImage(assets.TrayBackground, trayOpts)
}

// DrawTowerTray renders the tower selection tray
func (tm *TowerManager) DrawTowerTray(screen *ebiten.Image, params RenderParams, selectedTower int, uiManager *UIManager) {
	// Draw the tray background
	tm.drawTrayBackground(screen, params)
	tm.drawTowerOptions(screen, params)
}

// drawTowerOptions renders individual tower options in the tray
func (tm *TowerManager) drawTowerOptions(screen *ebiten.Image, params RenderParams) {
	const baseTowerSpacing = 140.0 // Reduced back to original spacing since no upgrade buttons
	const baseTowerStartY = 20.0
	const baseTowerWidth = 64.0
	const baseTowerHeight = 128.0

	scaledTowerSpacing := baseTowerSpacing * params.Scale
	scaledTowerStartY := baseTowerStartY * params.Scale
	scaledTowerWidth := baseTowerWidth * params.Scale
	scaledTowerHeight := baseTowerHeight * params.Scale

	for i, towerImg := range tm.towers {
		if towerImg == nil {
			continue
		}

		// Calculate tower position
		towerX := params.TrayX + (float64(params.TrayWidth)-scaledTowerWidth)/2
		towerY := scaledTowerStartY + float64(i)*scaledTowerSpacing

		// Only draw if the tower fits within the screen
		if towerY+scaledTowerHeight > float64(params.ScreenHeight) {
			continue
		}

		// Draw the tower image
		towerOpts := &ebiten.DrawImageOptions{}
		towerOpts.GeoM.Scale(params.Scale, params.Scale)
		towerOpts.GeoM.Translate(towerX, towerY)

		screen.DrawImage(towerImg, towerOpts)
	}
}

// HandleTowerSelection handles clicks on the tower tray
func (tm *TowerManager) HandleTowerSelection(level *TilemapJSON, currentGold int, params RenderParams) (bool, int) {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return false, 0
	}

	mouseX, mouseY := ebiten.CursorPosition()

	// Check if click is in the tray area
	if float64(mouseX) < params.TrayX || mouseX > params.ScreenWidth {
		return false, 0
	}

	// Calculate which tower was clicked
	const baseTowerSpacing = 140.0 // Updated to match drawTowerOptions
	const baseTowerStartY = 20.0
	const baseTowerHeight = 128.0

	scaledTowerSpacing := baseTowerSpacing * params.Scale
	scaledTowerStartY := baseTowerStartY * params.Scale
	scaledTowerHeight := baseTowerHeight * params.Scale

	relativeY := float64(mouseY) - scaledTowerStartY
	if relativeY < 0 {
		return false, 0
	}

	towerIndex := int(relativeY / scaledTowerSpacing)
	if towerIndex < 0 || towerIndex >= len(tm.towers) {
		return false, 0
	}

	// Check if click is within tower area only (not in button area)
	towerOffset := relativeY - float64(towerIndex)*scaledTowerSpacing

	// Only select tower if click is within tower image area, not button area
	if towerOffset < 0 || towerOffset > scaledTowerHeight {
		return false, 0
	}

	// Don't allow selection of towers (except "none") if player can't afford them
	if towerIndex > 0 && currentGold < towerCost {
		return false, 0 // Not enough gold to select tower
	}

	// Tower selected successfully
	return true, towerIndex
}

// DrawPlacementIndicator renders the tower image following the cursor
func (tm *TowerManager) DrawPlacementIndicator(screen *ebiten.Image, params RenderParams, selectedTowerID int, level *TilemapJSON) {
	if selectedTowerID > 0 && selectedTowerID < len(tm.towers) {
		mouseX, mouseY := ebiten.CursorPosition()
		ebiten.SetCursorMode(ebiten.CursorModeHidden)

		towerImg := tm.towers[selectedTowerID]
		if towerImg != nil && level != nil { // Ensure level is not nil
			gridX, gridY := tm.screenToGrid(mouseX, mouseY, params)
			// Corrected placement check:
			//canPlace := tm.isTileBuildable(gridX, gridY, level) && !tm.isTowerAtLocation(gridX, gridY)
			canPlace := tm.isTileBuildable(gridX, gridY, level)
			// World coordinates of the target tile's center
			tileCenterX_world := float64(gridX*level.TileWidth + level.TileWidth/2)
			tileCenterY_world := float64(gridY*level.TileHeight + level.TileHeight/2)

			imgUnscaledWidth := float64(towerImg.Bounds().Dx())
			imgUnscaledHeight := float64(towerImg.Bounds().Dy())

			// Top-left corner for drawing (world coordinates), to achieve bottom-center placement
			drawX_world := tileCenterX_world - (imgUnscaledWidth / 2)
			drawY_world := tileCenterY_world - imgUnscaledHeight // Bottom of image at tileCenterY_world

			indicatorOpts := &ebiten.DrawImageOptions{}
			indicatorOpts.GeoM.Scale(params.Scale, params.Scale)
			indicatorOpts.GeoM.Translate(
				drawX_world*params.Scale+params.OffsetX,
				drawY_world*params.Scale+params.OffsetY,
			)

			if canPlace {
				indicatorOpts.ColorScale.Scale(0.8, 1.0, 0.8, 0.5) // Greenish tint for valid
			} else {
				indicatorOpts.ColorScale.Scale(1.0, 0.5, 0.5, 0.5) // Reddish tint for invalid
			}
			screen.DrawImage(towerImg, indicatorOpts)
		} else if towerImg != nil && level == nil {
			// Fallback: Draw at cursor if level info is missing (should not happen in normal flow)
			indicatorOpts := &ebiten.DrawImageOptions{}
			indicatorOpts.GeoM.Scale(params.Scale, params.Scale)
			// Basic centering on cursor
			indicatorOpts.GeoM.Translate(float64(mouseX)-float64(towerImg.Bounds().Dx())*params.Scale/2, float64(mouseY)-float64(towerImg.Bounds().Dy())*params.Scale/2)
			screen.DrawImage(towerImg, indicatorOpts)
		}
	} else {
		ebiten.SetCursorMode(ebiten.CursorModeVisible)
	}
}

// screenToGrid converts screen coordinates to grid coordinates
func (tm *TowerManager) screenToGrid(screenX, screenY int, params RenderParams) (int, int) {
	const tileSize = 64

	// Convert screen position to map-relative position
	mapX := float64(screenX) - params.OffsetX
	mapY := float64(screenY) - params.OffsetY

	// Convert to grid coordinates
	gridX := int(mapX / (float64(tileSize) * params.Scale))
	gridY := int(mapY / (float64(tileSize) * params.Scale))

	return gridX, gridY
}

// isTileBuildable checks if a tile at given grid coordinates is buildable
func (tm *TowerManager) isTileBuildable(col, row int, level *TilemapJSON) bool {
	// Check if position is within map bounds
	if col < 0 || row < 0 {
		return false
	}
	//Inner bounds checked ok, now check outer bounds
	layer := level.Layers[0]
	if col >= layer.Width || row >= layer.Height {
		return false
	}
	// Check if there's already a tower at this position
	for _, placedTower := range tm.placedTowers {
		if placedTower.X == col && placedTower.Y == row {
			return false
		}
	}

	// Water Tiles, and Details Tiles and Road Tiles should be un-buildable.
	//So any tiles that aren't empty on those two layers should return false
	//All other tiles (grass) are buildable
	for _, layer := range level.Layers {
		index := row*layer.Width + col
		if index >= 0 && index < len(layer.Data) {
			tileID := layer.Data[index]

			// Skip empty tiles
			if tileID == 0 {
				continue
			}

			// Remove flip flags to get actual tile ID
			actualTileID := tileID &^ flipMask

			// Check if it's a water tile (cannot place towers on water)
			if actualTileID >= waterFirstTileIDLocal {
				return false
			}

			// Check if it's a details layer tile (cannot place towers on details)
			if layer.Name == "details" {
				return false
			}
		}
	}

	return true
}

func (tm *TowerManager) getTowerImage(towerID int) *ebiten.Image {
	switch towerID {
	case BallistaTowerID:
		return assets.BallistaTower
	case MagicTowerID:
		return assets.MagicTower
	default:
		return nil
	}
}

// HandleTowerPlacement handles clicks on the map to place towers
func (tm *TowerManager) HandleTowerPlacement(selectedTowerID int, level *TilemapJSON, params RenderParams, currentGold *int) {
	if selectedTowerID == 0 || level == nil { // No tower selected or level is nil
		return
	}

	mouseX, mouseY := ebiten.CursorPosition()

	// Convert screen coordinates to map grid coordinates
	// Use level.TileWidth and level.TileHeight for tile dimensions
	gridX := int((float64(mouseX) - params.OffsetX) / (float64(level.TileWidth) * params.Scale))
	gridY := int((float64(mouseY) - params.OffsetY) / (float64(level.TileHeight) * params.Scale))

	// Check if click is within map bounds (considering tray)
	mapPixelWidth := float64(level.Layers[0].Width*level.TileWidth) * params.Scale
	mapPixelHeight := float64(level.Layers[0].Height*level.TileHeight) * params.Scale

	if float64(mouseX) < params.OffsetX || float64(mouseX) > params.OffsetX+mapPixelWidth ||
		float64(mouseY) < params.OffsetY || float64(mouseY) > params.OffsetY+mapPixelHeight {
		return // Click is outside the map area
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if tm.isTileBuildable(gridX, gridY, level) { //&& !tm.isTowerAtLocation(gridX, gridY) {

			if *currentGold >= towerCost {
				*currentGold -= towerCost
				tm.startBuildingAnimation(gridX, gridY, selectedTowerID)
				return //Tower placement animation started!  This is the good case
			}
			return //Not enough gold
		}
	}
}

// startBuildingAnimation starts the building animation for a tower at the specified grid position
func (tm *TowerManager) startBuildingAnimation(col, row int, towerID int) {
	// Create a new building animation state
	buildingAnimation := &BuildingAnimationState{
		X:                col,
		Y:                row,
		TowerIDToPlace:   towerID,
		Stage:            StageBuilding,
		CurrentAnimation: NewAnimatedSprite(assets.TowerBuildAnimation, 1.0, false), // 1 second for build animation
	}

	// Start the animation
	buildingAnimation.CurrentAnimation.Play()

	// Add to the building animations slice
	tm.buildingAnimations = append(tm.buildingAnimations, buildingAnimation)
}

func (tm *TowerManager) placeTower(col, row int, towerID int) {
	towerImg := tm.getTowerImage(towerID)
	if towerImg == nil {
		return // Invalid tower ID
	}
	newTower := PlacedTower{
		Image:           towerImg,
		X:               col,
		Y:               row,
		TowerID:         towerID,
		Damage:          50,
		WeaponAngle:     -math.Pi / 2, // Initialize weapon angle to North (upwards)
		FireTimer:       0.0,          // Ready to fire immediately
		FiringAnimation: nil,          // No firing animation initially
		IdleAnimation:   nil,          // No idle animation initially
		WeaponFired:     false,        // Initialize WeaponFired to false
	} // If it's BallistaTower (ID from ballista_tower.go), add its static weapon image
	if towerID == BallistaTowerID && len(assets.BallistaWeaponFire) > 0 {
		newTower.WeaponImage = assets.BallistaWeaponFire[0]
	}

	// If it's Magic Tower (ID from magic_tower.go), add its idle animation and initial weapon image
	if towerID == MagicTowerID && len(assets.MagicTowerWeaponIdleAnimation) > 0 {
		newTower.WeaponImage = assets.MagicTowerWeaponIdleAnimation[0]
		// Create and start the looping idle animation
		newTower.IdleAnimation = NewAnimatedSprite(assets.MagicTowerWeaponIdleAnimation, 1.0, true) // 1 second duration, looping
		newTower.IdleAnimation.Play()
	}

	tm.placedTowers = append(tm.placedTowers, newTower)
}

func (tm *TowerManager) DrawPlacedTowers(screen *ebiten.Image, params RenderParams, level *TilemapJSON) {
	// STEP 1: Safety check - ensure we have valid map data
	if level == nil { // Guard against nil level
		return
	}

	// STEP 2: Iterate through all placed towers and render each one
	for _, tower := range tm.placedTowers {
		// Skip towers with missing sprites (safety check)
		if tower.Image == nil {
			continue
		}

		// STEP 3: COORDINATE CONVERSION - Grid to World
		// Convert the tower's grid position to world coordinates
		// We use the tile center as our reference point for consistent positioning
		tileCenterX := float64(tower.X*level.TileWidth + level.TileWidth/2)
		tileCenterY := float64(tower.Y*level.TileHeight + level.TileHeight/2)

		// STEP 4: TOWER BASE SPRITE POSITIONING
		// Get the dimensions of the tower sprite
		towerImg := tower.Image
		imgWidth := float64(towerImg.Bounds().Dx())
		imgHeight := float64(towerImg.Bounds().Dy())

		// Calculate the draw position for the tower base sprite
		// We want the tower to be:
		// - Horizontally centered on the tile
		// - Vertically positioned so its bottom edge sits on the tile center
		// This creates a natural "building sitting on ground" appearance
		drawX_world := tileCenterX - (imgWidth / 2.0) // Center horizontally
		drawY_world := tileCenterY - imgHeight        // Bottom-align on tile center

		// STEP 5: RENDER THE TOWER BASE
		// Create transformation matrix for the tower base sprite
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Scale(params.Scale, params.Scale) // Apply camera zoom
		opts.GeoM.Translate(                        // Convert to screen coordinates
			drawX_world*params.Scale+params.OffsetX, // World to screen X
			drawY_world*params.Scale+params.OffsetY, // World to screen Y
		)

		// Draw the tower base sprite
		screen.DrawImage(tower.Image, opts)

		// STEP 6: WEAPON RENDERING (if tower has a weapon)
		// Weapons are optional overlay sprites that can be static or animated
		if tower.WeaponImage != nil {
			// STEP 6a: DETERMINE WHICH WEAPON SPRITE TO USE
			// The weapon sprite depends on the tower's current state:
			// - Firing animation (if actively shooting)
			// - Idle animation (if has animated weapon when not shooting)
			// - Static weapon image (fallback for non-animated weapons)
			var weaponImg *ebiten.Image
			if tower.FiringAnimation != nil && tower.FiringAnimation.IsPlaying() {
				// Tower is currently firing - use firing animation frame
				weaponImg = tower.FiringAnimation.GetCurrentFrame()
			} else if tower.IdleAnimation != nil && tower.IdleAnimation.IsPlaying() {
				// Tower has idle animation (like magic tower) - use idle frame
				weaponImg = tower.IdleAnimation.GetCurrentFrame()
			} else {
				// Use static weapon image (for towers without animations)
				weaponImg = tower.WeaponImage
			}

			// STEP 6b: RENDER THE WEAPON (if we have a valid sprite)
			if weaponImg != nil {
				// Get weapon sprite dimensions
				w, h := weaponImg.Bounds().Dx(), weaponImg.Bounds().Dy()

				// Calculate tower center for weapon positioning reference
				tileCenterX := float64(tower.X*level.TileWidth + level.TileWidth/2)
				tileCenterY := float64(tower.Y*level.TileHeight + level.TileHeight/2)

				// Create transformation matrix for weapon positioning
				optsWeapon := &ebiten.DrawImageOptions{}

				// STEP 6c: TOWER-TYPE-SPECIFIC WEAPON POSITIONING
				// Different tower types position their weapons differently:
				if tower.TowerID == MagicTowerID {
					// MAGIC TOWER WEAPON POSITIONING:
					// - Weapon stays fixed (no rotation)
					// - Positioned at a specific offset from tower's top-left corner
					// - Magic towers have floating orbs that don't track targets

					weaponCenterX_local := float64(w) / 2.0 // Weapon's center point X
					weaponBottomY_local := float64(h)       // Weapon's bottom edge Y

					// Calculate tower's top-left corner in world coordinates
					towerTopLeftX := float64(tower.X * level.TileWidth)
					towerTopLeftY := float64(tower.Y * level.TileHeight)

					// Position weapon at fixed offset from tower (magic orb positioning)
					weaponAnchorX_world := towerTopLeftX + 32.0 // 32 pixels right of tower corner
					weaponAnchorY_world := towerTopLeftY        // At tower's top edge

					// Apply transformations:
					// 1. Move weapon's anchor point (center-bottom) to origin for positioning
					optsWeapon.GeoM.Translate(-weaponCenterX_local, -weaponBottomY_local)
					// 2. NO rotation for magic towers - weapons stay stationary
					// 3. Move weapon to its final position relative to tower
					optsWeapon.GeoM.Translate(weaponAnchorX_world, weaponAnchorY_world)
				} else {
					// DEFAULT WEAPON POSITIONING (Ballista and other rotating weapons):
					// - Weapon rotates to track targets
					// - Positioned at center of tower base
					// - Rotation pivot is weapon's center point

					weaponCenterX_local := float64(w) / 2.0 // Weapon's center point X
					weaponCenterY_local := float64(h) / 2.0 // Weapon's center point Y

					// Position weapon at the center of the tower base (not the sprite center)
					towerBaseCenterX_world := tileCenterX
					towerBaseCenterY_world := tileCenterY - (imgHeight / 2.0) // Center of tower base

					// Apply transformations for rotating weapons:
					// 1. Move weapon's rotation pivot (center) to origin
					optsWeapon.GeoM.Translate(-weaponCenterX_local, -weaponCenterY_local)
					// 2. Rotate weapon around origin to point toward target
					//    Note: We add π/2 to correct for sprite orientation (weapons point up by default)
					correctedAngle := tower.WeaponAngle + math.Pi/2
					optsWeapon.GeoM.Rotate(correctedAngle)
					// 3. Move rotated weapon to be centered on the tower base
					optsWeapon.GeoM.Translate(towerBaseCenterX_world, towerBaseCenterY_world)
				}

				// STEP 6d: APPLY GLOBAL TRANSFORMATIONS
				// Convert weapon from world coordinates to screen coordinates
				optsWeapon.GeoM.Scale(params.Scale, params.Scale)         // Apply camera zoom
				optsWeapon.GeoM.Translate(params.OffsetX, params.OffsetY) // Apply camera offset

				// Draw the weapon sprite
				screen.DrawImage(weaponImg, optsWeapon)
			}
		}
	}
}

// UpdateBuildingAnimations updates all towers currently in their build/transition animation
func (tm *TowerManager) UpdateBuildingAnimations(deltaTime float64) {
	updatedAnimations := make([]*BuildingAnimationState, 0, len(tm.buildingAnimations))
	for _, ba := range tm.buildingAnimations {
		if ba.CurrentAnimation != nil {
			ba.CurrentAnimation.Update(deltaTime)
			if !ba.CurrentAnimation.IsPlaying() {
				if ba.Stage == StageBuilding {
					// Transition to the next animation
					ba.Stage = StageTransitioning
					// Assuming animationLength of 0.75 second for transition, adjust as needed
					ba.CurrentAnimation = NewAnimatedSprite(assets.TowerTransitionAnimation, 0.75, false) // Adjusted duration
					ba.CurrentAnimation.Play()
					updatedAnimations = append(updatedAnimations, ba) // Keep it for next stage
				} else if ba.Stage == StageTransitioning {
					// Animation finished, place the actual tower
					tm.placeTower(ba.X, ba.Y, ba.TowerIDToPlace)
					// Do not add to updatedAnimations, effectively removing it
				}
			} else {
				updatedAnimations = append(updatedAnimations, ba) // Animation still playing
			}
		}
	}
	tm.buildingAnimations = updatedAnimations
}

// DrawBuildingAnimations draws all towers currently in their build/transition animation
func (tm *TowerManager) DrawBuildingAnimations(screen *ebiten.Image, params RenderParams, level *TilemapJSON) {
	if level == nil { // Guard against nil level
		return
	}
	const finalTowerSpriteHeight = 128.0 // Height of the final tower sprites

	for _, ba := range tm.buildingAnimations {
		if ba.CurrentAnimation != nil {
			frame := ba.CurrentAnimation.GetCurrentFrame()
			if frame != nil {
				opts := &ebiten.DrawImageOptions{}
				imgWidth := float64(frame.Bounds().Dx())  // Animation frame width
				imgHeight := float64(frame.Bounds().Dy()) // Animation frame height

				// Calculate the center of the target grid cell in world coordinates
				tileCenterX := float64(ba.X*level.TileWidth + level.TileWidth/2)
				tileCenterY := float64(ba.Y*level.TileHeight + level.TileHeight/2)

				// Calculate screenX to center the animation frame horizontally on the tile's center.
				screenX := tileCenterX - (imgWidth / 2.0)

				// Calculate screenY to align the visual center of the animation frame
				// with the visual center of the final tower sprite.
				screenY := tileCenterY - (finalTowerSpriteHeight+imgHeight)/2.0

				opts.GeoM.Scale(params.Scale, params.Scale)
				opts.GeoM.Translate(screenX*params.Scale+params.OffsetX, screenY*params.Scale+params.OffsetY)
				screen.DrawImage(frame, opts)
			}
		}
	}
}

// UpdatePlacedTowers updates the state of all placed towers, including weapon rotation and firing.
func (tm *TowerManager) UpdatePlacedTowers(deltaTime float64, activeCreeps []*Creep) {
	for i := range tm.placedTowers {
		tower := &tm.placedTowers[i]

		// Update fire timer
		if tower.FireTimer > 0 {
			tower.FireTimer -= deltaTime
			if tower.FireTimer < 0 {
				tower.FireTimer = 0
			}
		}

		// Update firing animation if active
		if tower.FiringAnimation != nil {
			tower.FiringAnimation.Update(deltaTime)
			if !tower.FiringAnimation.IsPlaying() {
				// Animation finished - spawn a projectile if weapon was fired
				if tower.WeaponFired {
					tm.spawnProjectileFromTower(tower)
				}
				tower.FiringAnimation = nil // Animation finished
			}
		}

		// Update idle animation if it exists
		if tower.IdleAnimation != nil {
			tower.IdleAnimation.Update(deltaTime)
		}

		if tower.WeaponImage == nil {
			continue // No weapon to rotate or fire
		}

		var nearestCreep *Creep
		minDistSq := math.MaxFloat64 // Use MaxFloat64 instead of -1.0

		// Tower's center in tile coordinates
		towerCenterX := float64(tower.X) + 0.5 // Add 0.5 to get center of tile
		towerCenterY := float64(tower.Y) + 0.5

		for _, creep := range activeCreeps {
			if creep == nil || !creep.IsActive() {
				continue
			}

			// Creep coordinates are already in tile coordinates
			creepX := creep.X
			creepY := creep.Y

			dx := creepX - towerCenterX
			dy := creepY - towerCenterY
			distSq := dx*dx + dy*dy

			if distSq < minDistSq {
				minDistSq = distSq
				nearestCreep = creep
			}
		}

		if nearestCreep != nil {
			// Skip weapon rotation for Magic Tower (it should remain stationary)
			if tower.TowerID != MagicTowerID {
				// Calculate angle to target (both in tile coordinates)
				dx := nearestCreep.X - towerCenterX
				dy := nearestCreep.Y - towerCenterY
				targetAngle := math.Atan2(dy, dx)

				currentAngle := tower.WeaponAngle

				// Calculate the shortest angular distance
				deltaAngle := targetAngle - currentAngle

				// Normalize angle difference to [-π, π]
				for deltaAngle > math.Pi {
					deltaAngle -= 2 * math.Pi
				}
				for deltaAngle < -math.Pi {
					deltaAngle += 2 * math.Pi
				}

				maxRotation := weaponRotationSpeed * deltaTime

				// Use smooth interpolation for more fluid rotation
				// Calculate rotation step with smoothing factor
				rotationStep := deltaAngle * weaponRotationSmoothness * deltaTime

				// Clamp the rotation step to maximum rotation speed
				if math.Abs(rotationStep) > maxRotation {
					if rotationStep > 0 {
						rotationStep = maxRotation
					} else {
						rotationStep = -maxRotation
					}
				}

				// Apply the smooth rotation step
				tower.WeaponAngle += rotationStep

				// Normalize weapon angle to [-π, π]
				for tower.WeaponAngle > math.Pi {
					tower.WeaponAngle -= 2 * math.Pi
				}
				for tower.WeaponAngle < -math.Pi {
					tower.WeaponAngle += 2 * math.Pi
				}
			}

			// Check if creep is within firing range and tower can fire
			distance := math.Sqrt(minDistSq)
			if distance <= towerRange && tower.FireTimer <= 0 {
				// Fire the weapon with target information
				tm.fireTowerWeapon(tower, nearestCreep)
			}
		}
	}
	// Update projectiles
	tm.projectileManager.Update(deltaTime, activeCreeps)

}

// fireTowerWeapon handles firing a tower's weapon
func (tm *TowerManager) fireTowerWeapon(tower *PlacedTower, targetCreep *Creep) {
	// Set the fire timer to prevent immediate refiring
	tower.FireTimer = fireDelay

	// Store target position for magic tower projectile targeting
	if targetCreep != nil {
		tower.TargetX = targetCreep.X
		tower.TargetY = targetCreep.Y
	}

	// Start firing animation based on tower type
	if tower.TowerID == BallistaTowerID {
		// Create and start ballista weapon fire animation
		if len(assets.BallistaWeaponFire) > 0 {
			tower.FiringAnimation = NewAnimatedSprite(assets.BallistaWeaponFire, fireAnimationDuration, false)
			tower.FiringAnimation.Play()

			// Spawn projectile when animation finishes - we'll implement this later
			// Store the tower reference to spawn projectile when animation finishes
			tower.WeaponFired = true
		}
	} else if tower.TowerID == MagicTowerID {
		// Create and start Magic Tower weapon fire animation
		if len(assets.MagicTowerWeaponAttackAnimation) > 0 {
			tower.FiringAnimation = NewAnimatedSprite(assets.MagicTowerWeaponAttackAnimation, fireAnimationDuration, false)
			tower.FiringAnimation.Play()

			// Store the tower reference to spawn projectile when animation finishes
			tower.WeaponFired = true
		}
	}
	// Add other tower types here as they get weapons
}
func (tm *TowerManager) spawnProjectileFromTower(tower *PlacedTower) {
	if !tower.WeaponFired {
		return // No projectile to spawn
	}

	// Reset the firing flag
	tower.WeaponFired = false

	// Tower's center in tile coordinates
	towerCenterX := float64(tower.X) + 0.5 // +0.5 to get center of tile
	towerCenterY := float64(tower.Y) + 0.5

	// Calculate weapon position based on tower type and weapon angle
	var weaponOffsetX, weaponOffsetY float64

	// Calculate weapon offset from tower center based on tower type
	if tower.TowerID == MagicTowerID {
		// Magic tower has weapon positioned at a fixed offset from tower center
		// Position slightly above the tower
		weaponOffsetX = 0
		weaponOffsetY = -0.3 // Offset upward a bit
	} else {
		// For ballista and other directional weapons, calculate position based on angle
		// The weapon barrel is about 0.4 tiles from tower center for ballista
		const weaponDistance = 0.4

		// Calculate weapon position using angle
		weaponOffsetX = math.Cos(tower.WeaponAngle) * weaponDistance
		weaponOffsetY = math.Sin(tower.WeaponAngle) * weaponDistance
	}

	// Final projectile spawn position (weapon tip position)
	spawnX := towerCenterX + weaponOffsetX
	spawnY := towerCenterY + weaponOffsetY - 1.0 // Move spawn up by 64 pixels (1 tile)

	// Use weapon angle for projectile direction
	var angle float64
	if tower.TowerID == MagicTowerID {
		// Calculate angle to the stored target position
		dx := tower.TargetX - towerCenterX
		dy := tower.TargetY - towerCenterY
		angle = math.Atan2(dy, dx)
	} else {
		angle = tower.WeaponAngle // Use the current weapon angle for other towers
	}

	// Spawn projectile
	tm.projectileManager.SpawnProjectile(spawnX, spawnY, angle, tower.TowerID)
}

// DrawProjectiles renders all active projectiles
func (tm *TowerManager) DrawProjectiles(screen *ebiten.Image, params RenderParams, level *TilemapJSON) {
	tm.projectileManager.Draw(screen, params, level)
}
