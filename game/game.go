package game

import (
	"image/color"
	"math/rand"
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Random number source for game board initialization.
var r *rand.Rand

const (
	// Seed for the random number source. r is seeded only once and is not reinitialized with the seed before every run, so
	// the order in which simulation runs are started will affect their initial board states.
	SEED = 0

	// The maximum number of go routines which can be spawned by the application. Used to limit to the concurrency when
	// updating the board.
	MAX_ROUTINES = 64
)

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
	worldGrid    []int8
	buffer       []int8
	gridX, gridY int

	// The image we draw to the screen during the draw step. Dead cells are black, live cells are white.
	img *ebiten.Image

	// The raw pixels of our image. Each image pixel is represented as 4 bytes in pixels (RGBA channels), so we must
	// have len(pixels) = 4 * (img width) * (img height).
	pixels []byte

	// Semi-transparent image to cover and "dim" the simulation image when paused.
	transparencyOverlay *ebiten.Image

	// Game rules.
	// A dead cell becomes alive iff bRules at the number of its living neighbours (out of 8) is true
	// A living cell stays alive iff SRules at the number of its living neighbours (out of 8) is true
	// These rules do NOT count a live cell as its own neighbour.
	bRules Ruleset
	sRules Ruleset

	// The degree to which the game is "zoomed in". For example, with a scale factor of 3, each game board cell is drawn
	// as a 3x3 square on a fullscreen window. Note that each cell still corresponds to one pixel in pixels.
	scaleFactor int

	// The percent (0.0 to 100.0) chance any given board cell will initialize as alive.
	avgStartingLiveCellPercentage float64

	// When paused, the simulation doesn't run and a settings change UI is displayed.
	isPaused bool

	// Struct managing functionality related to saving frames of the simulation to a .gif file.
	gifSaver GifSaver
	isSaving bool
}

// Update board rows from i to j inclusive.
func (g *Game) updateRange(minY, maxY int, wg *sync.WaitGroup) {
	defer wg.Done()

	// If we're still not at the goroutine limit and the range we're updating is at least 4 rows, do a concurrent divide
	// and conquer step.
	if runtime.NumGoroutine() < MAX_ROUTINES && maxY-minY+1 >= 4 {
		// Split rows into two parts and update them recursively. We leave a two pixel border between the updated ranges
		// because when updating a range, we also update neighbour board values, and so when updating touching ranges
		// in parallel we could try to update the same value at the same time.
		midY := (minY + maxY) / 2
		var newWg sync.WaitGroup
		newWg.Add(2)
		go g.updateRange(minY, midY-1, &newWg)
		go g.updateRange(midY+2, maxY, &newWg)

		// We need a new syncgroup so we can wait until the two recursive calls are done and then sequentially update
		// the 2-pixel border.
		newWg.Wait()
		// The waitgroup used when updating the border isn't actually used for anything, we just reuse newWg for
		// simplicity.
		newWg.Add(1)
		g.updateRange(midY, midY+1, &newWg)

	} else {
		// Update the game board.
		// We do this more efficiently by copying the board state into a buffer and modifying only those cells in the
		// buffer which are changing state (becoming alive or dying).
		for i := minY; i <= maxY; i++ {
			for j := 1; j <= g.gridX; j++ {
				// Getting the "2D g.worldGrid[i][j]" index from the 1D slice. +2 because of the board edge border.
				ind := i*(g.gridX+2) + j
				val := g.worldGrid[ind]

				if val&1 == 0 && g.bRules[val>>1] { // Checking if the cell is becoming alive. val&1 == 0 ensures that this
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
					setPixel(g.pixels, g.gridX, j-1, i-1, false)

				} else if val&1 == 1 && !g.sRules[val>>1-1] { // Checking if the cell is becoming dead. val&1 == 1 ensures
					// that this cell was alive previously. Since this cell is alive, val>>1 is the one more than the number
					// of live neighbours, as this cell is also counted in val>1, so we check val>>1-1 in SRules.

					// The rest of this case is analogous to the cell becoming alive case.
					g.buffer[ind] -= 1 // Set the last bit to 0 to indicate that this cell is now dead.
					for a := -1; a <= 1; a++ {
						for b := -1; b <= 1; b++ {
							g.buffer[(i+a)*(g.gridX+2)+j+b] -= 2
						}
					}
					setPixel(g.pixels, g.gridX, j-1, i-1, true)
				}

			}
		}
	}
}

