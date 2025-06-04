package main

import (
	"bytes"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
)

type EndScene struct {
	sceneManager *SceneManager
	titleFont    *text.GoTextFace
	subtitleFont *text.GoTextFace
}

func (t *EndScene) Draw(screen *ebiten.Image) {
	// Dark red background to indicate game over
	screen.Fill(color.RGBA{25, 10, 10, 255})

	// Get screen dimensions
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Draw "Game Over" title
	titleText := "Game Over"
	titleBounds, _ := text.Measure(titleText, t.titleFont, 0)
	titleX := (w - int(titleBounds)) / 2
	titleY := h/2 - 50

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(titleX), float64(titleY))
	op.ColorScale.ScaleWithColor(color.RGBA{255, 100, 100, 255}) // Light red text
	text.Draw(screen, titleText, t.titleFont, op)

	// Draw restart instruction
	subtitleText := "Press any key to restart"
	subtitleBounds, _ := text.Measure(subtitleText, t.subtitleFont, 0)
	subtitleX := (w - int(subtitleBounds)) / 2
	subtitleY := titleY + 80

	op2 := &text.DrawOptions{}
	op2.GeoM.Translate(float64(subtitleX), float64(subtitleY))
	op2.ColorScale.ScaleWithColor(color.RGBA{200, 150, 150, 255}) // Lighter red text
	text.Draw(screen, subtitleText, t.subtitleFont, op2)
}

func (t *EndScene) Update() error {
	// Check for key presses
	if ebiten.IsKeyPressed(ebiten.KeySpace) ||
		ebiten.IsKeyPressed(ebiten.KeyEnter) ||
		ebiten.IsKeyPressed(ebiten.KeyEscape) ||
		inpututil.IsKeyJustPressed(ebiten.KeyA) ||
		inpututil.IsKeyJustPressed(ebiten.KeyS) ||
		inpututil.IsKeyJustPressed(ebiten.KeyD) ||
		inpututil.IsKeyJustPressed(ebiten.KeyW) {
		t.sceneManager.gameScene.Reset()
		t.sceneManager.TransitionTo(SceneGame)
		return nil
	}
	// Check for mouse clicks
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		t.sceneManager.gameScene.Reset()
		t.sceneManager.TransitionTo(SceneGame)
		return nil
	}
	return nil
}

func (t *EndScene) Layout(outerWidth, outerHeight int) (int, int) {
	return outerWidth, outerHeight
}

func NewEndScene(sm *SceneManager) *EndScene {
	// Create fonts (same pattern as TitleScene)
	titleFontSource, _ := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	titleFont := &text.GoTextFace{
		Source: titleFontSource,
		Size:   48,
	}

	subtitleFontSource, _ := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	subtitleFont := &text.GoTextFace{
		Source: subtitleFontSource,
		Size:   24,
	}

	return &EndScene{
		sceneManager: sm,
		titleFont:    titleFont,
		subtitleFont: subtitleFont,
	}
}
