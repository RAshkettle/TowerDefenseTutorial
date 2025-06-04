package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// SceneType represents different game scenes
type SceneType int

const (
	SceneTitleScreen SceneType = iota
	SceneGame
	SceneEndScreen
)

type Scene interface {
	Update() error
	Draw(screen *ebiten.Image)
	Layout(outerWidth, outerHeight int) (int, int)
}

type SceneManager struct {
	currentScene Scene
	sceneType    SceneType

	// Scene instances
	titleScene *TitleScene
	gameScene  *GameScene
	endScene   *EndScene
}

// Update updates the current scene
func (sm *SceneManager) Update() error {
	return sm.currentScene.Update()
}

// Draw draws the current scene
func (sm *SceneManager) Draw(screen *ebiten.Image) {
	sm.currentScene.Draw(screen)
}

// Layout returns the screen layout from the current scene
func (sm *SceneManager) Layout(outerWidth, outerHeight int) (int, int) {
	return sm.currentScene.Layout(outerWidth, outerHeight)
}

func (sm *SceneManager) GetCurrentSceneType() SceneType {
	return sm.sceneType
}

func (sm *SceneManager) TransitionTo(sceneType SceneType) {
	sm.sceneType = sceneType

	switch sceneType {
	case SceneTitleScreen:
		sm.currentScene = sm.titleScene
	case SceneGame:
		sm.currentScene = sm.gameScene
	case SceneEndScreen:
		sm.currentScene = sm.endScene
	}
}

func NewSceneManager() *SceneManager {
	sm := &SceneManager{
		sceneType: SceneTitleScreen,
	}

	// Initialize scenes
	sm.titleScene = NewTitleScene(sm)
	sm.gameScene = NewGameScene(sm)
	sm.endScene = NewEndScene(sm)

	// Set initial scene
	sm.currentScene = sm.titleScene

	return sm
}
