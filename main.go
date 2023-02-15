package main

import (
	"fmt"
	// "image"
	"image/color"
	"image/png"
	"log"
	"math/rand"
	"os"
	"strconv"

	// "time"B

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// cool ones: 2456/5678, 3456/5678, 456/1234, 45678/2345, 345/4567 @156-165, 36/125, 3/123456, 3/1347
var (
	BRules    = [...]uint8{3}
	SRules    = [...]uint8{2, 3}
	runNumber int
)

const (
	scaleFactor                           = 1
	avgStartingLiveCellPercentage float64 = 6.0 // # out of 100
	saveRun                               = false
	gensToRun                             = 10 * 60
)

// good runs: 6, 13, some other one, 14

func init() {
	// rand.Seed(time.Now().UnixNano())
	rand.Seed(0)

	if saveRun {
		path := "run_count"
		dat, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		runNumber, _ = strconv.Atoi(string(dat))
		runNumber++
		fmt.Printf("run number: %v\n", runNumber)

		err = os.WriteFile(path, []byte(strconv.Itoa(runNumber)), 0o666)
		if err != nil {
			panic(err)
		}

		err = os.Mkdir(fmt.Sprintf("img/%v", runNumber), 0o777)
		if err != nil {
			panic(err)
		}
	}
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

type Game struct {
	worldGrid    []uint8
	gridX, gridY int
	pixels       *ebiten.Image
	generation   int
	name         string
	isPaused     bool
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.isPaused = !g.isPaused
	}
	if g.isPaused {
		return nil
	}
	if saveRun {
		// fmt.Printf("%06v\n", runNumber)
		f, err := os.Create(fmt.Sprintf("img/%v/%v_%06v.png", runNumber, g.name, g.generation))
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if err = png.Encode(f, g.pixels); err != nil {
			log.Printf("failed to encode pixels as png")
		}
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

func (g *Game) Draw(screen *ebiten.Image) {
	screen.DrawImage(g.pixels, nil)
	if !saveRun {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("%.2f gen %v", ebiten.CurrentFPS(), g.generation))
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.gridX, g.gridY
}

func (g *Game) Init() {
	// x, y := ebiten.ScreenSizeInFullscreen()
	x, y := 2560, 1600
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
	ebiten.SetWindowSize(2560, 1600)
	ebiten.SetFullscreen(true)
	ebiten.SetMaxTPS(ebiten.SyncWithFPS)
	// ebiten.SetVsyncEnabled(true)
	ebiten.SetWindowTitle("conway but good")
	fmt.Println(ebiten.ScreenSizeInFullscreen())
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)
	// ebiten.SetFPSMode(ebiten.FPSModeVsyncOn)
	g := &Game{}
	g.Init()
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
