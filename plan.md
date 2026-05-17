# CPCGo Project Plan

## Confirmed Direction

- First target: boot a CPC6128 to the Locomotive BASIC prompt.
- UI: Ebiten.
- Language: Go.
- Intended license: AGPL-3.0.
- ROMs available:
  - `cpc6128.rom`, 32K OS+BASIC, SHA-256 `31c3668c67bea027dab698ece233c9434d9324f9ba7dac84db58f400b6689562`
  - `amsdos.rom`, 16K AMSDOS, SHA-256 `ea65e0fb44ee93ede4b6c507509b7e5ddf497fb7155023bea91ef229469fa04d`

## License Notes

AGPL-3.0 is workable for this project, but it affects dependency choices and prior-art reuse:

- Prefer permissive dependencies: MIT, BSD, Apache-2.0, ISC, Zlib.
- GPLv3-compatible code can usually be combined with AGPL-3.0, subject to the exact license text.
- GPLv2-only code is a problem for AGPL-3.0 compatibility. Treat Caprice32 as a behavioral reference unless its relevant code is clearly GPLv2-or-later or we choose a compatible licensing route.
- ROM files should not be committed unless their redistribution rights are clear. Keep them local runtime inputs.

## Stage 0: Repository Foundation

Goal: create a clean Go project structure that supports emulator-core development, tests, and an Ebiten frontend without tying hardware logic to UI code.

Tasks:

- Create `go.mod`.
- Add `LICENSE` with AGPL-3.0 text or a clear AGPL-3.0 notice.
- Add `README.md` with current scope, ROM expectations, and basic run command.
- Add package skeleton:
  - `cmd/cpcgo`
  - `internal/cpc`
  - `internal/bus`
  - `internal/rom`
  - `internal/gatearray`
  - `internal/crtc`
  - `internal/ppi`
  - `internal/keyboard`
  - `internal/psg`
  - `internal/fdc`
  - `internal/dsk`
  - `internal/debugger`
- Add CI-style local commands:
  - `go test ./...`
  - `go vet ./...`
  - optional formatter check.

Exit criteria:

- `go test ./...` passes.
- CLI can parse `--rom`, `--amsdos`, and `--disk` flags even if most are not used yet.
- ROM files are loaded from paths, validated by size, and not embedded.

## Stage 1: CPU Core Evaluation

Goal: select a Z80 core that can boot firmware soon while leaving room for later timing accuracy.

Status: complete. The selected core is `github.com/user-none/go-chip-z80` version `v0.0.0-20260315161243-6c949bf925bb`. See `docs/stage1-z80.md`.

Candidates:

- `github.com/user-none/go-chip-z80`
- `github.com/romychs/z80go`

Tasks:

- Check license compatibility.
- Build a thin `internal/z80` adapter around the chosen core.
- Confirm the core supports:
  - memory read/write callbacks,
  - I/O read/write callbacks,
  - maskable interrupt handling,
  - reset,
  - cycle or T-state accounting.
- Run any upstream CPU tests if practical.
- Add a minimal bus test program to verify reads, writes, and I/O callbacks.

Exit criteria:

- One Z80 core is selected and wrapped.
- The rest of the emulator does not import the third-party CPU package directly.
- A small synthetic program can execute through the bus adapter.

## Stage 2: Memory, ROMs, And I/O Bus

Goal: implement enough CPC6128 memory and port behavior for firmware reset code to run against believable hardware.

Status: bus foundation complete. Memory banking, ROM overlays, upper ROM selection, I/O dispatch, and machine CPU wiring are implemented. Device-specific I/O decoding remains for later hardware stages. See `docs/stage2-bus.md`.

Tasks:

- Implement 128K RAM as eight 16K banks.
- Split `cpc6128.rom` into:
  - lower OS ROM,
  - upper BASIC ROM.
- Load `amsdos.rom` as upper ROM bank 7, unless later verification shows a different convention is needed.
- Implement lower ROM enable/disable.
- Implement upper ROM enable/disable and bank selection.
- Implement CPC6128 RAM banking configurations 0-7.
- Implement I/O device dispatch with partial address decoding.
- Add tests for:
  - RAM banking table,
  - ROM overlays,
  - writes passing through ROM overlays to underlying RAM,
  - upper ROM selection,
  - unknown I/O behavior.

Exit criteria:

- Memory and I/O tests pass.
- The CPU can reset and fetch instructions from lower ROM at `0x0000`.
- Debug tracing can show PC, opcode bytes, and selected device I/O.

## Stage 3: Firmware Boot Skeleton

Goal: run the CPC firmware far enough to exercise core hardware initialization without rendering a correct screen yet.

Status: initial skeleton complete and boot probing started. Gate Array memory-control commands, upper ROM selection, CRTC register writes, PPI register stubs, PSG latch path, keyboard matrix, approximate timing/interrupts, VSync bit, headless probe, and crude framebuffer PNG dump are wired into the machine. A real ROM PNG dump reaches the BASIC banner and `Ready` prompt. See `docs/stage3-firmware-skeleton.md`.

Tasks:

- Implement basic Gate Array command decoding:
  - pen selection,
  - ink values,
  - border color,
  - screen mode,
  - ROM control,
  - RAM banking command.
- Implement basic CRTC register index/data writes.
- Implement PPI port/control register behavior required by firmware.
- Stub PSG access path well enough that PPI/PSG reads and writes do not break firmware.
- Implement interrupt line plumbing from Gate Array to Z80, even if timing is approximate.
- Add an instruction-count or cycle-count run mode for headless testing.

Exit criteria:

- Firmware runs beyond early hardware setup without crashing into unmapped behavior.
- Trace logs identify the next missing hardware feature instead of opaque CPU failure.
- Headless run has a deterministic stop condition.

