package main

import (
	"math"
	"math/rand"
	"towerDefense/assets"

	"github.com/hajimehoshi/ebiten/v2"
)

// PathNode represents a single step in the path
// This is one point in our waypointing logic
type PathNode struct {
	X, Y int
}

// Direction enum for sprite facing
type Direction int

const (
	DirectionRight Direction = iota
	DirectionLeft
	DirectionUp
	DirectionDown
)

type Creep struct {
	ID               int
	X, Y             float64
	Speed            float64
	Health           float64
	MaxHealth        float64
	Path             []PathNode
	PathIndex        int
	Animation        *AnimatedSprite
	CurrentDirection Direction
	StartDelay       float64
	Timer            float64
	Active           bool
	Damage           float64
	IsDying          bool
}

func NewCreep(id int, x, y float64, path []PathNode, startDelay float64) *Creep {
	return &Creep{
		ID:               id,
		X:                x,
		Y:                y,
		Speed:            2.0 + rand.Float64()*2.0, // 2-4 speed range
		Health:           20.0,
		MaxHealth:        20.0,
		Path:             append([]PathNode(nil), path...), // Copy path
		PathIndex:        0,
		Animation:        NewAnimatedSprite(assets.FirebugSideIdle, 1.0, true),
		CurrentDirection: DirectionRight,
		StartDelay:       startDelay,
		Timer:            0,
		Active:           true,
		Damage:           2.0,
		IsDying:          false,
	}
}

// IsActive returns if the creep is still active
func (c *Creep) IsActive() bool {
	return c.Active
}

// GetID returns the creep's ID
func (c *Creep) GetID() int {
	return c.ID
}

// GetDamage returns the damage this creep deals when escaping
func (c *Creep) GetDamage() float64 {
	return c.Damage
}

// TakeDamage reduces the creep's health
func (c *Creep) TakeDamage(amount float64) {
	if c.IsDying {
		return
	}
	c.Health -= amount
	if c.Health < 0 {
		c.Health = 0
	}
}

// Draw renders the creep
func (c *Creep) Draw(screen *ebiten.Image, params RenderParams) {

	//Start by checking to make sure we have an animation.
	//Just basic bulletproofing
	if c.Animation == nil {
		return
	}

	frame := c.Animation.GetCurrentFrame()
	if frame == nil {
		return
	}

	//Maybe refactor this later to a centeral location
	tileSize := 64.0

	opts := &ebiten.DrawImageOptions{}

	// Calculate position
	worldX := c.X * tileSize
	worldY := c.Y * tileSize
	screenX := params.OffsetX + worldX*params.Scale
	screenY := params.OffsetY + worldY*params.Scale
	opts.GeoM.Scale(params.Scale, params.Scale)
	opts.GeoM.Translate(screenX, screenY)

	screen.DrawImage(frame, opts)
}

// updateDirection sets the creep's facing direction
func (c *Creep) updateDirection(dx, dy float64) {
	if math.Abs(dx) > math.Abs(dy) {
		if dx > 0 {
			c.CurrentDirection = DirectionRight
		} else {
			c.CurrentDirection = DirectionLeft
		}
	} else {
		if dy > 0 {
			c.CurrentDirection = DirectionDown
		} else {
			c.CurrentDirection = DirectionUp
		}
	}
}

// animationFramesEqual checks if two animation frame sets are the same
func animationFramesEqual(a, b []*ebiten.Image) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// setAnimation sets the appropriate animation based on state and direction
func (c *Creep) setAnimation() {
	var frames []*ebiten.Image

	// Choose animation based on movement state and direction
	if c.PathIndex >= len(c.Path)-1 || (c.PathIndex < len(c.Path)-1 && c.Timer >= c.StartDelay) {
		// Walking animation
		switch c.CurrentDirection {
		case DirectionUp:
			frames = assets.FirebugUpWalk
		case DirectionDown:
			frames = assets.FirebugDownWalk
		default: // Left/Right
			frames = assets.FirebugSideWalk
		}
	} else {
		// Idle animation
		switch c.CurrentDirection {
		case DirectionUp:
			frames = assets.FirebugUpIdle
		case DirectionDown:
			frames = assets.FirebugDownIdle
		default: // Left/Right
			frames = assets.FirebugSideIdle
		}
	}

	// Only change animation if it's different
	if c.Animation == nil || !animationFramesEqual(c.Animation.frames, frames) {
		c.Animation = NewAnimatedSprite(frames, 1.0, true)
		c.Animation.Play()
	}
}

