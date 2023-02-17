package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
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

const (
	FONT_PATH     = "fonts/JetBrainsMono-Medium.ttf"
	FONT_SIZE     = 13
	MARGIN        = 20
	SHADOW_OFFSET = 2
)

type UI struct {
	// Current state of the rules being edited in the pause menu.
	selectedBRules []uint8
	selectedSRules []uint8

	// Pointer to either selectedBRules or selectedSRules, depending on which is being edited.
	rulesBeingChanged *[]uint8

	selectedLiveCellPercent float64

	// Scale factors possible given the screen dimensions (they must divide both fullscreen width and height)
	// and the index of the scale factor currently selected in the pause menu.
	possibleScaleFactors []int
	scaleFactorIndex     int

	// FPS visibility during simulation.
	isFpsVisible bool

	// Font face for UI text rendering.
	uiFont font.Face
}

func (ui *UI) initialize(BRules, SRules []uint8, liveCellPercent float64, initialScaleIndex int) {
	// Needs BRules and SRules to make the initial rule buffers match the "default" rules of the simulation
	// which shows when you start the program and haven't changed anything yet. Same for the initial live cell percentage.
	ui.selectedBRules = make([]uint8, len(BRules))
	copy(ui.selectedBRules, BRules)
	ui.selectedSRules = make([]uint8, len(SRules))
	copy(ui.selectedSRules, SRules)
	ui.selectedLiveCellPercent = liveCellPercent

	ui.rulesBeingChanged = &ui.selectedBRules
	ui.isFpsVisible = true
	ui.scaleFactorIndex = initialScaleIndex

	// Initialize possible scale factors, i.e. find the integers which divide both the screen width and height.
	ui.possibleScaleFactors = []int{}
	screenX, screenY := ebiten.ScreenSizeInFullscreen()
	smallerDimension := intMin(screenX, screenY)
	for i := 1; i <= smallerDimension; i++ {
		if screenX%i == 0 && screenY%i == 0 {
			ui.possibleScaleFactors = append(ui.possibleScaleFactors, i)
		}
	}

	// Make the transparency overlay a black, semi-transparent rectangle. This is to "dim" the paused simulation
	// when looking at the pause menu.

	ui.uiFont = loadFontFace(FONT_PATH)
}

// Loads a font face from a .ttf font file.
func loadFontFace(path string) font.Face {
	fontBytes, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	tt, err := opentype.Parse(fontBytes)
	if err != nil {
		log.Fatal(err)
	}
	screenX, screenY := ebiten.ScreenSizeInFullscreen()
	uiFont, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    FONT_SIZE,
		DPI:     math.Sqrt(float64(screenX*screenY) / 100), // slightly cursed DPI approximation
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}
	return uiFont
}

func (ui *UI) handleInput(isGamePaused bool) {
	// If the simulation is running, then the only UI-related input to handle is FPS visibility.
	if inpututil.IsKeyJustPressed(ebiten.KeyF) && !ebiten.IsKeyPressed(ebiten.KeyShift) {
		ui.isFpsVisible = !ui.isFpsVisible
	}
	if !isGamePaused {
		return
	}

	// Toggle between editing birth vs survival rules on TAB press.
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		if ui.rulesBeingChanged == &ui.selectedBRules {
			ui.rulesBeingChanged = &ui.selectedSRules
		} else {
			ui.rulesBeingChanged = &ui.selectedBRules
		}
	}

	// Clear selected rules on C press.
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		if ui.rulesBeingChanged == &ui.selectedBRules {
			ui.selectedBRules = []uint8{}
		} else {
			ui.selectedSRules = []uint8{}
		}
	}

	// Change initial live cell percentage value, adjusting the increment if
	// SHIFT or CONTROL are pressed to allow for finer control.
	// Ideally this would be done with a GUI but that's nontrivial in Ebiten.
	delta := 10.0
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		delta = 1.0
	} else if ebiten.IsKeyPressed(ebiten.KeyControl) {
		delta = 0.1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) {
		ui.selectedLiveCellPercent += delta
	} else if inpututil.IsKeyJustPressed(ebiten.KeyMinus) {
		ui.selectedLiveCellPercent -= delta
	}

	// Change selected scale factor to the next larger/smaller scale factor.
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketRight) {
		ui.scaleFactorIndex++
	} else if inpututil.IsKeyJustPressed(ebiten.KeyBracketLeft) {
		ui.scaleFactorIndex--
	}

	// Handle the input for changing the selected rule set, i.e. number key presses.
	ui.handleNumberKeys()

	// Clamp live cell percentage and scale factor index to legal values
	ui.selectedLiveCellPercent = clamp(0.0, 100.0, ui.selectedLiveCellPercent)
	ui.scaleFactorIndex = clamp(0, len(ui.possibleScaleFactors)-1, ui.scaleFactorIndex)

	// Sort the rule sets to ensure they're always presented in order.
	sort.Slice(ui.selectedBRules, func(i, j int) bool { return ui.selectedBRules[i] < ui.selectedBRules[j] })
	sort.Slice(ui.selectedSRules, func(i, j int) bool { return ui.selectedSRules[i] < ui.selectedSRules[j] })
}

