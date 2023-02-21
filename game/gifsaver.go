package game

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"log"
	"os"
	"strconv"
	"time"
)

const (
	// The folder to which GIFs will be saved.
	IMAGE_FOLDER = "output"

	// Delay between frames in hundredths of seconds, approximating the 1/60 * 100 ≈ 1.667 required for 60 FPS.
	FRAME_DELAY = 2
)

type GifSaver struct {
	// The filename to which the GifSaver will save the GIF file.
	fileName string

	palette color.Palette

	// Paletted images corresponding to each GIF frame.
	frames []*image.Paletted

	// The successive delay times, one per frame. In practice this is always FRAME_DELAY.
	delays []int
}

func newGifSaver(bRules, sRules Ruleset) GifSaver {
	res := GifSaver{}

	// Give the run a filename which combines a timestamp and a simulation ruleset string.
	bNums, sNums := "", ""
	for i := 0; i <= 8; i++ {
		numStr := strconv.Itoa(i)
		if bRules[i] {
			bNums += numStr
		}
		if sRules[i] {
			sNums += numStr
		}
	}
	// Example filename: 20230221_202457_B3S23.gif (where B3S23 represents the ruleset)
	res.fileName = fmt.Sprintf("%v_B%vS%v.gif", time.Now().Format("20060102_150405"), bNums, sNums)

	// The pallette for our GIFs is always black and white.
	res.palette = color.Palette{color.Black, color.White}

	res.frames = []*image.Paletted{}
	res.delays = []int{}

	return res
}

func (gs *GifSaver) saveFrame(img image.Image) {

	// Created a paletted image from the simulation board image.
	bounds := img.Bounds()
	dst := image.NewPaletted(bounds, gs.palette)
	draw.Draw(dst, bounds, img, bounds.Min, draw.Src)

	// Add the image to our frames.
	gs.frames = append(gs.frames, dst)
	gs.delays = append(gs.delays, FRAME_DELAY)
}

func (gs *GifSaver) writeToFile() {
	// Create the image directory if it doesn't exist.
	if _, err := os.Stat(IMAGE_FOLDER); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(IMAGE_FOLDER, os.ModePerm)
		if err != nil {
			log.Fatal(fmt.Errorf("could not create image directory: %v", err))
		}
	}

	// Open the file which we'll be writing to.
	path := fmt.Sprintf("%v/%v", IMAGE_FOLDER, gs.fileName)
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Write the GIF to the opened file.
	err = gif.EncodeAll(f, &gif.GIF{
		Image:     gs.frames,
		Delay:     gs.delays,
		LoopCount: 0,
	})
	if err != nil {
		log.Fatal(err)
	}
}
