package main

import (
	"image/color"
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Random number source for game board initialization.
var r *rand.Rand

// Seed for the random number source. r is seeded only once and is not reinitialized with the seed before every run, so
// the order in which simulation runs are started will affect their initial board states.
const SEED = 0

// The value at index i corresponds to the birth/survival rule for when i neighbours are alive.
type Ruleset [9]bool

type Game struct {
	// UI state, mostly for pause menu.
	ui UI

	// Grid state.
	// Each uint8 value represents both the state of the cell at that position (dead or alive) and the number of living
	// neighbours (from 0 to 9) of that cells.
	//
	// The last bit is 0 if the cell is dead and 1 if the cell is alive.
	// The other bits (value>>1, i.e. all except the last) represent the number of living neighbours. This number of
	// live neigbours INCLUDES that cell if it is alive, so a cell can have up to 9 live "neighbours". This lets us
	// update the board state slightly more efficiently.
	//
	//
	// The worldGrid slice is implicitly two dimensional. There is a one cell border around the grid filled with cells
	// which are always dead. This allows us to skip index out of bounds checks.
	//
	// The grid size is dependent on the scale factor.
	worldGrid    []uint8
	buffer       []uint8
	gridX, gridY int

	// Dead cells are black, live cells are white. The size of pixels is like that of the board but without the border.
	pixels *ebiten.Image

	// Semi-transparent image to cover and "dim" the simulation image when paused.
	transparencyOverlay *ebiten.Image

	// Game rules.
	// A dead cell becomes alive iff BRules at the number of its living neighbours (out of 8) is true
	// A living cell stays alive iff SRules at the number of its living neighbours (out of 8) is true
	// These rules do NOT count a live cell as its own neighbour.
	BRules Ruleset
	SRules Ruleset

	// The degree to which the game is "zoomed in". For example, with a scale factor of 3, each game board cell is drawn
	// as a 3x3 square on a fullscreen window. Note that each cell still corresponds to one pixel in pixels.
	scaleFactor int

	// The percent (0.0 to 100.0) chance any given board cell will initialize as alive.
	avgStartingLiveCellPercentage float64

	// When paused, the simulation doesn't run and a settings change UI is displayed.
	isPaused bool
}

func (g *Game) Update() error {
	// Handle input not handled by the pause menu UI.
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.Restart()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.isPaused = !g.isPaused

		// The user has left the splash screen.
		g.ui.shouldDisplaySlashScreen = false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) && ebiten.IsKeyPressed(ebiten.KeyShift) {
		if ebiten.FPSMode() == ebiten.FPSModeVsyncOffMaximum {
			ebiten.SetFPSMode(ebiten.FPSModeVsyncOn)
		} else {
			ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)
		}
	}

	g.ui.handleInput(g.isPaused)

	if g.isPaused {
		return nil
	}

	// Update the game board.
	// We do this more efficiently by copying the board state into a buffer and modifying only those cells in the buffer
	// which are changing state (becoming alive or dying).
	copy(g.buffer, g.worldGrid)
	for i := 1; i <= g.gridY; i++ {
		for j := 1; j <= g.gridX; j++ {
			// Getting the "2D g.worldGrid[i][j]" index from the 1D slice. +2 because of the board edge border.
			ind := i*(g.gridX+2) + j
			val := g.worldGrid[ind]

			if val&1 == 0 && g.BRules[val>>1] { // Checking if the cell is becoming alive. val&1 == 0 ensures that this
				// cell was dead previously, and val>>1 gets the number of live neighbours.

				g.buffer[ind] |= 1 // Set the last bit to 1 to indicate that this cell is now alive.
				for a := -1; a <= 1; a++ {
					for b := -1; b <= 1; b++ {
						// Update all the neighbours and also this cell. Adding 2 adds to the bit-shifted
						// neighbour-count part of the value.
						g.buffer[(i+a)*(g.gridX+2)+j+b] += 2

					}
				}
				// -1 because i and j and 1-indexed due to the border, which the game board image doesn't have.
				g.pixels.Set(j-1, i-1, color.White)

			} else if val&1 == 1 && !g.SRules[val>>1-1] { // Checking if the cell is becoming dead. val&1 == 1 ensures
				// that this cell was alive previously. Since this cell is alive, val>>1 is the one more than the number
				// of live neighbours, as this cell is also counted in val>>1, so we check val>>1-1 in SRules.

				// The rest of this case is analogous to the cell becoming alive case.
				g.buffer[ind] -= 1 // Set the last bit to 0 to indicate that this cell is now dead.
				for a := -1; a <= 1; a++ {
					for b := -1; b <= 1; b++ {
						g.buffer[(i+a)*(g.gridX+2)+j+b] -= 2
					}
				}
				g.pixels.Set(j-1, i-1, color.Black)
			}

		}
	}
	// Make the buffer making our changes in the new world grid. We can't do the changes in place, using only one board,
	// since if we change a cell's state, that will effect how the neighbouring cells will get updated.
	copy(g.worldGrid, g.buffer)

	return nil
}