// CURRENTLY BORKEN
func (ui *UI) handleNumberKeys() {
	// Figure out which number key is being pressed. Handles the possibility of multiple at once, which is possible but rare.
	nums := []uint8{}
	keys := []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8}
	for _, key := range keys {
		if inpututil.IsKeyJustPressed(key) {
			nums = append(nums, uint8(int(key)-int(ebiten.Key0)))
		}
	}

	// The new rule set values are those which are in the rule set or are having their number key pressed BUT NOT BOTH.
	// So if the rule set contained 3 and 3 was pressed, 3 is removed. If 3 had not been in the rule set, it would have been added.
	newRules := []uint8{}
	all := []uint8{}
	all = append(all, nums...)
	all = append(all, *ui.rulesBeingChanged...)
	for _, v := range all {
		// add only those nums which appear in exactly one of the two slices
		if slices.Contains(*ui.rulesBeingChanged, v) != slices.Contains(nums, v) {
			newRules = append(newRules, v)
		}
	}

	*ui.rulesBeingChanged = make([]uint8, len(newRules))
	copy(*ui.rulesBeingChanged, newRules)
}

func (ui *UI) Draw(screen *ebiten.Image, isGamePaused bool) {
	if isGamePaused {
		lines := []string{
			"%vbirth rules: %v",
			"%vsurvival rules: %v",
			"inital percentage of live cells: %.1f",
			"board resolution: %v (%vx zoom)",
			"",
			"use number keys to modify cell %v rules (press TAB to switch, C to clear)",
			"use - and + to change initial live cell percentage (hold SHIFT/CTRL for smaller/smallest increment)",
			"use [ and ] to change resolution",
			"press F to toggle FPS visibility and SHIFT+F to toggle FPS cap",
			"",
			"press SPACE to pause/unpause or R to restart with new settings"}

		infoFormatString := strings.Join(lines, "\n")

		changeType := "BIRTH"
		if ui.rulesBeingChanged == &ui.selectedSRules {
			changeType = "SURVIVAL"
		}

		birthRulesIndicator := "*"
		survivalRulesIndicator := ""
		if ui.rulesBeingChanged == &ui.selectedSRules {
			birthRulesIndicator = ""
			survivalRulesIndicator = "*"
		}

		birthRules := ""
		for _, v := range ui.selectedBRules {
			birthRules += strconv.Itoa(int(v)) + " "
		}
		survivalRules := ""
		for _, v := range ui.selectedSRules {
			survivalRules += strconv.Itoa(int(v)) + " "
		}

		screenX, screenY := ebiten.ScreenSizeInFullscreen()
		scaleFactor := ui.getScaleFactor()
		resolution := fmt.Sprintf("%vx%v", screenX/scaleFactor, screenY/scaleFactor)

		infoString := fmt.Sprintf(infoFormatString, birthRulesIndicator, birthRules, survivalRulesIndicator, survivalRules,
			ui.selectedLiveCellPercent, resolution, ui.getScaleFactor(), changeType)

		boundsFirstLine := text.BoundString(ui.uiFont, lines[0])
		boundsAllLines := text.BoundString(ui.uiFont, infoString)

		infoX := MARGIN
		infoY := screenY - boundsAllLines.Dy() - MARGIN + boundsFirstLine.Dy()

		text.Draw(screen, infoString, ui.uiFont, infoX+SHADOW_OFFSET, infoY+SHADOW_OFFSET, color.Black)
		text.Draw(screen, infoString, ui.uiFont, infoX, infoY, color.White)
	}

	if ui.isFpsVisible {
		// Draw first with black to get a slight "shadow" which helps with readability

		fpsString := fmt.Sprintf("%.2f FPS", ebiten.ActualFPS())
		bounds := text.BoundString(ui.uiFont, fpsString)

		screenX, _ := screen.Size()
		fpsX := screenX - bounds.Dx() - MARGIN
		fpsY := bounds.Dy() + MARGIN

		text.Draw(screen, fmt.Sprintf("%.2f FPS", ebiten.ActualFPS()), ui.uiFont, fpsX+SHADOW_OFFSET, fpsY+SHADOW_OFFSET, color.Black)
		text.Draw(screen, fmt.Sprintf("%.2f FPS", ebiten.ActualFPS()), ui.uiFont, fpsX, fpsY, color.White)
	}

}

func (ui *UI) getScaleFactor() int {
	return ui.possibleScaleFactors[ui.scaleFactorIndex]
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clamp[T int | float64](min, max, a T) T {
	if a < min {
		return min
	}
	if a > max {
		return max
	}
	return a
}
