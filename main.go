package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// cool ones: 2456/5678, 3456/5678, 456/1234, 45678/2345, 345/4567 @156-165, 36/125, 3/123456, 3/1347
var (
	BRules                                = []uint8{3}
	SRules                                = []uint8{2, 3}
	scaleFactor                           = 2
	avgStartingLiveCellPercentage float64 = 6.0 // # out of 100
	gensToRun                             = 100 * 60
	seed                          int64   = 0
)

const ()

func init() {
	rand.New(rand.NewSource(seed))
}

func becomesAlive(n uint8) bool {
	if n&1 == 1 {
		return false
	}
	for _, v := range BRules {
		if n == 2*v {
			return true
		}
	}
	return false
}

func becomesDead(n uint8) bool {
	if n&1 == 0 {
		return false
	}
	for _, v := range SRules {
		if n == 2*v+3 {
			return false
		}
	}
	return true
}

type EditState bool

const (
	ChangingBirthRules   EditState = false
	ChangingSurviveRules EditState = true
)

type Game struct {
	worldGrid    []uint8
	gridX, gridY int
	pixels       *ebiten.Image
	generation   int
	name         string
	isPaused     bool
	editState    EditState
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.isPaused = !g.isPaused
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.Restart()
	}
	if g.isPaused {
		if inpututil.IsKeyJustPressed(ebiten.KeyB) {
			g.editState = ChangingBirthRules
		} else if inpututil.IsKeyJustPressed(ebiten.KeyS) {
			g.editState = ChangingSurviveRules
		}

		sort.Slice(BRules, func(i, j int) bool { return BRules[i] < BRules[j] })
		sort.Slice(SRules, func(i, j int) bool { return SRules[i] < SRules[j] })
		return nil
	}
	g.generation++
	if g.generation == gensToRun {
		os.Exit(0)
	}
	buffer := make([]uint8, (g.gridX+2)*(g.gridY+2))
	copy(buffer, g.worldGrid)
	for i := 1; i <= g.gridY; i++ {
		for j := 1; j <= g.gridX; j++ {
			ind := i*(g.gridX+2) + j
			if g.worldGrid[ind] == 0 {
				continue
			}
			val := g.worldGrid[ind]
			if becomesAlive(val) { // cell becomes alive
				buffer[ind] |= 1
				for a := -1; a <= 1; a++ {
					for b := -1; b <= 1; b++ {
						buffer[(i+a)*(g.gridX+2)+j+b] += 2
					}
				}
				g.pixels.Set(j-1, i-1, color.White)
			} else if becomesDead(val) { // cell dies
				buffer[ind] -= 1
				for a := -1; a <= 1; a++ {
					for b := -1; b <= 1; b++ {
						buffer[(i+a)*(g.gridX+2)+j+b] -= 2
					}
				}
				g.pixels.Set(j-1, i-1, color.Black)
			}

		}
	}
	copy(g.worldGrid, buffer)

	return nil
}

func (g *Game) Restart() {
	g.Init()
	g.generation = 0
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.DrawImage(g.pixels, nil)
	if g.isPaused {
		transparencyOverlay := ebiten.NewImage(g.gridX, g.gridY)
		// transparencyOverlay.Fill(color.RGBA64{255, 255, 255, 128})
		c := color.Black
		fmt.Println(c.RGBA())
		transparencyOverlay.Fill(color.RGBA{0, 0, 0, 255 * 3 / 4}) // black but not completely opaque
		screen.DrawImage(transparencyOverlay, nil)

		info := ""
		info += "use number keys to modify cell "
		if g.editState == ChangingBirthRules {
			info += "BIRTH"
		} else {
			info += "SURIVAL"
		}
		info += " rules (press S or B to switch)\n"

		info += "birth rules: "
		for _, v := range BRules {
			info += strconv.Itoa(int(v)) + " "

		}
		info += "\n"
		info += "survive rules: "
		for _, v := range SRules {
			info += strconv.Itoa(int(v)) + " "

		}
		info += "\n"

		info += fmt.Sprintf("%.2f gen %v", ebiten.ActualFPS(), g.generation)

		ebitenutil.DebugPrint(screen, info)

	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.gridX, g.gridY
}

func (g *Game) Init() {
	x, y := ebiten.ScreenSizeInFullscreen()
	g.gridX = x / scaleFactor
	g.gridY = y / scaleFactor

	g.name += "B"
	for _, v := range BRules {
		g.name += strconv.Itoa(int(v))
	}
	g.name += "S"
	for _, v := range SRules {
		g.name += strconv.Itoa(int(v))
	}

	g.pixels = ebiten.NewImage(g.gridX, g.gridY)
	g.pixels.Fill(color.Black)
	g.worldGrid = make([]uint8, (g.gridX+2)*(g.gridY+2))
	for i := 1; i <= g.gridY; i++ {
		for j := 1; j <= g.gridX; j++ {
			if rand.Intn(100000) < int(1000*avgStartingLiveCellPercentage) {
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
	ebiten.SetWindowSize(ebiten.ScreenSizeInFullscreen())
	ebiten.SetFullscreen(true)
	ebiten.SetTPS(ebiten.SyncWithFPS)
	ebiten.SetWindowTitle("go-llca")
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)
	// ebiten.SetFPSMode(ebiten.FPSModeVsyncOn)
	g := &Game{}
	g.Init()
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

// TODO
// - UI:
// 	- automaton rule
// 	- scale
// 	- fps (capped vs uncapped)
//  - random density
// 	- seed
// 	- hide fps / generation bar

// space to pause/unpause
// enter to start a new run with the given settings
// or maybe enter to finish editing s/b rules, and then a button to start?
