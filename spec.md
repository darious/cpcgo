# CPCGo Amstrad CPC6128 Emulator Spec

## Goal

Build a Go emulator for the Amstrad CPC6128 that behaves like real hardware for normal BASIC use, commercial games, demos, and disk software. The design should aim for low-level hardware emulation rather than a high-level BASIC or firmware interpreter.

Initial workspace ROM:

- `cpc6128.rom`
- Size: 32768 bytes
- SHA-256: `31c3668c67bea027dab698ece233c9434d9324f9ba7dac84db58f400b6689562`
- Observed contents: CPC6128 firmware and Locomotive BASIC 1.1 strings. This looks like a 32K OS+BASIC image, not a full 48K OS+BASIC+AMSDOS set. Disk support will likely need an additional 16K AMSDOS ROM.

## Non-Goals For The First Milestone

- CPC Plus / GX4000 support.
- Exact analogue CRT simulation.
- Printer, serial, expansion bus peripherals, Multiface, and memory expansions beyond stock 128K.
- Tape support before disk and keyboard/video/audio basics are stable.

## Prior Art And References

- Caprice32: mature C/C++ CPC emulator with CPC464, CPC664, CPC6128, disk/tape/snapshot support, and developer tools. Useful as a behavioral reference, but GPLv2 licensing means we should not copy code unless this project deliberately becomes GPL-compatible. Reference: https://github.com/ColinPitrat/caprice32
- CPCemu: mature emulator whose feature list highlights the hard parts: realistic CPU/interrupt timing, CRTC variants, overscan, scan doubling, sound, and FDC edge cases. Reference: https://www.cpcwiki.eu/index.php/CPCemu
- Ronald: Rust CPC emulator, MIT licensed, useful for a smaller and more readable architecture reference. Its README says CPC6128 is work in progress, so do not treat it as authoritative for 6128 banking or disk behavior. Reference: https://github.com/mdm/ronald
- CLK: large, accurate, MIT licensed multi-machine emulator with Amstrad CPC support. Useful as a timing-oriented reference and test comparison target. Reference: https://github.com/TomHarte/CLK
- CPCWiki technical docs: primary reference for Gate Array, CRTC, PPI, PSG, I/O ports, memory banking, and disk controller behavior. Start with:
  - Technical documentation index: https://www.cpcwiki.eu/index.php/Technical_documentation
  - Default I/O ports: https://www.cpcwiki.eu/index.php/Default_I/O_Port_Summary
  - Gate Array: https://www.cpcwiki.eu/index.php/Gate_Array
  - CRTC: https://www.cpcwiki.eu/index.php/CRTC
  - PSG via PPI: https://www.cpcwiki.eu/index.php/How_to_access_the_PSG_via_PPI
- Z80 CPU options in Go:
  - `github.com/user-none/go-chip-z80`: cycle-counted Z80 package with bus callbacks, interrupts, save-state support, and documented limitations around WZ/MEMPTR and q flag behavior. Candidate for early boot and UI progress. Reference: https://pkg.go.dev/github.com/user-none/go-chip-z80
  - `github.com/romychs/z80go`: Go Z80 emulator/disassembler that advertises ZEXALL and Fuse Z80 test-suite success. Candidate if its API and license fit better. Reference: https://pkg.go.dev/github.com/romychs/z80go

Decision: start by evaluating an existing Go Z80 core instead of writing one from scratch. The CPC-specific work is already large; a tested CPU core shortens time to first boot. We can replace or fork the CPU later if timing or undocumented flag behavior blocks compatibility.

## Hardware Model

Target model: stock PAL Amstrad CPC6128.

Core components:

- Zilog Z80A-compatible CPU.
- 128K RAM organized as eight 16K banks.
- Lower ROM at `0x0000-0x3fff` and upper ROM at `0xc000-0xffff`.
- Gate Array / PAL RAM mapping, palette, graphics mode, interrupt timing, and CPU wait-state timing.
- Motorola 6845-compatible CRTC.
- Intel 8255 PPI.
- AY-3-8912 PSG accessed through the PPI.
- Keyboard matrix.
- NEC uPD765-compatible floppy disk controller with motor/drive state.
- Single built-in 3-inch drive A initially; optional drive B later.

## Timing Model

The emulator should use a deterministic master scheduler instead of running each subsystem independently by wall-clock time.

Recommended timebase:

- Use Z80 T-states as the CPU-facing unit.
- Convert CRTC/Gate Array work to CPC character and scanline timing.
- Run the machine in frame budgets for rendering/audio output, but keep internal execution cycle based.

Important constraints:

