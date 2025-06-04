package assets

import (
	"bytes"
	"embed"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed *
var assets embed.FS

var GrassTileSet = loadImage("map/Grass Tileset.png")
var WaterTileSet = loadImage("map/Animated water tiles.png")

var TrayBackground = loadImage("map/tray.png")
var NoneIndicator = loadImage("ui/none.png")

// Tower sprite sheets
var ballistaTowerSpriteSheet = loadImage("towers/Tower 01.png") // Renamed from towerOneSpriteSheet
var magicTowerSpriteSheet = loadImage("towers/Tower 05.png")
var BallistaTower = getFirstTower(ballistaTowerSpriteSheet) // Renamed from towerOneLevels
var MagicTower = getFirstTower(magicTowerSpriteSheet)

var towerBuildSpriteSheet = loadImage("towers/Tower Construction.png")
var TowerBuildAnimation = loadAnimation(towerBuildSpriteSheet, 0, 6, 192, 256)
var TowerTransitionAnimation = loadAnimation(towerBuildSpriteSheet, 1, 5, 192, 256)

var magicTowerWeaponSpriteSheet = loadImage("towers/Tower 05 - Level 01 - Weapon.png")
var MagicTowerWeaponIdleAnimation = loadAnimation(magicTowerWeaponSpriteSheet, 0, 8, 96, 96)
var MagicTowerWeaponAttackAnimation = loadAnimation(magicTowerWeaponSpriteSheet, 1, 27, 96, 96)
var magicTowerProjectileSpriteSheet = loadImage("towers/Tower 05 - Level 01 - Projectile.png")
var MagicTowerProjectileAnimation = loadAnimation(magicTowerProjectileSpriteSheet, 0, 12, 32, 32)
var magicTowerProjectileImpactSpriteSheet = loadImage("towers/Tower 05 - Level 01 - Projectile - Impact.png")
var MagicTowerProjectileImpactAnimation = loadAnimation(magicTowerProjectileImpactSpriteSheet, 0, 11, 64, 64)

var ballistaWeaponSpriteSheet = loadImage("towers/Tower 01 - Level 01 - Weapon.png")               // Renamed from towerOneWeaponStyleSheet
var ballistaWeaponProjectileSpriteSheet = loadImage("towers/Tower 01 - Level 01 - Projectile.png") // Renamed from towerOneWeaponProjectileStyleSheet
var BallistaWeaponProjectileAnimation = loadAnimation(ballistaWeaponProjectileSpriteSheet, 0, 3, 8, 40)
var ballistaWeaponImpactSpriteSheet = loadImage("towers/Tower 01 - Weapon - Impact.png")
var BallisticWeaponImpactAnimation = loadAnimation(ballistaWeaponImpactSpriteSheet, 0, 6, 64, 64)
var BallistaWeaponFire = loadAnimation(ballistaWeaponSpriteSheet, 0, 6, 96, 96) // Renamed from TowerOneWeaponFire

var HealthLeft = loadImage("ui/barRed_horizontalLeft.png")
var HealthFill = loadImage("ui/barRed_horizontalMid.png")
var HealthRight = loadImage("ui/barRed_horizontalRight.png")

// Load the creeps
var FirebugSpriteSheet = loadImage("creeps/Firebug.png")

// Firebug Animations
var FirebugSideIdle = loadAnimation(FirebugSpriteSheet, 2, 5, 128, 64)
var FirebugUpIdle = loadAnimation(FirebugSpriteSheet, 1, 5, 128, 64)
var FirebugDownIdle = loadAnimation(FirebugSpriteSheet, 0, 5, 128, 64)

var FirebugDownWalk = loadAnimation(FirebugSpriteSheet, 3, 7, 128, 64)
var FirebugUpWalk = loadAnimation(FirebugSpriteSheet, 4, 7, 128, 64)
var FirebugSideWalk = loadAnimation(FirebugSpriteSheet, 5, 7, 128, 64)

var FirebugDownDeath = loadAnimation(FirebugSpriteSheet, 6, 10, 128, 64)
var FirebugUpDeath = loadAnimation(FirebugSpriteSheet, 7, 10, 128, 64)
var FirebugSideDeath = loadAnimation(FirebugSpriteSheet, 8, 10, 128, 64)

func loadImage(filePath string) *ebiten.Image {
	data, err := assets.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}

	ebitenImg := ebiten.NewImageFromImage(img)
	return ebitenImg
}

func ReadFile(filepath string) ([]byte, error) {
	return assets.ReadFile(filepath)
}
func loadAnimation(spriteSheet *ebiten.Image, row int, numberOfFrames int, frameWidth int, frameHeight int) []*ebiten.Image {
	frames := make([]*ebiten.Image, 0, numberOfFrames)

	for frameIndex := 0; frameIndex < numberOfFrames; frameIndex++ {
		x := frameIndex * frameWidth
		y := row * frameHeight

		frame := spriteSheet.SubImage(image.Rect(x, y, x+frameWidth, y+frameHeight)).(*ebiten.Image)
		frames = append(frames, frame)
	}

	return frames
}
func getFirstTower(spriteSheet *ebiten.Image) *ebiten.Image {

	// Each tower image is 64px wide and 128px tall
	//They all have 3 versions
	const towerWidth = 64
	const towerHeight = 128

	towerImage := spriteSheet.SubImage(image.Rect(0, 0, towerWidth, towerHeight)).(*ebiten.Image)

	return towerImage
}
