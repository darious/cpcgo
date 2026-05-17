//go:build liveui

package main

import (
	"flag"
	"fmt"
	"os"

	"cpcgo/internal/cpc"
	"cpcgo/internal/rom"
	"cpcgo/internal/ui/ebitenui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var (
		romPath    = flag.String("rom", "cpc6128.rom", "path to 32K CPC6128 OS+BASIC ROM")
		amsdosPath = flag.String("amsdos", "", "path to optional 16K AMSDOS ROM")
		model      = flag.String("model", string(cpc.Model6128), "CPC model to emulate")
		scale      = flag.Int("scale", 1, "display scale factor")
		screenshot = flag.String("screenshot", "", "optional PNG path written when F12 is pressed")
	)
	flag.Parse()

	if *model != string(cpc.Model6128) {
		return fmt.Errorf("unsupported model %q", *model)
	}
	if *scale < 1 {
		return fmt.Errorf("scale must be at least 1")
	}

	image, err := rom.LoadOSBasic(*romPath)
	if err != nil {
		return err
	}
	if *amsdosPath != "" {
		image, err = image.LoadAMSDOS(*amsdosPath)
		if err != nil {
			return err
		}
	}

	machine, err := cpc.New(cpc.Config{
		Model: cpc.Model(*model),
		ROMs:  image,
		Scale: *scale,
	})
	if err != nil {
		return err
	}

	return ebitenui.Run(machine, ebitenui.Config{Scale: *scale, ScreenshotPath: *screenshot})
}
