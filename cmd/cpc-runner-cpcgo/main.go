// Command cpc-runner-cpcgo drives cpcgo for the cpc-validation harness.
//
// It implements the runner protocol from cpc-validation's
// schema/runner-protocol.md:
//
//	cpc-runner-cpcgo --model {Cpc464|Cpc664|Cpc6128} --crtc {Type0|Type1|Type2|Type4}
//	                 --frames N --output-dir DIR [--disk-a P] [--disk-b P]
//	                 [--rom P] [--input SCRIPT]
//
// Firmware ROMs are read from --rom-dir, $CPCGO_ROM_DIR, the executable's
// directory or the current directory: cpc464.rom, cpc664.rom, cpc6128.rom
// (32K OS+BASIC each) and amsdos.rom (16K).
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cpcgo/internal/cpc"
	"cpcgo/internal/dsk"
	"cpcgo/internal/keyboard"
	"cpcgo/internal/rom"
)

type options struct {
	model, crtc     string
	frames          int
	outputDir       string
	diskA, diskB    string
	romOverride     string
	input           string
	romDir          string
	framesSpecified bool
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "cpc-runner-cpcgo:", err)
		os.Exit(2)
	}
}

func parseArgs(args []string) (options, error) {
	var o options
	for i := 0; i < len(args); i++ {
		name := args[i]
		value := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("missing value for %s", name)
			}
			i++
			return args[i], nil
		}
		var err error
		switch name {
		case "--model":
			o.model, err = value()
		case "--crtc":
			o.crtc, err = value()
		case "--frames":
			var v string
			if v, err = value(); err == nil {
				o.frames, err = strconv.Atoi(v)
				o.framesSpecified = true
			}
		case "--output-dir":
			o.outputDir, err = value()
		case "--disk-a":
			o.diskA, err = value()
		case "--disk-b":
			o.diskB, err = value()
		case "--rom":
			o.romOverride, err = value()
		case "--input":
			o.input, err = value()
		case "--rom-dir":
			o.romDir, err = value()
		default:
			fmt.Fprintf(os.Stderr, "ignoring unknown flag %s\n", name)
		}
		if err != nil {
			return o, err
		}
	}
	if o.model == "" || o.crtc == "" || !o.framesSpecified || o.frames < 0 || o.outputDir == "" {
		return o, errors.New("usage: --model M --crtc C --frames N --output-dir DIR [--disk-a P] [--disk-b P] [--rom P] [--input P]")
	}
	return o, nil
}

func run(args []string) error {
	o, err := parseArgs(args)
	if err != nil {
		return err
	}

	model, romFile, err := modelFor(o.model)
	if err != nil {
		return err
	}
	crtcType, err := crtcFor(o.crtc)
	if err != nil {
		return err
	}

	romDir := findROMDir(o.romDir, romFile)
	image, err := rom.LoadOSBasic(filepath.Join(romDir, romFile))
	if err != nil {
		return err
	}
	if model != cpc.Model464 || o.diskA != "" || o.diskB != "" {
		image, err = image.LoadAMSDOS(filepath.Join(romDir, "amsdos.rom"))
		if err != nil {
			return err
		}
	}
	if o.romOverride != "" {
		data, err := os.ReadFile(o.romOverride)
		if err != nil {
			return err
		}
		switch len(data) {
		case rom.BankSize:
			image.LowerOS = data
		case rom.OSBasicSize:
			image.LowerOS, image.Basic = data[:rom.BankSize], data[rom.BankSize:]
		default:
			return fmt.Errorf("--rom %s must be 16K or 32K", o.romOverride)
		}
	}

	machine, err := cpc.New(cpc.Config{
		Model:         model,
		ROMs:          image,
		CRTCType:      crtcType,
		DiskInterface: o.diskA != "" || o.diskB != "",
	})
	if err != nil {
		return err
	}
	for drive, path := range []string{o.diskA, o.diskB} {
		if path == "" {
			continue
		}
		disk, err := dsk.Load(path)
		if err != nil {
			return err
		}
		machine.FDC().Insert(drive, disk)
	}

	var script []op
	if o.input != "" {
		if script, err = loadScript(o.input); err != nil {
			return err
		}
	}

	inputFrames := runScript(machine, script)
	for i := 0; i < o.frames; i++ {
		machine.RunFrame()
	}
	return writeArtefacts(o, machine, inputFrames+o.frames)
}

func modelFor(name string) (cpc.Model, string, error) {
	switch name {
	case "Cpc464":
		return cpc.Model464, "cpc464.rom", nil
	case "Cpc664":
		return cpc.Model664, "cpc664.rom", nil
	case "Cpc6128":
		return cpc.Model6128, "cpc6128.rom", nil
	}
	return "", "", fmt.Errorf("unknown model %s", name)
}

