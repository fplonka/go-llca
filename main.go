package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/exp/slices"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// cool ones: 2456/5678, 3456/5678, 456/1234, 45678/2345, 345/4567 @156-165, 36/125, 3/123456, 3/1347
var (
	BRules                        = []uint8{3}
	SRules                        = []uint8{2, 3}
	BRulesBuffer                  []uint8
	SRulesBuffer                  []uint8
	possibleScaleFactors          []int
	scaleFactor                   int     = 1
	scaleFactorIndex              int     = 0
	avgStartingLiveCellPercentage float64 = 50.0 // # out of 100
	gensToRun                             = 100 * 60
	seed                          int64   = 0
	uiFont                        font.Face
	isFpsVisible                  bool = true
)

const ()

func init() {
	rand.New(rand.NewSource(seed))

	BRulesBuffer = make([]uint8, len(BRules))
	copy(BRulesBuffer, BRules)
	SRulesBuffer = make([]uint8, len(SRules))
	copy(SRulesBuffer, SRules)
	scaleFactorIndex = scaleFactor

	fontBytes, err := os.ReadFile("fonts/JetBrainsMono-Medium.ttf")
	if err != nil {
		log.Fatal(err)
	}
	tt, err := opentype.Parse(fontBytes)
	if err != nil {
		log.Fatal(err)
	}
	screenX, screenY := ebiten.ScreenSizeInFullscreen()
	uiFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    11,
		DPI:     math.Sqrt(float64(screenX*screenY) / 100),
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}

	possibleScaleFactors = []int{}
	smallerDimension := int(math.Min(float64(screenX), float64(screenY)))
	for i := 1; i <= smallerDimension; i++ {
		if screenX%i == 0 && screenY%i == 0 {
			possibleScaleFactors = append(possibleScaleFactors, i)
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

type EditState bool

const (
	ChangingBirthRules   EditState = false
	ChangingSurviveRules EditState = true
)

type Game struct {
	worldGrid           []uint8
	buffer              []uint8
	gridX, gridY        int
	transparencyOverlay *ebiten.Image
	pixels              *ebiten.Image
	generation          int
	name                string
	isPaused            bool
	editState           EditState
}

func updateRules(rules []uint8, nums []uint8) []uint8 {
	res := []uint8{}

	all := []uint8{}
	all = append(all, nums...)
	all = append(all, rules...)

	for _, v := range all {
		// add only those nums which appear in exactly one of the two slices
		if slices.Contains(rules, v) != slices.Contains(nums, v) {
			res = append(res, v)
		}
	}

	return res
}

func (g *Game) updatePaused() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.editState = !g.editState
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		if g.editState == ChangingBirthRules {
			BRulesBuffer = []uint8{}
		} else {
			SRulesBuffer = []uint8{}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			avgStartingLiveCellPercentage += 0.1
		} else {
			avgStartingLiveCellPercentage += 1.0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			avgStartingLiveCellPercentage -= 0.1
		} else {
			avgStartingLiveCellPercentage -= 1.0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketRight) && scaleFactorIndex+1 < len(possibleScaleFactors) {
		scaleFactorIndex++
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketLeft) && scaleFactorIndex-1 >= 0 {
		scaleFactorIndex--
	}

	avgStartingLiveCellPercentage = math.Max(avgStartingLiveCellPercentage, 0.0)
	avgStartingLiveCellPercentage = math.Min(avgStartingLiveCellPercentage, 100.0)

	// TOOD: check if 0 is handled correctly in updates
	nums := []uint8{}
	keys := []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8}
	for _, key := range keys {
		if inpututil.IsKeyJustPressed(key) {
			nums = append(nums, uint8(int(key)-int(ebiten.Key0)))
		}
	}

	if g.editState == ChangingBirthRules {
		BRulesBuffer = updateRules(BRulesBuffer, nums)
	} else {
		SRulesBuffer = updateRules(SRulesBuffer, nums)
	}

	sort.Slice(BRulesBuffer, func(i, j int) bool { return BRulesBuffer[i] < BRulesBuffer[j] })
	sort.Slice(SRulesBuffer, func(i, j int) bool { return SRulesBuffer[i] < SRulesBuffer[j] })
	return nil
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.Restart()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.isPaused = !g.isPaused
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			if ebiten.FPSMode() == ebiten.FPSModeVsyncOffMaximum {
				ebiten.SetFPSMode(ebiten.FPSModeVsyncOn)
			} else {
				ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)
			}
		} else {
			isFpsVisible = !isFpsVisible
		}
	}
	if g.isPaused {
		return g.updatePaused()
	}

	g.generation++
	if g.generation == gensToRun {
		os.Exit(0)
	}

	copy(g.buffer, g.worldGrid)
	for i := 1; i <= g.gridY; i++ {
		for j := 1; j <= g.gridX; j++ {
			ind := i*(g.gridX+2) + j
			if g.worldGrid[ind] == 0 {
				continue
			}
			val := g.worldGrid[ind]
			if becomesAlive(val) { // cell becomes alive
				g.buffer[ind] |= 1
				for a := -1; a <= 1; a++ {
					for b := -1; b <= 1; b++ {
						g.buffer[(i+a)*(g.gridX+2)+j+b] += 2
					}
				}
				g.pixels.Set(j-1, i-1, color.White)
			} else if becomesDead(val) { // cell dies
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
	BRules = make([]uint8, len(BRulesBuffer))
	copy(BRules, BRulesBuffer)

	SRules = make([]uint8, len(SRulesBuffer))
	copy(SRules, SRulesBuffer)
	g.generation = 0

	scaleFactor = possibleScaleFactors[scaleFactorIndex]
	g.Init()
}

func (g *Game) Draw(screen *ebiten.Image) {
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(float64(scaleFactor), float64(scaleFactor))
	screen.DrawImage(g.pixels, options)

	if g.isPaused {
		screen.DrawImage(g.transparencyOverlay, options)

		lines := []string{
			"%vbirth rules: %v",
			"%vsurvival rules: %v",
			"inital percentage of live cells: %.1f",
			"scale factor: %v",
			"%.2f FPS, generation %v",
			"",
			"use number keys to modify cell %v rules (press TAB to switch, C to clear)",
			"use - and + to change initial live cell percentage (hold SHIFT for a smaller increment)",
			"use [ and ] to change scale factor",
			"press SPACE to pause/unpause or R to restart with new settings",
			"press F to toggle FPS visibility and SHIFT+F to toggle FPS cap"}

		infoFormatString := strings.Join(lines, "\n")

		changeType := "BIRTH"
		if g.editState == ChangingSurviveRules {
			changeType = "SURVIVAL"
		}

		birthRulesIndicator := "*"
		survivalRulesIndicator := ""
		if g.editState == ChangingSurviveRules {
			birthRulesIndicator = ""
			survivalRulesIndicator = "*"
		}

		birthRules := ""
		for _, v := range BRulesBuffer {
			birthRules += strconv.Itoa(int(v)) + " "
		}
		survivalRules := ""
		for _, v := range SRulesBuffer {
			survivalRules += strconv.Itoa(int(v)) + " "
		}

		infoString := fmt.Sprintf(infoFormatString, birthRulesIndicator, birthRules, survivalRulesIndicator, survivalRules,
			avgStartingLiveCellPercentage, possibleScaleFactors[scaleFactorIndex], ebiten.ActualFPS(), g.generation, changeType)

		text.Draw(screen, infoString, uiFont, 20, 40, color.White)
	} else if isFpsVisible {
		text.Draw(screen, fmt.Sprintf("%.2f FPS", ebiten.ActualFPS()), uiFont, 20, 40, color.White)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ebiten.ScreenSizeInFullscreen()
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
	g.buffer = make([]uint8, (g.gridX+2)*(g.gridY+2))
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

	g.transparencyOverlay = ebiten.NewImage(g.gridX, g.gridY)
	g.transparencyOverlay.Fill(color.RGBA{0, 0, 0, 255 * 3 / 4}) // black but not completely opaque
}

func main() {
	ebiten.SetFullscreen(false)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSize(ebiten.ScreenSizeInFullscreen())
	ebiten.SetTPS(ebiten.SyncWithFPS)
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)
	ebiten.SetWindowTitle("go-llca")
	// ebiten.SetFPSMode(ebiten.FPSModeVsyncOn)
	g := &Game{}
	g.Init()
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

// TODO
// - UI:
// 	- fps (capped vs uncapped)
// 	- seed
// 	- hide fps / generation bar
//  - control info

// holding key down to change scale/live%

// BOUNDS ON SCALE FACTOR AND CELL PERCENTAGE

// cute idea: CA-based evolution
// have cells randomly mutate their rules when being born sometimes (?)
