package main

import (
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// CreepManager handles spawning creeps and tracking their count
type CreepManager struct {
	creeps        []*Creep
	onCreepEscape func(damage float64)
	onCreepKilled func(goldReward int)
	nextCreepID   int
}

func NewCreepManager() *CreepManager {
	return &CreepManager{
		nextCreepID: 1,
	}
}

// SetOnCreepEscape sets the callback for when a creep escapes
func (cm *CreepManager) SetOnCreepEscape(cb func(damage float64)) {
	cm.onCreepEscape = cb
}

// SetOnCreepKilled sets the callback for when a creep is killed
func (cm *CreepManager) SetOnCreepKilled(cb func(goldReward int)) {
	cm.onCreepKilled = cb
}

// SpawnCreeps creates and adds creeps to the manager
func SpawnCreeps(manager *CreepManager, numCreeps int, startX, startY float64, pathNodes []PathNode) {
	if manager == nil {
		return
	}

	for i := 0; i < numCreeps; i++ {
		startDelay := 1.0 + rand.Float64()*4.0 // Random delay 1-5 seconds
		creep := NewCreep(manager.GetNextCreepID(), startX, startY, pathNodes, startDelay)
		manager.AddCreep(creep)
	}
}

func (cm *CreepManager) Update(level *TilemapJSON, deltaTime float64) {
	var remainingCreeps []*Creep
	for _, creep := range cm.creeps {
		if creep.IsActive() {
			// Create a callback function to handle escapes
			onEscape := func(damage float64) {
				if cm.onCreepEscape != nil {
					cm.onCreepEscape(damage)
				}

			}
			creep.Update(deltaTime, level, onEscape, cm.onCreepKilled)
		}
		// Check if creep is still active after update (may have escaped)
		if creep.IsActive() {
			remainingCreeps = append(remainingCreeps, creep)
		}
	}
	cm.creeps = remainingCreeps
}

// Draw renders all creeps
func (cm *CreepManager) Draw(screen *ebiten.Image, params RenderParams) {
	for _, creep := range cm.creeps {
		if creep.IsActive() {
			creep.Draw(screen, params)
		}
	}
}

// AddCreep adds a new creep
func (cm *CreepManager) AddCreep(creep *Creep) {
	cm.creeps = append(cm.creeps, creep)
}

// GetNextCreepID provides a unique ID for a new creep
func (cm *CreepManager) GetNextCreepID() int {
	id := cm.nextCreepID
	cm.nextCreepID++
	return id
}
