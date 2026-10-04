# Validation

cpcgo is validated with [cpc-validation](https://github.com/darious/cpc-validation),
an emulator-independent harness. Each test runs the same scenario on a
reference emulator and on cpcgo and compares the artefacts: the 768x536
canonical screen (pixel for pixel), RAM contents, and optionally sound.
[CPCEC](https://github.com/cpcitor/cpcec), driven as a black box through its
runner, provides the expected results. cpcgo's implementation is written
from hardware documentation; CPCEC's source is not used.

## Running

```sh
go build -o cpc-runner-cpcgo ./cmd/cpc-runner-cpcgo   # ROMs next to it or in $CPCGO_ROM_DIR
cd ../cpc-validation
uv run cpc-validation run --runner ../cpcgo/cpc-runner-cpcgo --catalog catalog/
```

## Coverage

| Area | Tests |
|------|-------|
| Boot | 464, 664, 6128 banners and BASIC prompt (pixel-exact) |
| Z80 timing | 82 instruction sequences timed against the raster; interrupt response in IM 0/1/2 and from HALT |
| Gate Array | border/palette changes at microsecond resolution, mode changes per interrupt, raster interrupt counter and VSYNC resynchronisation, RAM banking (6128 and 64K) |
| CRTC | standard and non-standard geometry, overscan (32K page carry), vertical rupture, R1 splits, R8 skew/disable, short VSYNC, register and status reads; on types 0, 1, 2 and 4 |
| PPI/PSG/keyboard | port behaviour, register masks, keyboard matrix through input scripts, BASIC typing of every symbol and line editing |
| Sound | PSG tones and stereo placement, envelope timing, noise rate, BASIC SOUND |
| Disk | uPD765 command results (seek, sense, READ ID, reads with missing, deleted and CRC-error sectors, write, format), AMSDOS CAT, SAVE/LOAD on 464 (DDI-1), 664 and 6128, AmstradDiag |

## Status

87 of 111 tests pass; the 24 failures are newer CRTC probe tests (see below). Where documented
hardware behaviour and the reference disagree, the case was left out of the
tests rather than copied (for example the light pen registers R16/R17, R14/R15
reads on types 1/2, and an early SENSE DRIVE STATUS reporting write protect).

### Failing: CRTC edge-case probes

`catalog/crtc/{hsync-width,hsync-position,line-length,r9-midframe,r12-midframe,r4-overflow}`
each change one CRTC register on every raster interrupt, on CRTC types 0, 1,
2 and 4 (24 tests). They fail on cpcgo. (`r6-midframe`, `r7-midframe` and
`r13-midframe`, written the same way, pass.)

Most of the differences come from the monitor model. cpcgo's monitor snaps
to every HSYNC and to every VSYNC at least 200 lines into a field. A black-box
characterisation of the reference (static screens with different register
values, run through its runner) shows a monitor that behaves like a pair of
phase-locked loops:

- **Horizontal position follows HSYNC (R2)**, 16 pixels per character.
- **HSYNC width (R3):** the picture moves 8 pixels per character for widths
  2-5 and is unchanged from 6 up. This matches the monitor locking on the
  middle of the Gate Array's HSYNC pulse, which starts 2 µs into the CRTC
  pulse and lasts at most 4 µs. Width 1 produces no Gate Array pulse.
- **Line length (R0):** 62-65 µs lines lock with the picture moved by half the
  length difference (static phase error); longer or shorter lines do not lock.
- **Transients:** after HSYNC moves, the picture moves towards the new
  position by 5 pixels per line while the error exceeds 15 pixels, then 3
  while it exceeds 5, then 1 until aligned.
- **Vertical:** VSYNC is accepted only 296-352 lines after the last retrace.
  While locked, the window starts 34 + (L - 312)/2 lines after VSYNC for a
  frame of L lines (the picture stays centred). Out of range the monitor
  ignores VSYNC and retraces every 353 lines, so the picture rolls.

`r9-midframe` produces 262-line frames, which the reference does not lock to.
`r4-overflow` differs on a single scanline. `r12-midframe` differs on every
displayed line, which points at when R12 is latched rather than at the
monitor; it has not been investigated yet.

### Next steps

1. Rebuild the monitor as horizontal and vertical PLLs with the behaviour
   above, and check it against the probes.
2. Investigate R12/R13 latching (r12-midframe) and the one-line R4 overflow
   difference.
3. Further probes: interrupt acceptance edge cases (EI/HALT, LDIR, prefixes,
   LD A,I/R during an interrupt), Gate Array counter reset/acknowledge
   timing, FDC timing (data rate, seek and READ ID timing).
4. Third-party suites (Longshot's SHAKER, Kevin Thacker's Arnold tests) if
   their disks can be obtained: cpcwiki.eu is blocked by the session's network
   policy.

Not implemented: tape (CDT), snapshots (SNA), CPC Plus, interlace, debugger.