func crtcFor(name string) (int, error) {
	switch name {
	case "Type0":
		return 0, nil
	case "Type1":
		return 1, nil
	case "Type2":
		return 2, nil
	case "Type4":
		return 4, nil
	}
	return 0, fmt.Errorf("unknown crtc %s", name)
}

func findROMDir(flagDir, probe string) string {
	candidates := []string{flagDir, os.Getenv("CPCGO_ROM_DIR")}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(exe))
	}
	candidates = append(candidates, ".")
	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, probe)); err == nil {
			return dir
		}
	}
	return "."
}

// ---------------------------------------------------------------------------
// Input scripts

type opKind int

const (
	opSleep opKind = iota
	opPress
	opRelease
)

type op struct {
	kind   opKind
	frames int
	key    keyboard.Key
}

func loadScript(path string) ([]op, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var ops []op
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimLeft(strings.TrimRight(scanner.Text(), "\r"), " \t")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "sleep "):
			n, err := strconv.Atoi(strings.TrimSpace(line[6:]))
			if err != nil {
				return nil, fmt.Errorf("bad sleep: %s", line)
			}
			ops = append(ops, op{kind: opSleep, frames: n})
		case strings.HasPrefix(line, "type_text "):
			ops = append(ops, typeText(line[10:])...)
		case strings.HasPrefix(line, "key_press "), strings.HasPrefix(line, "key_release "):
			press := strings.HasPrefix(line, "key_press ")
			name := strings.TrimSpace(line[strings.IndexByte(line, ' ')+1:])
			key, ok := keyboard.Names[name]
			if !ok {
				fmt.Fprintf(os.Stderr, "input script: unknown key %s\n", name)
				continue
			}
			kind := opRelease
			if press {
				kind = opPress
			}
			ops = append(ops, op{kind: kind, key: key})
		default:
			fmt.Fprintf(os.Stderr, "input script: unknown directive %s\n", line)
		}
	}
	return ops, scanner.Err()
}

// typeText holds each key for 2 frames and releases it for 2 frames. The
// literal sequence \n also means Enter.
func typeText(text string) []op {
	text = strings.ReplaceAll(text, `\n`, "\n")
	var ops []op
	for _, r := range text {
		chord, ok := keyboard.ChordForRune(r)
		if !ok {
			fmt.Fprintf(os.Stderr, "type_text: skipping unsupported char %q\n", r)
			continue
		}
		for _, k := range chord {
			ops = append(ops, op{kind: opPress, key: k})
		}
		ops = append(ops, op{kind: opSleep, frames: 2})
		for i := len(chord) - 1; i >= 0; i-- {
			ops = append(ops, op{kind: opRelease, key: chord[i]})
		}
		ops = append(ops, op{kind: opSleep, frames: 2})
	}
	return ops
}

// runScript executes the input script and returns the frames it took.
func runScript(machine *cpc.Machine, ops []op) int {
	frames := 0
	for _, o := range ops {
		switch o.kind {
		case opPress:
			machine.Keyboard().Press(o.key)
		case opRelease:
			machine.Keyboard().Release(o.key)
		case opSleep:
			for i := 0; i < o.frames; i++ {
				machine.RunFrame()
			}
			frames += o.frames
		}
	}
	return frames
}

// ---------------------------------------------------------------------------
// Artefacts

func writeArtefacts(o options, machine *cpc.Machine, framesRun int) error {
	if err := os.MkdirAll(o.outputDir, 0o755); err != nil {
		return err
	}

	img := machine.Monitor().Image()
	file, err := os.Create(filepath.Join(o.outputDir, "screen.png"))
	if err != nil {
		return err
	}
	if err := png.Encode(file, img); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	ram := machine.Memory().PhysicalRAM()
	if err := os.WriteFile(filepath.Join(o.outputDir, "ram.bin"), ram, 0o644); err != nil {
		return err
	}

	meta := map[string]any{
		"model":       o.model,
		"crtc":        o.crtc,
		"frames_run":  framesRun,
		"exit":        "frames_complete",
		"ram_size":    len(ram),
		"screen_mode": machine.GateArray().Mode(),
		"screen_ma":   int(machine.CRTC().Register(12))<<8 | int(machine.CRTC().Register(13)),
		"screen":      map[string]int{"width": img.Bounds().Dx(), "height": img.Bounds().Dy()},
		"emulator":    "cpcgo",
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(o.outputDir, "meta.json"), append(data, '\n'), 0o644)
}
