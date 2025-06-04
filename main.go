package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	sceneManager := NewSceneManager()
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Towers of Defenders")
	ebiten.SetWindowSize(1920, 1280)

	err := ebiten.RunGame(sceneManager)
	if err != nil {
		panic(err)
	}
}
