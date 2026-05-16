# Stage 3 Firmware Boot Skeleton

Implemented pieces:

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
- Machine wiring that registers Gate Array, ROM select, CRTC, and PPI devices on the I/O bus.
- CPU-level test using real Z80 `OUT (C),A` instructions to drive memory controls through I/O.

Current deliberate limits:

- Gate Array interrupt timing is not implemented yet; the interrupt reset bit is only recorded.
- CRTC has register storage only, not counters, sync generation, or display address generation.
- PPI does not yet integrate PSG, keyboard matrix, cassette, or live VSync.
- Gate Array palette values are stored as hardware colour numbers only; RGB conversion is a later video-rendering task.

Verification:

- `./test.sh`
