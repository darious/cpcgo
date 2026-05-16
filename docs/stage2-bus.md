# Stage 2 Bus Foundation

Implemented pieces:

- CPC6128 128K RAM model as eight 16K banks.
- CPC6128 RAM configurations 0-7.
- Lower ROM overlay at `0x0000-0x3fff`.
- Upper ROM overlay at `0xc000-0xffff`.
- Upper ROM bank 0 for BASIC and bank 7 for AMSDOS.
- ROM write-through behavior: writes always update underlying RAM.
- Partial I/O dispatch framework.
- Machine wiring from CPU adapter to memory and I/O bus.
- Headless instruction runner for early firmware execution tests.

Current deliberate limits:

- Gate Array command decoding is not implemented here. Later Gate Array work should call the memory control methods for ROM enable/disable and RAM banking.
- CRTC, PPI, PSG, and FDC ports are not decoded yet. They should register devices with `internal/bus.IO`.
- Missing selected upper ROMs currently expose underlying RAM to keep tests and early boot behavior deterministic.

Verification:

- `./test.sh`
