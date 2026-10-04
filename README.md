# cpcgo

`cpcgo` is an Amstrad CPC emulator written in Go. It emulates the CPC 464,
664 and 6128 at the hardware level: a Z80 with CPC bus timing, the 6845 CRTC,
the Gate Array, the 8255 PPI, the AY-3-8912 PSG and the uPD765 floppy disk
controller.

## Current Status

- Z80 core written for cpcgo: all documented and undocumented instructions,
  undocumented flags, MEMPTR, interrupt modes 0/1/2, and machine-cycle timing
  with the Gate Array's WAIT stretching (instructions take whole
  microseconds, I/O happens at the right point inside the instruction).
  Passes ZEXDOC and ZEXALL.
- CRTC types 0, 1, 2 and 4 (character-clocked counters, HSYNC/VSYNC, type
  specific register reads and sync widths).
- Gate Array: palette, modes 0-3 (mode changes take effect at HSYNC), ROM and
  RAM mapping, raster interrupt counter with the VSYNC resynchronisation.
- Monitor model producing 768x536 frames (48 µs x 268 scanlines, doubled).
- PPI and PSG including keyboard scanning and sound generation.
- uPD765 FDC with standard and extended DSK images.
- Headless CLI, Ebiten live UI with sound, and a
  [cpc-validation](https://github.com/darious/cpc-validation) runner.

## ROMs

ROM files are local runtime inputs and are not committed. Place them in the
repository root (or pass paths):

- `cpc464.rom`, `cpc664.rom`, `cpc6128.rom`: 32K OS+BASIC images.
- `amsdos.rom`: 16K AMSDOS image (664/6128 disk support).

## Run

Headless, saving the final frame:

```sh
go run ./cmd/cpcgo --rom cpc6128.rom --amsdos amsdos.rom --frames 200 --dump-frame /tmp/cpcgo.png
```

Other models: `--model 464 --rom cpc464.rom`, `--model 664 --rom cpc664.rom`.
Insert a disk with `--disk game.dsk`.

Live Ebiten UI:

```sh
go run -tags liveui ./cmd/cpcgo-ui --rom cpc6128.rom --amsdos amsdos.rom [--disk game.dsk]
```

Add `--screenshot /tmp/cpcgo-live.png` and press `F12` to capture the screen.

## Validation

`cmd/cpc-runner-cpcgo` implements the cpc-validation runner protocol. With
cpc-validation checked out next to cpcgo:

```sh
go build -o cpc-runner-cpcgo ./cmd/cpc-runner-cpcgo   # finds ROMs next to itself or in $CPCGO_ROM_DIR
cd ../cpc-validation
uv run cpc-validation run --runner ../cpcgo/cpc-runner-cpcgo --catalog catalog/
```

## Test

```sh
./test.sh                          # format, vet, unit tests, UI compile, smoke test
CPCGO_ZEX=1 go test ./internal/z80 # ZEXDOC/ZEXALL instruction exercisers (several minutes)
```

## License

AGPL-3.0-or-later. See `LICENSE`.