## Stage 4: First Video And BASIC Prompt

Goal: display the CPC boot screen and BASIC prompt in an Ebiten window.

Status: first live display complete. The Ebiten UI is build-tagged behind
`liveui`, boots to the BASIC prompt, uses the CPC hardware colour table, fills
the border from the Gate Array border pen, and presents a line-doubled display
instead of raw 640x200 square pixels.

Tasks:

- Add Ebiten frontend in `cmd/cpcgo-ui`.
- Keep emulator core independent from Ebiten imports.
- Implement framebuffer generation for modes 0, 1, and 2.
- Implement CPC palette mapping.
- Implement screen base/address decoding sufficiently for firmware text display.
- Render border and visible display region.
- Add frame pacing around 50 Hz PAL output.
- Add a screenshot/debug frame dump command for testing.

Exit criteria:

- Running `go run -tags liveui ./cmd/cpcgo-ui --rom cpc6128.rom --amsdos amsdos.rom` opens a window.
- The emulator reaches a recognizable BASIC prompt.
- Screen output is stable enough to read firmware text.
- `./test.sh` passes.

## Stage 5: Keyboard Input

Goal: type into BASIC through the real CPC keyboard matrix path.

Status: in progress. The CPC matrix coordinates now use the documented
10-by-8 row/bit layout and the Ebiten frontend maps letters, digits, core
punctuation, arrows, modifiers, return, delete, and function keys through that
matrix.

Tasks:

- Implement CPC keyboard matrix.
- Map Ebiten key events to CPC rows and columns.
- Implement PSG/PPI keyboard row selection path.
- Support Shift, Ctrl, Enter, Backspace/Delete behavior as CPC firmware expects.
- Add a small input script mode for automated smoke tests.

Exit criteria:

- User can type `PRINT 1+1` and press Enter.
- BASIC prints `2`.
- Automated smoke test can inject simple key sequences.

## Stage 6: Timing And Interrupt Tightening

Goal: move from "boots by luck" timing to a stable CPC-like execution model.

Tasks:

- Add a deterministic machine scheduler.
- Track CPU T-states or equivalent CPC timing units.
- Model Gate Array interrupt counter behavior.
- Connect CRTC HSync/VSync events to Gate Array interrupts.
- Ensure frame timing and CPU execution stay deterministic across runs.
- Add debugger counters for frames, scanlines, interrupts, and CPU cycles.

Exit criteria:

- BASIC prompt still boots.
- Keyboard remains reliable.
- Interrupt counts and frame cadence are plausible for PAL CPC behavior.
- Timing-sensitive firmware behavior is not dependent on host frame rate.

## Stage 7: Audio

Goal: make BASIC `SOUND` and software PSG output audible.

Tasks:

- Implement AY-3-8912 registers.
- Implement tone, noise, mixer, envelope, and amplitude table behavior.
- Generate audio samples from emulator time.
- Feed samples into Ebiten audio.
- Add basic buffering and underrun diagnostics.

Exit criteria:

- BASIC `SOUND` commands produce audible output.
- Audio does not materially destabilize video or emulation speed.
- Muting and volume controls exist.

## Stage 8: Disk And AMSDOS

Goal: load `.dsk` images and run simple disk software.

Tasks:

- Implement standard DSK parser.
- Add extended DSK parser after standard DSK works.
- Implement uPD765 command state machine for common AMSDOS operations:
  - `SPECIFY`,
  - `SENSE INTERRUPT STATUS`,
  - `RECALIBRATE`,
  - `SEEK`,
  - `READ ID`,
  - `READ DATA`,
  - `WRITE DATA` later.
- Implement floppy motor control and drive selection.
- Wire AMSDOS upper ROM into ROM selection.
- Add CLI flag `--disk path/to/file.dsk`.

Exit criteria:

- BASIC command `CAT` reads a standard disk directory.
- A simple BASIC program can load from disk.
- At least one known simple game or demo starts from disk.

## Stage 9: Compatibility And Debugging Tools

Goal: make the emulator practical to develop and improve.

Tasks:

- Add debugger console or overlay:
  - pause/resume,
  - step instruction,
  - breakpoints,
  - register view,
  - memory view,
  - I/O trace filters.
- Add save/load state.
- Add screenshot capture.
- Add machine config file.
- Add compatibility notes for tested programs.

Exit criteria:

- Bugs can be investigated without adding ad hoc print statements each time.
- Save states are deterministic for CPU, memory, Gate Array, CRTC, PPI, PSG, keyboard, and FDC state.
- Compatibility list records pass/fail details and emulator version.

## Stage 10: Accuracy Pass

Goal: improve compatibility with games, demos, raster effects, and edge-case disk behavior.

Tasks:

- Move video toward scanline-aware rendering.
- Support CRTC type profiles.
- Tighten Gate Array wait-state and interrupt timing.
- Improve FDC status/result edge cases.
- Add overscan and split-raster behavior.
- Add joystick support.
- Improve keyboard layout configurability.
- Add optional CRT-style display filters without making them required.

Exit criteria:

- A representative game corpus works.
- Raster effects and overscan cases improve measurably.
- Known failing cases have tracked issues or compatibility notes.

## Default Development Loop

For each stage:

1. Add or update tests for the behavior being implemented.
2. Implement the smallest hardware behavior needed for the stage.
3. Run `go test ./...`.
4. Run a headless boot or smoke test when available.
5. Update `spec.md`, `plan.md`, or compatibility notes if facts change.

## Immediate Next Actions

1. Create the Go module and package skeleton.
2. Add AGPL-3.0 license notice.
3. Evaluate the two Go Z80 cores for license/API fit.
4. Implement ROM loading and the memory banking tests.
5. Build the first headless CPU reset run.