- The Gate Array arbitrates bus access for video and CPU. CPCWiki notes that CPU timings are effectively stretched so instruction timings land on microsecond boundaries and the effective CPU rate is about 3.3 MHz.
- Gate Array interrupts are tied to CRTC HSync/VSync. The interrupt counter increments on HSync falling edges and normally produces a 300 Hz interrupt cadence under 50 Hz PAL display timing.
- Accurate demos will eventually require per-scanline or finer rendering and interrupt positioning. The MVP may render per frame, but the architecture must not prevent scanline rendering later.

## Memory And ROM Banking

RAM:

- Store physical RAM as `[8][0x4000]byte`.
- Maintain four logical 16K read/write windows for CPU address ranges:
  - `0x0000-0x3fff`
  - `0x4000-0x7fff`
  - `0x8000-0xbfff`
  - `0xc000-0xffff`

CPC6128 RAM configurations from Gate Array/PAL MMR:

| Config | 0000-3FFF | 4000-7FFF | 8000-BFFF | C000-FFFF |
| --- | --- | --- | --- | --- |
| 0 | RAM_0 | RAM_1 | RAM_2 | RAM_3 |
| 1 | RAM_0 | RAM_1 | RAM_2 | RAM_7 |
| 2 | RAM_4 | RAM_5 | RAM_6 | RAM_7 |
| 3 | RAM_0 | RAM_3 | RAM_2 | RAM_7 |
| 4 | RAM_0 | RAM_4 | RAM_2 | RAM_3 |
| 5 | RAM_0 | RAM_5 | RAM_2 | RAM_3 |
| 6 | RAM_0 | RAM_6 | RAM_2 | RAM_3 |
| 7 | RAM_0 | RAM_7 | RAM_2 | RAM_3 |

ROM:

- Lower ROM maps over `0x0000-0x3fff` when enabled. Writes still go to underlying RAM.
- Upper ROM maps over `0xc000-0xffff` when enabled. Writes still go to underlying RAM.
- Upper ROM bank selection must support BASIC ROM initially and AMSDOS ROM once disk support is added.
- If only `cpc6128.rom` is supplied, split first 16K as lower OS ROM and second 16K as upper BASIC ROM.

## I/O Decoding

Implement CPC partial address decoding, not exact 8-bit port equality. Multiple devices may see a transaction if the port address selects them.

Initial required devices:

- Gate Array writes: selected when address bit 15 is `0` and bit 14 is `1`; recommended firmware port is `0x7fxx`.
- RAM MMR writes: same output address family, command bits identify MMR command.
- CRTC index/data ports: standard CPC CRTC I/O decoding.
- Upper ROM select: standard CPC ROM select port.
- PPI 8255 ports A/B/C/control: needed for PSG access, keyboard rows, cassette status, and VSync bit.
- FDC and floppy motor ports for CPC6128 disk support.

The I/O implementation should be table-driven or device-dispatch based so partial decoding rules stay explicit and testable.

## Video

Implement two layers:

1. CRTC model:
   - Registers R0-R17.
   - Horizontal/vertical counters.
   - MA/RA address generation.
   - HSync/VSync generation.
   - Type-specific behavior later. Start with a common behavior profile that boots firmware.

2. Gate Array renderer:
   - Palette and border color.
   - Mode 0: 160x200, 16 colors, 2 pixels per byte.
   - Mode 1: 320x200, 4 colors, 4 pixels per byte.
   - Mode 2: 640x200, 2 colors, 8 pixels per byte.
   - Overscan and split-raster support later.

MVP rendering:

- Produce a 50 Hz framebuffer from the visible area.
- Show border color.
- Correctly decode CPC planar pixel packing.

Accuracy target:

- Move to scanline rendering once boot, keyboard, and disk are working.
- Keep CRTC and Gate Array state serializable for save states and debugger views.

## Audio

Implement AY-3-8912 PSG:

- Three tone channels.
- Noise generator.
- Envelope generator.
- Mixer and amplitude tables.
- Register access via PPI port A and PPI port C control bits.

MVP can generate mono or stereo-mirrored output. Later add CPC-accurate speaker filtering and host audio latency tuning.

Potential approaches:

- Implement PSG directly in Go for license/control.
- Evaluate permissive PSG implementations for behavior comparison.

## Keyboard And Input

Implement CPC keyboard matrix, not direct ASCII input.

Host input requirements:

- Map modern keyboard keys to CPC rows/columns.
- Support key combinations with Shift/Ctrl.
- Provide configurable layout file later.
- Add joystick emulation through keyboard/gamepad mapping.

The PPI/PSG path must allow firmware keyboard scanning to work as on hardware.

## Disk Support

Required for a useful CPC6128 emulator:

