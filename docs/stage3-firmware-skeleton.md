# Stage 3 Firmware Boot Skeleton

Implemented pieces:

- Headless boot probe with bounded instruction count, CPU register summary, and I/O statistics.
- Gate Array write decoding for `7Fxx`-style ports:
  - pen selection,
  - ink updates,
  - screen mode bits,
  - lower/upper ROM enable control,
  - interrupt reset bit tracking,
  - CPC6128 RAM MMR config writes.
- Upper ROM select latch for `DFxx`-style writes.
- CRTC register select/data write stubs.
- PPI port A/B/C/control stubs.
- AY-3-8912 PSG register latch/read/write path through PPI port A and port C BDIR/BC1 bits.
- PPI keyboard line latch from port C low nibble.
- Machine wiring that registers Gate Array, ROM select, CRTC, and PPI devices on the I/O bus.
- CPU-level test using real Z80 `OUT (C),A` instructions to drive memory controls through I/O.

Current deliberate limits:

- Gate Array interrupt timing is not implemented yet; the interrupt reset bit is only recorded.
- CRTC has register storage only, not counters, sync generation, or display address generation.
- PPI does not yet integrate keyboard matrix, cassette, or live VSync.
- Gate Array palette values are stored as hardware colour numbers only; RGB conversion is a later video-rendering task.

Probe notes from the real ROMs:

```text
go run ./cmd/cpcgo --rom cpc6128.rom --amsdos amsdos.rom --probe-instructions 1000000
```

Observed summary:

- CPU reaches `PC=1ea2`, then a longer 5,000,000 instruction probe stays in the same area around `PC=1eb2`.
- I/O activity stops after the initial setup burst.
- Most setup traffic is handled by Gate Array, PPI, CRTC, and ROM select.
- Remaining unhandled setup ports seen so far: `fb7e`, `fa7e`, `ef7f`, `f8ff`.

Likely next blocker:

- Gate Array/CRTC interrupt and frame timing. The firmware appears to progress through hardware setup and then wait in a timing-dependent loop.

Verification:

- `./test.sh`
