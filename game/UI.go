package game

import (
	_ "embed"
	"fmt"
	"image/color"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

const (
	FONT_PATH = "https://github.com/JetBrains/JetBrainsMono/raw/master/fonts/ttf/JetBrainsMono-Medium.ttf"
	FONT_SIZE = 12

	// How many pixels away from the edge of the screen to draw UI elements.
	MARGIN = 20

	// How many pixels to offset the black shadow text from the white foreground text.
	SHADOW_OFFSET = 2
)

// The font used by the UI. Embedded so that the binary can be used without depending on a file or remote asset.
//
//go:embed assets/JetBrainsMono-Medium.ttf
var fontBytes []byte

type UI struct {
	// Current state of the rules being edited in the pause menu.
	selectedBRules Ruleset
	selectedSRules Ruleset

	// Pointer to either selectedBRules or selectedSRules, depending on which is being edited.
	rulesBeingChanged *Ruleset

	selectedLiveCellPercent float64

	// Scale factors possible given the screen dimensions (they must divide both fullscreen width and height)
	// and the index of the scale factor currently selected in the pause menu.
	possibleScaleFactors []int
	scaleFactorIndex     int

	// FPS visibility during simulation.
	isFpsVisible bool

	// True when the application is first started, false afterwards.
	shouldDisplaySlashScreen bool

	shouldDisplayWritingToFileText bool
	shouldDisplayRecordingText     bool

	// Font face for UI text rendering.
	fontFace font.Face

	// Game updates 2^speed * 60 times per second. So speed = 2 gives effective 120FPS, speed = -3 gives 7.5FPS.
	// gets rounded when actually setting the Ticks Per Second).
	speed int
}

func (ui *UI) initialize(BRules, SRules Ruleset, liveCellPercent float64, initialScaleIndex int) {
	// Needs BRules and SRules to make the initial rule buffers match the "default" rules of the simulation which shows
	// when you start the program and haven't changed anything yet. Same for the initial live cell percentage and scale
	// factor index.
	ui.selectedBRules = BRules
	ui.selectedSRules = SRules
	ui.selectedLiveCellPercent = liveCellPercent

	ui.rulesBeingChanged = &ui.selectedBRules
	ui.isFpsVisible = true
	ui.shouldDisplaySlashScreen = true
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

	ui.fontFace = loadFontFace(FONT_PATH)
}

// Returns the font face used by the UI, loaded from the embedded font file byte slice.
func loadFontFace(path string) font.Face {
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
	// Toggle FPS visibility on F press.
	if inpututil.IsKeyJustPressed(ebiten.KeyF) && !ebiten.IsKeyPressed(ebiten.KeyShift) {
		ui.isFpsVisible = !ui.isFpsVisible
	}

	// Adjust update speed on left/right arrow press.
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		ui.speed -= 1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		ui.speed += 1
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
			ui.selectedBRules = Ruleset{}
		} else {
			ui.selectedSRules = Ruleset{}
		}
	}

	// Change initial live cell percentage value, adjusting the increment if SHIFT or CONTROL are pressed to allow for
	// finer control. Ideally this would be done with a GUI but that's nontrivial in Ebiten.
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
}

func (ui *UI) handleNumberKeys() {
	// Figure out which number key is being pressed. Handles the possibility of multiple at once, which is unlikely but
	// possible.
	nums := []uint8{}
	keys := []ebiten.Key{ebiten.Key0, ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8}
	for _, key := range keys {
		if inpututil.IsKeyJustPressed(key) {
			nums = append(nums, uint8(int(key)-int(ebiten.Key0)))
		}
	}

	for _, num := range nums {
		(*ui.rulesBeingChanged)[num] = !(*ui.rulesBeingChanged)[num]
	}
}