func (g *Game) Restart() {
	// Change the rules, scale factor and initial live cell percentage to the ones selected in the UI.
	g.BRules = g.ui.selectedBRules
	g.SRules = g.ui.selectedSRules

	g.scaleFactor = g.ui.getScaleFactor()
	g.avgStartingLiveCellPercentage = g.ui.selectedLiveCellPercent

	// Reset the board with the new paremeters.
	g.initializeBoard()
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draw the game board image in (0, 0) and scaling by the scale factor to fill the whole screen.
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(float64(g.scaleFactor), float64(g.scaleFactor))
	screen.DrawImage(g.pixels, options)

	// To dim the simulation in the background so that the pause menu UI is visible.
	if g.isPaused {
		screen.DrawImage(g.transparencyOverlay, nil)
	}

	// Draw UI text elements.
	g.ui.Draw(screen, g.isPaused)
}

// Returns the size of the screen we want to be rendering to.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ebiten.ScreenSizeInFullscreen()
}

// Initializes the initial simulation state. Called only once, before ebiten.runGame(g).
func (g *Game) initializeState() {
	// Currently seed is always 0, kind of redundant.
	r = rand.New(rand.NewSource(SEED))

	// Initial rule set is just Conway's Game of Life.
	g.BRules = Ruleset{}
	g.BRules[3] = true
	g.SRules = Ruleset{}
	g.SRules[2] = true
	g.SRules[3] = true

	g.avgStartingLiveCellPercentage = 50.0

	g.isPaused = true

	// Start the simulation at the second smallest scale factor, i.e. slightly zoomed in. For most screen resolutions
	// this will be a 2x zoom (since both screen height and width are usually even).
	initialScaleIndex := 1

	// Initialize UI, get the chosen scale factor from it.
	g.ui.initialize(g.BRules, g.SRules, g.avgStartingLiveCellPercentage, initialScaleIndex)
	g.scaleFactor = g.ui.getScaleFactor()

	x, y := ebiten.ScreenSizeInFullscreen()
	g.gridX = x / g.scaleFactor
	g.gridY = y / g.scaleFactor

	// The transparency overlay has a constant size corresponding to the max screen size, so that we can always use this
	// same overlay instead of creating new ones when the scale factors changes.
	g.transparencyOverlay = ebiten.NewImage(x, y)
	g.transparencyOverlay.Fill(color.RGBA{0, 0, 0, 255 * 3 / 4}) // black but not completely opaque
}

// Initializes the simulation board, filling it with cells randomly, and creates the corresponding initial simulation
// image. The chance of a given cell being set to alive is given by g.avgStartingLiveCellPercentage.
func (g *Game) initializeBoard() {

	x, y := ebiten.ScreenSizeInFullscreen()
	g.gridX = x / g.scaleFactor
	g.gridY = y / g.scaleFactor

	g.pixels = ebiten.NewImage(g.gridX, g.gridY)
	g.pixels.Fill(color.Black)
	g.worldGrid = make([]uint8, (g.gridX+2)*(g.gridY+2))
	g.buffer = make([]uint8, (g.gridX+2)*(g.gridY+2))

	for i := 1; i <= g.gridY; i++ {
		for j := 1; j <= g.gridX; j++ {
			if int(r.Int63n(100000)) < int(1000*g.avgStartingLiveCellPercentage) { // Cell becomes alive.
				g.worldGrid[i*(g.gridX+2)+j] |= 1
				g.pixels.Set(j-1, i-1, color.White)
				// Update live neighbour counts in the cells affected by this cell becoming alive.
				for a := -1; a <= 1; a++ {
					for b := -1; b <= 1; b++ {
						g.worldGrid[(i+a)*(g.gridX+2)+j+b] += 2
					}
				}
			}
		}
	}
}

func main() {
	// Set the right window properties. Should give pixel perfect image in fullscreen.
	ebiten.SetFullscreen(true)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSize(ebiten.ScreenSizeInFullscreen())

	// By default, update and render as fast as possible. Currently this makes the simulation speed "pulsate" slightly,
	// maybe because of GC activity?
	ebiten.SetTPS(ebiten.SyncWithFPS)
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)

	ebiten.SetWindowTitle("go-llca")

	g := &Game{}
	g.initializeState() // Only called here.
	g.initializeBoard()

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
