package main

import (
	"flag"
	"fmt"
	"os"

	"cpcgo/internal/cpc"
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
		romPath    = flag.String("rom", "cpc6128.rom", "path to 32K CPC6128 OS+BASIC ROM")
		amsdosPath = flag.String("amsdos", "", "path to optional 16K AMSDOS ROM")
		diskPath   = flag.String("disk", "", "path to optional DSK image")
		model      = flag.String("model", string(cpc.Model6128), "CPC model to emulate")
		scale      = flag.Int("scale", 2, "display scale factor")
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
		Disk:  *diskPath,
		Scale: *scale,
	})
	if err != nil {
		return err
	}

	config := machine.Config()
	fmt.Printf("cpcgo model=%s scale=%d os=%d basic=%d amsdos=%d\n",
		config.Model,
		config.Scale,
		len(config.ROMs.LowerOS),
		len(config.ROMs.Basic),
		len(config.ROMs.AMSDOS),
	)

	if config.Disk != "" {
		fmt.Printf("disk=%s\n", config.Disk)
	}

	return nil
}
