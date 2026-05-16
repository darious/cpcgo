# Stage 1 Z80 Core Decision

Selected core: `github.com/user-none/go-chip-z80`

Version:

```text
v0.0.0-20260315161243-6c949bf925bb
```

## Why This Core

- MIT licensed, which is compatible with the intended AGPL-3.0-or-later project license.
- Provides a CPC-useful bus API:
  - M1 opcode fetch via `Fetch`,
  - memory read/write,
  - full 16-bit I/O port read/write.
- `Step` returns consumed T-states.
- `StepCycles` supports cycle-budgeted execution without long-term drift.
- Interrupt APIs cover maskable INT and NMI.
- Exposes register snapshots, reset, halted state, cycle count, and serialization support.

## Candidate Not Chosen

`github.com/romychs/z80go` is BSD-3-Clause licensed and has strong stated CPU test coverage, including ZEXALL and Fuse tests. It was not selected for the initial adapter because its bus API does not expose a separate M1 fetch path, and the public execution interface is less directly aligned with the CPC timing model we want.

## Integration Rule

Only `internal/z80` should import the third-party CPU package. Other emulator packages should depend on the local adapter types so the CPU core can be replaced or forked later if timing or undocumented behavior becomes a blocker.
