package main

import (
	"fmt"
	"math/rand"
	"time"

	stopwatch "github.com/RAshkettle/Stopwatch"
	"github.com/hajimehoshi/ebiten/v2"
)

type GameScene struct {
	sceneManager *SceneManager
	level        *TilemapJSON
	images       TileImageMap

	creepManager  *CreepManager
	lastUpdate    time.Time
	renderer      *Renderer
	playerHealth  int
	maxHealth     int
	spawnTimer    *stopwatch.Stopwatch
	hasSpawned    bool // Flag to prevent multiple spawns
	uiManager     *UIManager
	towerManager  *TowerManager
	currentGold   int
	goldTimer     *stopwatch.Stopwatch
	selectedTower int
}

func (g *GameScene) Draw(screen *ebiten.Image) {

	if g.level == nil {
		return
	}

	const tileSize = 64 // Standard tile size

	// Use the same logical screen size for both input and rendering to ensure consistency
	dummyImageForParams := ebiten.NewImage(1920, 1280) // Use the same dimensions as Layout()
	params := g.renderer.CalculateRenderParams(dummyImageForParams, g.level)

	// Draw layers in reverse order (last layer first)
	//for i := len(g.level.Layers) - 1; i >= 0; i-- {
	for i := 0; i < len(g.level.Layers); i++ {

		layer := g.level.Layers[i]

		// Draw each tile in the layer
		for y := 0; y < layer.Height; y++ {
			for x := 0; x < layer.Width; x++ {
				index := y*layer.Width + x
				if index < len(layer.Data) {
					tileID := layer.Data[index]

					// Skip empty tiles (ID 0)
					if tileID == 0 {
						continue
					}

					// Get the tile image from the map
					if tileImage, exists := g.images.Images[tileID]; exists && tileImage != nil {
						// Calculate screen position using render params
						screenX := float64(x * tileSize)
						screenY := float64(y * tileSize)

						// Draw the tile with proper scaling and offset
						opts := &ebiten.DrawImageOptions{}
						opts.GeoM.Scale(params.Scale, params.Scale)
						opts.GeoM.Translate(screenX*params.Scale+params.OffsetX, screenY*params.Scale+params.OffsetY)
						screen.DrawImage(tileImage, opts)
					}
				}
			}
		}
	}

	g.towerManager.DrawTowerTray(screen, params, g.selectedTower, g.uiManager)
	g.creepManager.Draw(screen, params)
	g.towerManager.DrawPlacedTowers(screen, params, g.level)
	g.uiManager.DrawHealthBar(screen, params, g.playerHealth, g.maxHealth)
	g.uiManager.DrawGoldDisplay(screen, params, g.currentGold)
	g.towerManager.DrawBuildingAnimations(screen, params, g.level)
	g.towerManager.DrawPlacementIndicator(screen, params, g.selectedTower, g.level)
	g.towerManager.DrawProjectiles(screen, params, g.level)
}

func (g *GameScene) Update() error {
	// Check if player health has reached zero - game over!
	if g.playerHealth <= 0 {
		g.sceneManager.TransitionTo(SceneEndScreen)
		return nil
	}

	// Calculate delta time
	now := time.Now()
	deltaTime := float64(0) / float64(ebiten.TPS())

	if !g.lastUpdate.IsZero() {
		deltaTime = now.Sub(g.lastUpdate).Seconds()
	}
	g.lastUpdate = now

	g.creepManager.Update(g.level, deltaTime)
	g.towerManager.UpdateBuildingAnimations(deltaTime)
	// Update the spawn timer
	g.spawnTimer.Update()

	// Check if timer is done to spawn new wave
	if g.spawnTimer.IsDone() && !g.hasSpawned {
		g.spawnNewWave()
		g.spawnTimer.Stop()
		g.hasSpawned = true
	}

	// Check if all creeps are removed and timer is not already running
	if len(g.creepManager.creeps) == 0 && !g.spawnTimer.IsRunning() && g.hasSpawned {
		g.spawnTimer.Start()
		g.hasSpawned = false
	}

	g.goldTimer.Update()
	if g.goldTimer.IsDone() {
		g.currentGold++
		g.goldTimer.Reset()
	}

	// Handle tower selection input (pass current gold for cost checking)
	// Use the logical screen size from Layout() instead of actual window size
	dummyImageForParams := ebiten.NewImage(1920, 1280) // Use the same dimensions as Layout()
	inputParams := g.renderer.CalculateRenderParams(dummyImageForParams, g.level)

	// Handle tower selection input
	if clicked, towerIndex := g.towerManager.HandleTowerSelection(g.level, g.currentGold, inputParams); clicked {
		g.selectedTower = towerIndex
	}
	g.towerManager.HandleTowerPlacement(g.selectedTower, g.level, inputParams, &g.currentGold)
	g.towerManager.UpdatePlacedTowers(deltaTime, g.creepManager.creeps)

	return nil
}

func (t *GameScene) Layout(outerWidth, outerHeight int) (int, int) {
	return 1920, 1280
}

func NewGameScene(sm *SceneManager) *GameScene {
	t, err := NewTilemapJSON("map/level.tmj")
	if err != nil {
		panic(err)
	}
	g := &GameScene{
		sceneManager: sm,
		lastUpdate:   time.Now(),                              // Initialize the timer
		creepManager: NewCreepManager(),                       // Initialize the creep manager
		spawnTimer:   stopwatch.NewStopwatch(5 * time.Second), // 5 second timer
		hasSpawned:   true,                                    // Start as true since we spawn initially
		maxHealth:    100,
		playerHealth: 100,
		uiManager:    NewUIManager(), // Initialize the UI manager
		renderer:     NewRenderer(),  // Initialize the renderer
		towerManager: NewTowerManager(),
	}
	g.level = t
	g.images = t.LoadTiles()
	g.creepManager.SetOnCreepEscape(func(damage float64) {
		g.playerHealth -= int(damage)
		if g.playerHealth < 0 {
			g.playerHealth = 0
		}
	})
	g.creepManager.SetOnCreepKilled(func(goldReward int) {
		g.currentGold += goldReward
	})
	g.currentGold = 350
	g.goldTimer = stopwatch.NewStopwatch(2 * time.Second)
	g.goldTimer.Start()
	g.creepManager.SetOnCreepKilled(func(goldReward int) {
		g.currentGold += goldReward
	})

	g.spawnNewWave()
	return g
}

// spawnNewWave spawns a new wave of creeps with randomized count
func (g *GameScene) spawnNewWave() {
	pathNodes := g.level.GetWaypoints()
	if len(pathNodes) == 0 {
		fmt.Println("Warning: No waypoints found for spawning creeps")
		return
	}

	// Spawn firebugs at the first waypoint (waypoint 0)
	startX := float64(pathNodes[0].X)
	startY := float64(pathNodes[0].Y)

	// Randomize creep count between 5-10
	creepCount := 5 + rand.Intn(6) // 5 + (0-5) = 5-10

	SpawnCreeps(g.creepManager, creepCount, startX, startY, pathNodes)

}

// Reset resets the game scene to initial state
func (g *GameScene) Reset() {
	g.playerHealth = g.maxHealth

	// Reset components
	g.creepManager = NewCreepManager()
	g.creepManager.SetOnCreepEscape(func(damage float64) {
		g.playerHealth -= int(damage)
		if g.playerHealth < 0 {
			g.playerHealth = 0
		}
	})
	// Spawn first wave
	if g.level != nil {
		pathNodes := g.level.GetWaypoints()
		if len(pathNodes) > 0 {
			g.spawnNewWave()
		}
	}
}
