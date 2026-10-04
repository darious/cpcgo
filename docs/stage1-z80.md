# Stage 1 Z80 Core

cpcgo uses its own Z80 core in `internal/z80`. It replaced the earlier
`github.com/user-none/go-chip-z80` adapter, which counted T-states per
instruction but could not model the CPC's bus timing or MEMPTR.

## Design

- One instruction per `Step`, executed as the real sequence of machine
  cycles: M1 opcode fetches (4 T), memory reads and writes (3 T), I/O cycles
  (4 T) and internal delays.
- `WaitStates` enables CPC timing. The Gate Array holds WAIT active on three
  T-states out of four, so every cycle is delayed until the T-state at which
  the Z80 samples WAIT is a free one (M1 and memory cycles sample in T2, I/O
  cycles in the automatic wait state). This one rule reproduces the CPC's
  documented "NOP" timings for every instruction and puts each I/O access at
  the right point inside its instruction.
- The bus callbacks run after the cycle counter includes the access, so the
  machine can bring the video hardware up to the exact microsecond first.
- Interrupt modes 0/1/2, NMI, HALT, the EI delay, and the interrupt
  acknowledge cycle (with an optional `InterruptAcknowledger` for the data bus
  value).
- Undocumented behaviour: flags 3 and 5, MEMPTR (WZ) including its effect on
  BIT n,(HL), the Q latch for SCF/CCF, SLL, IXH/IXL/IYH/IYL, DDCB result
  copies, block I/O flags, ED aliases.

## Verification

- `CPCGO_ZEX=1 go test ./internal/z80` runs ZEXDOC and ZEXALL (all tests
  pass).
- Unit tests check standard T-states and CPC NOP timings per instruction.
- The cpc-validation `instruction-timing` test measures 82 instruction
  sequences against the raster on real firmware-free hardware set-up and
  matches the reference emulator exactly on all CRTC types.
