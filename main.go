package main

import (
	"log"

	"github.com/fplonka/go-llca/game"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	// Set the right window properties. Should give pixel perfect image in fullscreen.
	ebiten.SetFullscreen(false)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSize(ebiten.ScreenSizeInFullscreen())

	// By default, update and render as fast as possible. Currently this makes the simulation speed "pulsate" slightly,
	// maybe because of GC activity?
	ebiten.SetTPS(ebiten.SyncWithFPS)
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOn)

	ebiten.SetWindowTitle("go-llca")

	g := &game.Game{}
	g.InitializeState() // Only called here.
	g.InitializeBoard()

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
