package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// AnimatedSprite represents a sprite with animation capabilities
type AnimatedSprite struct {
	frames        []*ebiten.Image // Animation frames
	frameCount    int             // Total number of frames
	frameDuration float64         // Duration per frame in seconds
	currentFrame  int             // Current frame index
	frameTimer    float64         // Timer for current frame
	isPlaying     bool            // Whether animation is currently playing
	loop          bool            // Whether to loop the animation
	totalDuration float64         // Total animation duration in seconds
}

func NewAnimatedSprite(frames []*ebiten.Image, animationLength float64, loop bool) *AnimatedSprite {
	if len(frames) == 0 {
		return nil
	}

	frameCount := len(frames)
	frameDuration := animationLength / float64(frameCount)

	return &AnimatedSprite{
		frames:        frames,
		frameCount:    frameCount,
		frameDuration: frameDuration,
		currentFrame:  0,
		frameTimer:    0.0,
		isPlaying:     false,
		loop:          loop,
		totalDuration: animationLength,
	}
}

func (as *AnimatedSprite) Play() {
	as.isPlaying = true
	as.currentFrame = 0
	as.frameTimer = 0.0
}

// IsPlaying returns true if the animation is currently playing.
func (as *AnimatedSprite) IsPlaying() bool {
	return as.isPlaying
}

// GetCurrentFrame returns the current frame image
func (as *AnimatedSprite) GetCurrentFrame() *ebiten.Image {
	if as.frameCount == 0 || as.currentFrame < 0 || as.currentFrame >= as.frameCount {
		return nil
	}
	return as.frames[as.currentFrame]
}
func (as *AnimatedSprite) Update(deltaTime float64) {
	if !as.isPlaying || as.frameCount <= 1 { // No need to update if not playing or single frame
		return
	}

	as.frameTimer += deltaTime

	// Advance frames based on how much time has passed relative to frameDuration
	// This handles cases where deltaTime might be larger than frameDuration (e.g., lag spikes)
	for as.frameTimer >= as.frameDuration && as.isPlaying { // Keep as.isPlaying check for non-looping animations
		as.frameTimer -= as.frameDuration
		as.currentFrame++

		if as.currentFrame >= as.frameCount {
			if as.loop {
				as.currentFrame = 0 // Loop back to start
			} else {
				as.currentFrame = as.frameCount - 1 // Stay on last frame
				as.isPlaying = false                // Stop animation
				// No need to subtract from frameTimer further if animation stopped
				// and we are on the last frame.
				break
			}
		}
	}
}
