// Command cpcgo runs the emulator headlessly: it boots the selected model,
// optionally inserts a disk, runs for a number of frames and can save the
// final frame as a PNG.
package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"

	"cpcgo/internal/cpc"
	"cpcgo/internal/dsk"
	"cpcgo/internal/rom"
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
		frames     = flag.Int("frames", 200, "frames to run")
		dumpFrame  = flag.String("dump-frame", "", "write the final frame to this PNG")
	)
	flag.Parse()

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

	fmt.Printf("cpcgo model=%s crtc=%d os=%d basic=%d amsdos=%d\n",
		*model, *crtcType, len(image.LowerOS), len(image.Basic), len(image.AMSDOS))

	for i := 0; i < *frames; i++ {
		machine.RunFrame()
	}
	regs := machine.CPU().Registers()
	fmt.Printf("frames=%d micros=%d pc=%04x sp=%04x mode=%d\n",
		*frames, machine.Micros(), regs.PC, regs.SP, machine.GateArray().Mode())

	if *dumpFrame != "" {
		file, err := os.Create(*dumpFrame)
		if err != nil {
			return err
		}
		defer file.Close()
		if err := png.Encode(file, machine.Framebuffer()); err != nil {
			return err
		}
		fmt.Printf("frame=%s\n", *dumpFrame)
	}
	return nil
}