func (ui *UI) Draw(screen *ebiten.Image, isGamePaused bool) {
	// Draw the splash screen if needed.
	if ui.shouldDisplaySlashScreen {
		line1 := "go-llca"
		bounds1 := text.BoundString(ui.fontFace, line1)

		line2 := "press SPACE to pause/unpause"
		bounds2 := text.BoundString(ui.fontFace, line2)

		// Line height as an int for centering the text.
		h := ui.fontFace.Metrics().Height.Round()

		screenX, screenY := screen.Size()

		drawTextWithShadow(screen, line1, ui.fontFace, (screenX-bounds1.Dx())/2, (screenY-bounds1.Dy())/2-h)
		drawTextWithShadow(screen, line2, ui.fontFace, (screenX-bounds2.Dx())/2, (screenY-bounds2.Dy())/2+h)

		return
	} else if ui.shouldDisplayWritingToFileText {
		drawTextUpperLeft(screen, "saving gif to file...", ui.fontFace)
	} else if ui.shouldDisplayRecordingText {
		drawTextUpperLeft(screen, "recording...", ui.fontFace)
	}

	if ui.isFpsVisible {
		fpsText := fmt.Sprintf("%.2f FPS (%vx)", ebiten.ActualFPS(), ui.getSpeedup())
		drawTextUpperRight(screen, fpsText, ui.fontFace)
	}

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
			"use ← and → to change speed",
			"press F to toggle FPS visibility",
			"",
			"press SPACE to pause/unpause or R to restart with new settings",
			"to start recording, unpause with SHIFT+SPACE and then pause again with SPACE to stop"}

		infoFormatString := strings.Join(lines, "\n")

		// Says whether we're editing the birth or survival rules.
		changeType := "BIRTH"
		if ui.rulesBeingChanged == &ui.selectedSRules {
			changeType = "SURVIVAL"
		}

		// Show a * in front of the rules which are being edited.
		birthRulesIndicator := "*"
		survivalRulesIndicator := ""
		if ui.rulesBeingChanged == &ui.selectedSRules {
			birthRulesIndicator = ""
			survivalRulesIndicator = "*"
		}

		// Concatenate the rule sets into strings.
		birthRules := ""
		survivalRules := ""
		for i := 0; i <= 8; i++ {
			if ui.selectedBRules[i] {
				birthRules += strconv.Itoa(int(i)) + " "
			}
			if ui.selectedSRules[i] {
				survivalRules += strconv.Itoa(int(i)) + " "
			}
		}

		// Make a string showing the selected board resolution.
		screenX, screenY := ebiten.ScreenSizeInFullscreen()
		scaleFactor := ui.getScaleFactor()
		resolution := fmt.Sprintf("%vx%v", screenX/scaleFactor, screenY/scaleFactor)

		// The pause menu UI is just this one formatted string.
		infoString := fmt.Sprintf(infoFormatString, birthRulesIndicator, birthRules, survivalRulesIndicator, survivalRules,
			ui.selectedLiveCellPercent, resolution, ui.getScaleFactor(), changeType)

		// Because text.Draw() is weird about positioning, we use the height of the first line to offset the y position
		// of the UI text.
		boundsFirstLine := text.BoundString(ui.fontFace, lines[0])
		boundsAllLines := text.BoundString(ui.fontFace, infoString)
		infoX := MARGIN
		infoY := screenY - boundsAllLines.Dy() - MARGIN + boundsFirstLine.Dy()

		drawTextWithShadow(screen, infoString, ui.fontFace, infoX, infoY)
	}
}

func drawTextUpperLeft(dst *ebiten.Image, str string, face font.Face) {
	bounds := text.BoundString(face, str)

	textX := MARGIN
	textY := bounds.Dy() + MARGIN

	drawTextWithShadow(dst, str, face, textX, textY)
}

func drawTextUpperRight(dst *ebiten.Image, str string, face font.Face) {
	bounds := text.BoundString(face, str)

	screenX, _ := dst.Size()
	textX := screenX - bounds.Dx() - MARGIN
	textY := bounds.Dy() + MARGIN

	drawTextWithShadow(dst, str, face, textX, textY)

}

func (ui *UI) getScaleFactor() int {
	return ui.possibleScaleFactors[ui.scaleFactorIndex]
}

// Draw offset text in black and then white to get a slight "shadow" which helps with readability.
func drawTextWithShadow(dst *ebiten.Image, str string, face font.Face, x, y int) {
	text.Draw(dst, str, face, x+SHADOW_OFFSET, y+SHADOW_OFFSET, color.Black)
	text.Draw(dst, str, face, x, y, color.White)
}

func (ui *UI) getSpeedup() float64 {
	return math.Pow(2, float64(ui.speed))
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
