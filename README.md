# cpcgo

`cpcgo` is a Go Amstrad CPC6128 emulator project. The first target is a faithful boot to the Locomotive BASIC prompt, followed by keyboard input, timing accuracy, audio, and disk support.

## Current Status

Stage 0 scaffold is in progress:

- Go module and package layout.
- ROM loading and size validation.
- CLI flags for ROM, AMSDOS, disk image, model, and scale.
- Z80 adapter using `github.com/user-none/go-chip-z80`.
- CPC6128 memory/ROM bus foundation.
- Initial Gate Array, CRTC, and PPI I/O skeleton.
- CPC keyboard matrix path through PSG/PPI.
- Project specs in `spec.md` and staged plan in `plan.md`.

## ROMs

ROM files are local runtime inputs and should not be committed.

Expected files while developing:

- `cpc6128.rom`: 32K OS+BASIC image.
- `amsdos.rom`: optional 16K AMSDOS image for later disk support.

## Run

```sh
go run ./cmd/cpcgo --rom cpc6128.rom --amsdos amsdos.rom
```

At this stage the command validates inputs, runs the emulated machine headlessly, and can dump a crude framebuffer that reaches the BASIC prompt.

Headless boot probe:

```sh
go run ./cmd/cpcgo --rom cpc6128.rom --amsdos amsdos.rom --probe-instructions 1000000
```

Crude framebuffer dump:

```sh
go run ./cmd/cpcgo --rom cpc6128.rom --amsdos amsdos.rom --frame-instructions 1000000 --dump-frame /tmp/cpcgo-frame.png
```

## Test

```sh
go test ./...
```

## License

AGPL-3.0-or-later. See `LICENSE`.
