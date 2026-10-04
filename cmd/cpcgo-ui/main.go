//go:build liveui

package main

import (
	"flag"
	"fmt"
	"os"

	"cpcgo/internal/cpc"
	"cpcgo/internal/dsk"
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
		romPath    = flag.String("rom", "cpc6128.rom", "path to the 32K OS+BASIC ROM for the model")
		amsdosPath = flag.String("amsdos", "", "path to the 16K AMSDOS ROM (needed for disks)")
		diskPath   = flag.String("disk", "", "DSK image to insert in drive A")
		model      = flag.String("model", string(cpc.Model6128), "CPC model: 464, 664 or 6128")
		crtcType   = flag.Int("crtc", 1, "CRTC type (0-4)")
		scale      = flag.Int("scale", 1, "display scale factor")
		mute       = flag.Bool("mute", false, "disable sound")
		screenshot = flag.String("screenshot", "", "optional PNG path written when F12 is pressed")
	)
	flag.Parse()

	if *scale < 1 {
		return fmt.Errorf("scale must be at least 1")
	}
	image, err := rom.LoadOSBasic(*romPath)
	if err != nil {
		return err
	}
	if *amsdosPath != "" {
		if image, err = image.LoadAMSDOS(*amsdosPath); err != nil {
			return err
		}
	}

	machine, err := cpc.New(cpc.Config{
		Model:         cpc.Model(*model),
		ROMs:          image,
		CRTCType:      *crtcType,
		DiskInterface: *diskPath != "",
	})
	if err != nil {
		return err
	}
	if *diskPath != "" {
		if machine.FDC() == nil {
			return fmt.Errorf("--disk needs --amsdos")
		}
		disk, err := dsk.Load(*diskPath)
		if err != nil {
			return err
		}
		machine.FDC().Insert(0, disk)
	}

	return ebitenui.Run(machine, ebitenui.Config{Scale: *scale, ScreenshotPath: *screenshot, Mute: *mute})
}
