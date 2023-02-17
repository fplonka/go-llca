package main

import (
	"image/color"
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// TODO: make this controllable / less hardcoded somehow
const seed = 0

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

	// Dead cells are black, live cells are white. The size of pixels is like that of the boardbut without the border.
	pixels *ebiten.Image

	// Semi-transparent image to cover and "dim" the simulation image when paused.
	transparencyOverlay *ebiten.Image

	// Game rules.
	// A dead cell becomes alive iff the number of its living neighbours (out of 8) is in BRules.
	// A living cell stays alive iff the number of its living neighbours (out of 8) is in SRules
	// These rules do NOT count a live cell as its own neighbour.
	BRules []uint8
	SRules []uint8

	// The degree to which the game is "zoomed in". For example, with a scale factor of 3, each game board cell is drawn
	// as a 3x3 square on a fullscreen window. Note that each cell still corresponds to one pixel in pixels.
	scaleFactor int

	// The percent (0.0 to 100.0) chance any given board cell will initialize as alive.
	avgStartingLiveCellPercentage float64

	// When paused, the simulation doesn't run and a settings change UI is displayed.
	isPaused bool
}

// True iff a cell with the value n becomes alive given these birth rules.
func becomesAlive(n uint8, BRules []uint8) bool {
	// Last bit is alive/dead state: if it's 1, cell is already alive.
	if n&1 == 1 {
		return false
	}
	// n bit shifted to the right by 1 is the number of live neighbors,
	// so if it's in BRules, the cell becomes alive.
	for _, v := range BRules {
		if n>>1 == v {
			return true
		}
	}
	return false
}

// True iff a cell with the value n becomes dead given these survival rules.
func becomesDead(n uint8, SRules []uint8) bool {
	// Last bit is alive/dead state: if it's 0, this cell is already dead.
	if n&1 == 0 {
		return false
	}
	// n>>1 is the number of live neighbours INCLUDING this cell. We subtract 1 to account for that, since we know that
	// this cell is alive.
	for _, v := range SRules {
		if n>>1-1 == v {
			return false
		}
	}
	return true
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.Restart()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.isPaused = !g.isPaused
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

	copy(g.buffer, g.worldGrid)
	for i := 1; i <= g.gridY; i++ {
		for j := 1; j <= g.gridX; j++ {
			ind := i*(g.gridX+2) + j
			if g.worldGrid[ind] == 0 {
				continue
			}
			val := g.worldGrid[ind]
			if becomesAlive(val, g.BRules) { // cell becomes alive
				g.buffer[ind] |= 1
				for a := -1; a <= 1; a++ {
					for b := -1; b <= 1; b++ {
						g.buffer[(i+a)*(g.gridX+2)+j+b] += 2
					}
				}
				g.pixels.Set(j-1, i-1, color.White)
			} else if becomesDead(val, g.SRules) { // cell dies
				g.buffer[ind] -= 1
				for a := -1; a <= 1; a++ {
					for b := -1; b <= 1; b++ {
						g.buffer[(i+a)*(g.gridX+2)+j+b] -= 2
					}
				}
				g.pixels.Set(j-1, i-1, color.Black)
			}

		}
	}
	copy(g.worldGrid, g.buffer)

	return nil
}

func (g *Game) Restart() {
	// Change the rules, scale factor and initial live cell percentage to the ones selected in the UI.
	g.BRules = make([]uint8, len(g.ui.selectedBRules))
	copy(g.BRules, g.ui.selectedBRules)
	g.SRules = make([]uint8, len(g.ui.selectedSRules))
	copy(g.SRules, g.ui.selectedSRules)
	g.scaleFactor = g.ui.getScaleFactor()
	g.avgStartingLiveCellPercentage = g.ui.selectedLiveCellPercent

	// Reset the board with the new paremeters.
	g.initializeBoard()
}

func (g *Game) Draw(screen *ebiten.Image) {
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(float64(g.scaleFactor), float64(g.scaleFactor))
	screen.DrawImage(g.pixels, options)

	if g.isPaused {
		screen.DrawImage(g.transparencyOverlay, nil)
	}

	g.ui.Draw(screen, g.isPaused)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ebiten.ScreenSizeInFullscreen()
}

// Initializes the initial simulation state. Called only once, before ebiten.runGame(g)
func (g *Game) initializeState() {
	// Currently seed is always 0, kind of redundant.
	rand.New(rand.NewSource(seed))

	// Initial rule set is just Conway's Game of Life.
	g.BRules = []uint8{3}
	g.SRules = []uint8{2, 3}

	g.avgStartingLiveCellPercentage = 50.0

	g.isPaused = false

	// Start the simulation at the second smallest scale factor, i.e. slightly zoomed in.
	// For most screen resolutions this will be a 2x zoom (since both screen height and width are usually even).
	initialScaleIndex := 1

	// Initialize UI, get the chosen scale factor from it.
	g.ui.initialize(g.BRules, g.SRules, g.avgStartingLiveCellPercentage, initialScaleIndex)
	g.scaleFactor = g.ui.getScaleFactor()

	x, y := ebiten.ScreenSizeInFullscreen()
	g.gridX = x / g.scaleFactor
	g.gridY = y / g.scaleFactor

	// The transparency overlay has a constant size corresponding to the max screen size, so that we can
	// always use this same overlay instead of creating new ones when the scale factors changes.
	g.transparencyOverlay = ebiten.NewImage(x, y)
	g.transparencyOverlay.Fill(color.RGBA{0, 0, 0, 255 * 3 / 4}) // black but not completely opaque
}

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
			if rand.Intn(100000) < int(1000*g.avgStartingLiveCellPercentage) {
				g.worldGrid[i*(g.gridX+2)+j] |= 1
				g.pixels.Set(j-1, i-1, color.White)
				for a := -1; a <= 1; a++ {
					for b := -1; b <= 1; b++ {
						if i+a >= 1 && i+a <= g.gridY && j+b >= 1 && j+b <= g.gridX {
							g.worldGrid[(i+a)*(g.gridX+2)+j+b] += 2
						}
					}
				}
			}
		}
	}
}

func main() {
	ebiten.SetFullscreen(false)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSize(ebiten.ScreenSizeInFullscreen())
	ebiten.SetTPS(ebiten.SyncWithFPS)
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)
	ebiten.SetWindowTitle("go-llca")
	g := &Game{}
	g.initializeState()
	g.initializeBoard()
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

// TODO:
// holding key down to change scale/live%
// save the given parameter config to a file
// save a given run to a .gif or .mp4

// FINISH COMMENTING STUFF
// go over TODO points and do something about them

// TODO: rules should OBVIOUSLY be bool arrays. Ugh.

// cute idea: CA-based evolution
// have cells randomly mutate their rules when being born sometimes (?)
