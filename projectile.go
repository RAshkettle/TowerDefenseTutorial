package main

import (
	"math"
	"towerDefense/assets"

	"github.com/hajimehoshi/ebiten/v2"
)

// Projectile represents a projectile fired by a tower
type Projectile struct {
	X, Y            float64         // Position in tile coordinates
	VelocityX       float64         // Velocity in tiles per second
	VelocityY       float64         // Velocity in tiles per second
	Angle           float64         // Rotation angle in radians
	Animation       *AnimatedSprite // Projectile animation
	ImpactAnimation *AnimatedSprite // Impact animation (nil until impact)
	Active          bool            // Whether projectile is active
	TravelDistance  float64         // How far the projectile has traveled
	MaxDistance     float64         // Maximum travel distance (5 tiles)
	ProjectileType  int             // Type of projectile (based on tower type)
	IsImpacting     bool            // Whether projectile is currently playing impact animation
}

// ProjectileManager handles all active projectiles
type ProjectileManager struct {
	projectiles []Projectile
}

// Constants for projectile system
const (
	projectileSpeed    = 12.0 // Tiles per second
	maxProjectileRange = 5.0  // Maximum travel distance in tiles
	// Collision detection radii (in tiles)
	projectileCollisionRadius = 0.3 // How large projectiles are for collision
	creepCollisionRadius      = 0.4 // How large creeps are for collision
)

// NewProjectileManager creates a new projectile manager
func NewProjectileManager() *ProjectileManager {
	return &ProjectileManager{
		projectiles: make([]Projectile, 0),
	}
}

func (pm *ProjectileManager) SpawnProjectile(startX, startY float64, angle float64, projectileType int) {
	// Calculate velocity components
	velocityX := math.Cos(angle) * projectileSpeed
	velocityY := math.Sin(angle) * projectileSpeed

	// Create projectile animation based on type
	var animation *AnimatedSprite
	switch projectileType {
	case BallistaTowerID:
		if len(assets.BallistaWeaponProjectileAnimation) > 0 {
			animation = NewAnimatedSprite(assets.BallistaWeaponProjectileAnimation, 0.5, true)
			animation.Play()
		}
	case MagicTowerID:
		if len(assets.MagicTowerProjectileAnimation) > 0 {
			animation = NewAnimatedSprite(assets.MagicTowerProjectileAnimation, 0.5, true)
			animation.Play()
		}
	}

	projectile := Projectile{
		X:              startX,
		Y:              startY,
		VelocityX:      velocityX,
		VelocityY:      velocityY,
		Angle:          angle,
		Animation:      animation,
		Active:         true,
		TravelDistance: 0.0,
		MaxDistance:    maxProjectileRange,
		ProjectileType: projectileType,
		IsImpacting:    false,
	}

	pm.projectiles = append(pm.projectiles, projectile)
}

func (pm *ProjectileManager) Draw(screen *ebiten.Image, params RenderParams, level *TilemapJSON) {

	for i := range pm.projectiles {
		projectile := &pm.projectiles[i]

		if !projectile.Active {
			continue
		}

		var currentFrame *ebiten.Image

		//Use impact animation if impacting, otherwise use projectile animation
		if projectile.IsImpacting && projectile.ImpactAnimation != nil {
			currentFrame = projectile.ImpactAnimation.GetCurrentFrame()
		} else if projectile.Animation != nil {
			currentFrame = projectile.Animation.GetCurrentFrame()
		}

		if currentFrame != nil {
			// Convert tile coordinates to world coordinates
			worldX := projectile.X * float64(level.TileWidth)
			worldY := projectile.Y * float64(level.TileHeight)

			opts := &ebiten.DrawImageOptions{}

			// Get frame dimensions
			frameWidth := float64(currentFrame.Bounds().Dx())
			frameHeight := float64(currentFrame.Bounds().Dy())

			// Center the projectile on its position
			centerX := frameWidth / 2.0
			centerY := frameHeight / 2.0

			// Apply rotation (only for non-impact projectiles)
			if !projectile.IsImpacting {
				// For ballista projectiles, we need to align the sprite with the movement direction
				// The sprite's natural orientation is vertical (pointing down), so we need to adjust
				// by adding PI/2 to make it point in the direction of movement
				var rotationAngle float64

				if projectile.ProjectileType == BallistaTowerID {
					// For ballista, add PI/2 because the sprite is oriented vertically
					rotationAngle = projectile.Angle + math.Pi/2
				} else {
					// For other projectiles, use the angle directly
					rotationAngle = projectile.Angle
				}

				// Translate to center, rotate to face movement direction, then translate back
				opts.GeoM.Translate(-centerX, -centerY)
				opts.GeoM.Rotate(rotationAngle)
				opts.GeoM.Translate(centerX, centerY)
			}

			// Position at world coordinates
			opts.GeoM.Translate(worldX-centerX, worldY-centerY)

			// Apply scaling and screen offset
			opts.GeoM.Scale(params.Scale, params.Scale)
			opts.GeoM.Translate(params.OffsetX, params.OffsetY)

			screen.DrawImage(currentFrame, opts)
		}
	}
}