- Load standard `.dsk` images first.
- Support extended `.dsk` after standard disks are stable.
- Emulate uPD765 commands used by AMSDOS and common games.
- Model drive motor, track, sector IDs, status registers, and result phases.
- Add AMSDOS ROM loading and upper ROM bank selection.

Later:

- Copy-protection and strange sector formats.
- IPF/CT-RAW only if needed; these may require external libraries and licensing review.

## User Interface

Go UI options:

- SDL2 via Go bindings: mature for emulator windows, keyboard, audio, and game controllers, but CGO/native dependency heavy.
- Ebiten: pure-Go-friendly game loop, simple keyboard/audio/window handling, good for fast iteration.
- Fyne/Gio: better for app-style UI, less ideal for low-latency emulation surfaces.

Recommendation: use Ebiten for the first playable emulator unless SDL-level control becomes necessary. Keep the emulator core independent of UI so SDL, headless tests, or web frontends can be added later.

CLI shape:

```text
cpcgo --rom cpc6128.rom [--amsdos amsdos.rom] [--disk game.dsk] [--model 6128] [--scale 2]
```

## Package Layout

Proposed structure:

```text
cmd/cpcgo/              CLI and UI startup
internal/cpc/           Machine orchestration and scheduler
internal/bus/           Memory map and I/O dispatch
internal/z80/           Adapter around chosen CPU core
internal/gatearray/     Palette, ROM/RAM control, interrupts, renderer
internal/crtc/          6845-compatible CRTC
internal/ppi/           8255 PPI
internal/psg/           AY-3-8912
internal/keyboard/      CPC matrix and host key mapping
internal/fdc/           uPD765-compatible controller
internal/dsk/           DSK/extended DSK parser
internal/debugger/      Trace, breakpoints, memory/register inspection
testdata/               Small legal test ROMs/programs and metadata
```

Keep all hardware packages free of UI imports.

## Testing Strategy

CPU:

- Use the selected Z80 core's upstream tests.
- Add CPC bus adapter tests for memory reads/writes, ROM overlay writes, and I/O dispatch.

ROM boot:

- Golden test that runs from reset for a fixed cycle count and checks expected firmware state.
- Optional screenshot hash once rendering stabilizes.

Hardware unit tests:

- Gate Array command decoding.
- RAM banking table.
- ROM enable/disable behavior.
- CRTC register read/write and sync timing.
- PPI mode/control behavior used by keyboard/PSG.
- PSG register latch/data flow through PPI.
- DSK parser against known small images.
- FDC command/result sequences for `READ ID`, `SEEK`, and `READ DATA`.

Compatibility tests:

- Boot to BASIC prompt.
- Keyboard input `PRINT 1+1`.
- Load and run simple BASIC program.
- Load directory from DSK via AMSDOS.
- Run a small known game/demo corpus and compare screenshots/audio smoke checks.

## Milestones

1. Repository foundation:
   - Create Go module.
   - Pick UI library.
   - Pick/evaluate Z80 core.
   - Load/split ROMs.
   - Add machine skeleton and tests for RAM/ROM banking.

2. Firmware boot:
   - Implement bus, Gate Array ROM controls, basic CRTC ports, and enough PPI behavior for firmware.
   - Run CPU from reset.
   - Add debug trace and instruction limit.

3. First video:
   - Implement palette, modes 0/1/2, and framebuffer output.
   - Show boot screen/BASIC prompt.

4. Keyboard:
   - Implement matrix scanning through PSG/PPI.
   - Type commands into BASIC.

5. Audio:
   - Implement AY registers and sample generation.
   - Verify BASIC `SOUND` commands and game audio smoke tests.

6. Disk:
   - Load AMSDOS ROM.
   - Parse standard DSK.
   - Implement enough uPD765 behavior for `CAT`, `LOAD`, and common games.

7. Accuracy pass:
   - Replace frame rendering with scanline-aware rendering.
   - Tighten interrupt timing.
   - Add CRTC type profiles.
   - Add save states and debugger.

8. Compatibility hardening:
   - Extended DSK.
   - Edge-case FDC behavior.
   - Demos and raster effects.
   - Configurable input and joystick profiles.

## Open Questions

- Should the first target be booting to BASIC only, or should disk games be required in the first playable release?
- Are you comfortable adding an additional AMSDOS ROM file for disk support, if `cpc6128.rom` is only OS+BASIC?
- Do you prefer a permissive-license project, or is GPL acceptable if Caprice32 code becomes more than a reference?
- Preferred UI direction: pure Go/Ebiten for faster setup, or SDL2 for a more traditional emulator stack?
- Should the emulator prioritize developer/debugger features early, or focus first on running games?