func (g *Game) Update() error {
	// Handle input not handled by the pause menu UI.
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.restart()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		// After this frame, the user has entered/left the pause menu.
		defer func() { g.isPaused = !g.isPaused }()

		// A SHIFT+SPACE press when paused, so we start saving.
		if g.isPaused && ebiten.IsKeyPressed(ebiten.KeyShift) && !g.isSaving {
			g.isSaving = true
			g.ui.shouldDisplayRecordingText = true
			g.gifSaver = newGifSaver(g.bRules, g.sRules)

			// Return instead of doing an update step, since saving the frame happens in Draw() and so if we update
			// before that we will skip one frame of the initial random board state.
			return nil
		}

		// A SPACE press when not paused and saving, so we stop saving.
		if !g.isPaused && g.isSaving {
			g.isSaving = false
			g.ui.shouldDisplayRecordingText = false
			go func() {
				// Write to file concurrently so as to not cause a freeze, as this can take a few seconds, and tell the
				// UI to indicate that we're saving.
				g.ui.shouldDisplayWritingToFileText = true
				g.gifSaver.writeToFile()
				g.ui.shouldDisplayWritingToFileText = false
			}()
		}

		// The user has left the splash screen.
		g.ui.shouldDisplaySlashScreen = false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) && ebiten.IsKeyPressed(ebiten.KeyShift) {
		// Toggle FPS mode between uncapped and vsynced.
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
	var wg sync.WaitGroup
	wg.Add(1)
	go g.updateRange(1, g.gridY, &wg)
	wg.Wait()
	copy(g.worldGrid, g.buffer)

	return nil
}

func (g *Game) restart() {
	// Change the rules, scale factor and initial live cell percentage to the ones selected in the UI.
	g.bRules = g.ui.selectedBRules
	g.sRules = g.ui.selectedSRules

	g.scaleFactor = g.ui.getScaleFactor()
	g.avgStartingLiveCellPercentage = g.ui.selectedLiveCellPercent

	// Reset the board with the new paremeters.
	g.InitializeBoard()
}

func (g *Game) Draw(screen *ebiten.Image) {
	// We write our board pixels to our game image, and then draw this image scaled in (0, 0) scaling by the scale
	// factor to fill the whole screen.
	g.img.WritePixels(g.pixels)
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(float64(g.scaleFactor), float64(g.scaleFactor))
	screen.DrawImage(g.img, options)

	// To dim the simulation in the background so that the pause menu UI is visible.
	if g.isPaused {
		screen.DrawImage(g.transparencyOverlay, nil)
	}

	if g.isSaving {
		// This could also receive screen instead of g.img, to always save full resolution gifs, but saving higher
		// resolution GIFs is slow and takes up a lot of space, so we save unscaled smaller GIFs. A user can always
		// manually upscale them if desired.
		g.gifSaver.saveFrame(g.img)
	}

	// Draw UI text elements.
	g.ui.Draw(screen, g.isPaused)
}

// Sets a pixel at a given index to either black or white.
func setPixel(pixels []byte, gridX, x, y int, isBlack bool) {
	color := []byte{255, 255, 255, 255}
	if isBlack {
		color = []byte{0, 0, 0, 255}
	}
	ind := 4 * (y*gridX + x)
	copy(pixels[ind:ind+4], color)
}

// Returns the size of the screen we want to be rendering to.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ebiten.ScreenSizeInFullscreen()
}

// Initializes the initial simulation state. Called only once, before ebiten.runGame(g).
func (g *Game) InitializeState() {
	// Currently seed is always 0, kind of redundant.
	r = rand.New(rand.NewSource(SEED))

	// Initial rule set is just Conway's Game of Life.
	g.bRules = Ruleset{}
	g.bRules[3] = true
	g.sRules = Ruleset{}
	g.sRules[2] = true
	g.sRules[3] = true

	g.avgStartingLiveCellPercentage = 50.0

	g.isPaused = true
	g.isSaving = false

	// Start the simulation at the second smallest scale factor, i.e. slightly zoomed in. For most screen resolutions
	// this will be a 2x zoom (since both screen height and width are usually even).
	initialScaleIndex := 1

	// Initialize UI, get the chosen scale factor from it.
	g.ui.initialize(g.bRules, g.sRules, g.avgStartingLiveCellPercentage, initialScaleIndex)
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
func (g *Game) InitializeBoard() {
	x, y := ebiten.ScreenSizeInFullscreen()
	g.gridX = x / g.scaleFactor
	g.gridY = y / g.scaleFactor

	g.img = ebiten.NewImage(g.gridX, g.gridY)
	g.img.Fill(color.Black)

	// RGBA channels, so 4 bytes per image pixel.
	g.pixels = make([]byte, 4*g.gridX*g.gridY)

	// Make all pixels black initially.
	for i := 0; i < g.gridY; i++ {
		for j := 0; j < g.gridX; j++ {
			setPixel(g.pixels, g.gridX, j, i, true)
		}
	}

	g.worldGrid = make([]int8, (g.gridX+2)*(g.gridY+2))
	g.buffer = make([]int8, (g.gridX+2)*(g.gridY+2))
	for i := 1; i <= g.gridY; i++ {
		for j := 1; j <= g.gridX; j++ {
			if int(r.Int63n(100000)) < int(1000*g.avgStartingLiveCellPercentage) { // Cell becomes alive.
				g.worldGrid[i*(g.gridX+2)+j] |= 1
				// g.pixels.Set(j-1, i-1, color.White)
				setPixel(g.pixels, g.gridX, j-1, i-1, false)
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