// Update handles creep movement and state
// This function is called every frame to move the creep along its path
// deltaTime: time elapsed since last frame (in seconds)
// level: the game map containing boundaries and layout
func (c *Creep) Update(deltaTime float64, level *TilemapJSON, onEscape func(float64), onKilled func(int)) {
	// Update the internal timer - this tracks how long the creep has been alive
	c.Timer += deltaTime

	// Handle death
	if c.Health <= 0 && !c.IsDying {
		c.IsDying = true
		c.Animation = NewAnimatedSprite(assets.FirebugSideDeath, 1.0, false)
		c.Animation.Play()
		if onKilled != nil {
			onKilled(15) // Give 15 gold
		}
		return
	}

	// If dying, just update animation and check if finished
	if c.IsDying {
		c.Animation.Update(deltaTime)
		if !c.Animation.IsPlaying() {
			c.Active = false
		}
		return
	}

	// PHASE 1: Check if we should wait before starting to move
	// Some creeps have a delay before they begin moving (for staggered spawning)
	if c.Timer < c.StartDelay {
		// Still waiting, just update animation and don't move yet
		c.Animation.Update(deltaTime)
		return
	}

	// PHASE 2: Safety check - make sure we have a path to follow
	// If there's no path, the creep can't move anywhere
	if len(c.Path) == 0 {
		// No path available, just update animation
		c.Animation.Update(deltaTime)
		return
	}

	// PHASE 3: Main movement logic
	// Check if we're still following the path (not at the final waypoint yet)
	if c.PathIndex < len(c.Path)-1 {
		// We're still on the path, move toward the next waypoint
		target := c.Path[c.PathIndex+1] // Get the next waypoint to move toward

		// Calculate the direction vector from current position to target
		dx := float64(target.X) - c.X  // Horizontal distance to target
		dy := float64(target.Y) - c.Y  // Vertical distance to target
		distance := math.Hypot(dx, dy) // Total distance using Pythagorean theorem

		// Only move if there's actually distance to cover
		if distance > 0 {
			// Calculate how far we can move this frame based on speed
			moveDistance := c.Speed * deltaTime

			// Check if we can reach the target this frame
			if moveDistance >= distance {
				// We can reach the waypoint this frame - snap to it exactly
				c.X = float64(target.X)
				c.Y = float64(target.Y)
				c.PathIndex++ // Move to the next waypoint
			} else {
				// We can't reach the waypoint this frame - move part way there
				// Normalize the direction vector (make it length 1) and scale by move distance
				c.X += (dx / distance) * moveDistance
				c.Y += (dy / distance) * moveDistance
			}

			// Update which direction the sprite should face based on movement
			c.updateDirection(dx, dy)
		}
	} else {
		// PHASE 4: We've reached the end of the path - move off screen
		// Calculate the direction to continue moving (same as last path segment)
		var dx, dy float64 = 1, 0 // Default direction is right if we can't calculate

		// If we have at least 2 path points, use the direction of the last segment
		if len(c.Path) > 1 {
			last := c.Path[len(c.Path)-1] // Final waypoint
			prev := c.Path[len(c.Path)-2] // Second-to-last waypoint

			// Calculate direction vector of the last path segment
			dx = float64(last.X - prev.X)
			dy = float64(last.Y - prev.Y)

			// Normalize the direction vector so it has length 1
			norm := math.Hypot(dx, dy)
			if norm > 0 {
				dx /= norm
				dy /= norm
			}
		}

		// Continue moving in that direction to exit the screen
		moveDistance := c.Speed * deltaTime
		c.X += dx * moveDistance
		c.Y += dy * moveDistance
	}

	// PHASE 5: Check if the creep has escaped (moved outside the map boundaries)
	if level != nil && len(level.Layers) > 0 {
		// Get the map dimensions from the first layer
		mapWidth := float64(level.Layers[0].Width)
		mapHeight := float64(level.Layers[0].Height)

		// Check if creep is outside map bounds (with small buffer of -1)
		if c.X < -1 || c.X > mapWidth || c.Y < -1 || c.Y > mapHeight {
			// Creep has escaped! Mark it as inactive so it gets removed
			c.Active = false
			// TODO: This is where we would call a function to handle player losing health
			if onEscape != nil {
				onEscape(c.Damage)
			}
			return
		}
	}

	// PHASE 6: Update visual appearance
	// Set the correct animation based on current state and direction
	c.setAnimation()
	// Update the animation frames (for walking/idle cycles)
	c.Animation.Update(deltaTime)
}