func (pm *ProjectileManager) startImpactAnimation(projectile *Projectile) {
	projectile.IsImpacting = true
	projectile.Animation = nil // Stop projectile animation

	// Create impact animation based on projectile type
	switch projectile.ProjectileType {
	case BallistaTowerID:
		if len(assets.BallisticWeaponImpactAnimation) > 0 {
			projectile.ImpactAnimation = NewAnimatedSprite(assets.BallisticWeaponImpactAnimation, 0.5, false)
			projectile.ImpactAnimation.Play()
		} else {
			// No impact animation available, just remove projectile
			projectile.Active = false
		}
	case MagicTowerID:
		if len(assets.MagicTowerProjectileImpactAnimation) > 0 {
			projectile.ImpactAnimation = NewAnimatedSprite(assets.MagicTowerProjectileImpactAnimation, 0.5, false)
			projectile.ImpactAnimation.Play()
		} else {
			// No impact animation available, just remove projectile
			projectile.Active = false
		}
	default:
		// Unknown projectile type, just remove
		projectile.Active = false
	}
}
func (pm *ProjectileManager) Update(deltaTime float64, activeCreeps []*Creep) {
	updatedProjectiles := make([]Projectile, 0, len(pm.projectiles))

	for i := range pm.projectiles {
		projectile := &pm.projectiles[i]

		if !projectile.Active {
			continue
		}

		// Update projectile animation
		if projectile.Animation != nil {
			projectile.Animation.Update(deltaTime)
		}

		// Update impact animation if active
		if projectile.ImpactAnimation != nil {
			projectile.ImpactAnimation.Update(deltaTime)
			if !projectile.ImpactAnimation.IsPlaying() {
				// Impact animation finished, remove projectile
				projectile.Active = false
				continue
			}
		}

		// Only move projectile if not impacting
		if !projectile.IsImpacting {
			// Check for collision with creeps before moving
			if pm.checkCollisionWithCreeps(projectile, activeCreeps) {
				// Start impact animation on collision
				pm.startImpactAnimation(projectile)
				// Keep projectile for impact animation
				if projectile.Active {
					updatedProjectiles = append(updatedProjectiles, *projectile)
				}
				continue
			}

			// Move projectile
			moveDistance := projectileSpeed * deltaTime
			projectile.X += projectile.VelocityX * deltaTime
			projectile.Y += projectile.VelocityY * deltaTime
			projectile.TravelDistance += moveDistance

			// Update angle to match movement direction
			projectile.Angle = math.Atan2(projectile.VelocityY, projectile.VelocityX)

			// Check if projectile has traveled maximum distance
			if projectile.TravelDistance >= projectile.MaxDistance {
				// Start impact animation
				pm.startImpactAnimation(projectile)
			}
		}

		// Keep active projectiles
		if projectile.Active {
			updatedProjectiles = append(updatedProjectiles, *projectile)
		}
	}

	pm.projectiles = updatedProjectiles
}
func (pm *ProjectileManager) checkCollisionWithCreeps(projectile *Projectile, activeCreeps []*Creep) bool {
	for _, creep := range activeCreeps {
		if creep == nil || !creep.IsActive() {
			continue
		}

		if creep.IsDying {
			continue // Skip creeps that are already dying
		}

		// Calculate distance between projectile and creep centers
		dx := projectile.X - creep.X
		dy := projectile.Y - creep.Y
		distance := math.Sqrt(dx*dx + dy*dy)

		// Check if collision occurred
		collisionDistance := projectileCollisionRadius + creepCollisionRadius
		if distance <= collisionDistance {
			// Collision detected! Apply damage to creep
			var damage float64
			switch projectile.ProjectileType {
			case BallistaTowerID:
				damage = 25.0 // Ballista damage
			case MagicTowerID:
				damage = 20.0 // Magic tower damage
			default:
				damage = 15.0 // Default damage
			}

			// Apply damage to the creep
			creep.TakeDamage(damage)

			return true // Collision occurred
		}
	}
	return false // No collision
}
