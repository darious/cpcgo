package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

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
		probe      = flag.Int("probe-instructions", 0, "run a bounded headless boot probe for N instructions")
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
	if *probe > 0 {
		printProbe(machine.Probe(cpc.ProbeOptions{Instructions: *probe}))
	}

	return nil
}

func printProbe(result cpc.ProbeResult) {
	regs := result.Registers
	fmt.Printf("probe instructions=%d cycles=%d halted=%v\n", result.Instructions, result.Cycles, result.Halted)
	fmt.Printf("pc=%04x sp=%04x af=%04x bc=%04x de=%04x hl=%04x ix=%04x iy=%04x i=%02x r=%02x im=%d\n",
		regs.PC,
		regs.SP,
		regs.AF,
		regs.BC,
		regs.DE,
		regs.HL,
		regs.IX,
		regs.IY,
		regs.I,
		regs.R,
		regs.IM,
	)
	fmt.Printf("io reads=%d writes=%d unhandled_reads=%d unhandled_writes=%d\n",
		result.IO.Reads,
		result.IO.Writes,
		result.IO.UnhandledReads,
		result.IO.UnhandledWrites,
	)
	fmt.Printf("timing cycles=%d frames=%d interrupts=%d frame_cycle=%d vsync=%v\n",
		result.Timing.Cycles,
		result.Timing.Frames,
		result.Timing.Interrupts,
		result.Timing.FrameCycle,
		result.Timing.VSync,
	)
	printTopPorts("top_read_ports", result.IO.ReadPorts)
	printTopPorts("top_write_ports", result.IO.WritePorts)
	printTopPorts("top_unhandled_ports", result.IO.UnhandledPorts)
}

func printTopPorts(label string, ports map[uint16]int) {
	const limit = 10
	type count struct {
		port uint16
		n    int
	}
	counts := make([]count, 0, len(ports))
	for port, n := range ports {
		counts = append(counts, count{port: port, n: n})
	}
	sort.Slice(counts, func(i, j int) bool {
		if counts[i].n == counts[j].n {
			return counts[i].port < counts[j].port
		}
		return counts[i].n > counts[j].n
	})

	fmt.Printf("%s", label)
	if len(counts) == 0 {
		fmt.Println(" none")
		return
	}
	for i, count := range counts {
		if i >= limit {
			break
		}
		fmt.Printf(" %04x:%d", count.port, count.n)
	}
	fmt.Println()
}
