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

All tests pass. Places where documented hardware behaviour and the reference
disagree were left out of the tests rather than copied (for example the
light pen registers R16/R17 and an early SENSE DRIVE STATUS reporting write
protect).
