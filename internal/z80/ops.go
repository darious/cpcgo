package z80

// Instruction decoding follows the regular structure of the Z80 opcode map:
// op = xx yyy zzz, with p = yyy>>1 and q = yyy&1.

var (
	sz53  [256]uint8 // S, Z, Y and X flags of a result byte
	sz53p [256]uint8 // as sz53, plus parity in PV
)

func init() {
	for i := 0; i < 256; i++ {
		v := uint8(i)
		f := v & (FlagS | FlagY | FlagX)
		if v == 0 {
			f |= FlagZ
		}
		sz53[i] = f
		parity := uint8(0)
		for b := v; b != 0; b >>= 1 {
			parity ^= b & 1
		}
		if parity == 0 {
			f |= FlagPV
		}
		sz53p[i] = f
	}
}

func parityFlag(v uint8) uint8 { return sz53p[v] & FlagPV }

// ---------------------------------------------------------------------------
// Register access

// hlReg returns HL, IX or IY depending on the active prefix.
func (c *CPU) hlReg() uint16 {
	switch c.idx {
	case 1:
		return c.ix
	case 2:
		return c.iy
	}
	return pair(c.h, c.l)
}

func (c *CPU) setHLReg(v uint16) {
	switch c.idx {
	case 1:
		c.ix = v
	case 2:
		c.iy = v
	default:
		c.h, c.l = hi(v), lo(v)
	}
}

// reg8 reads register r (0-7, never 6) with IXH/IXL/IYH/IYL substitution.
func (c *CPU) reg8(r uint8) uint8 {
	switch r {
	case 4:
		return hi(c.hlReg())
	case 5:
		return lo(c.hlReg())
	}
	return c.plainReg(r)
}

func (c *CPU) setReg8(r uint8, v uint8) {
	switch r {
	case 4:
		c.setHLReg(uint16(v)<<8 | uint16(lo(c.hlReg())))
	case 5:
		c.setHLReg(uint16(hi(c.hlReg()))<<8 | uint16(v))
	default:
		c.setPlainReg(r, v)
	}
}

// plainReg reads register r without index substitution.
func (c *CPU) plainReg(r uint8) uint8 {
	switch r {
	case 0:
		return c.b
	case 1:
		return c.c
	case 2:
		return c.d
	case 3:
		return c.e
	case 4:
		return c.h
	case 5:
		return c.l
	case 7:
		return c.a
	}
	return 0
}

func (c *CPU) setPlainReg(r uint8, v uint8) {
	switch r {
	case 0:
		c.b = v
	case 1:
		c.c = v
	case 2:
		c.d = v
	case 3:
		c.e = v
	case 4:
		c.h = v
	case 5:
		c.l = v
	case 7:
		c.a = v
	}
}

// rp reads register pair p: BC, DE, HL/IX/IY, SP.
func (c *CPU) rp(p uint8) uint16 {
	switch p {
	case 0:
		return pair(c.b, c.c)
	case 1:
		return pair(c.d, c.e)
	case 2:
		return c.hlReg()
	}
	return c.sp
}

func (c *CPU) setRP(p uint8, v uint16) {
	switch p {
	case 0:
		c.b, c.c = hi(v), lo(v)
	case 1:
		c.d, c.e = hi(v), lo(v)
	case 2:
		c.setHLReg(v)
	default:
		c.sp = v
	}
}

// rp2 reads register pair p with AF in place of SP (PUSH/POP).
func (c *CPU) rp2(p uint8) uint16 {
	if p == 3 {
		return pair(c.a, c.f)
	}
	return c.rp(p)
}

func (c *CPU) setRP2(p uint8, v uint16) {
	if p == 3 {
		c.a = hi(v)
		c.setF(lo(v))
		return
	}
	c.setRP(p, v)
}

// memOperand returns the address of the (HL) or (IX+d) operand. For indexed
// forms it fetches the displacement and spends the 5 T-states the Z80 needs
// to add it.
func (c *CPU) memOperand() uint16 {
	if c.idx == 0 {
		return pair(c.h, c.l)
	}
	d := int8(c.imm8())
	c.internal(5)
	addr := c.hlReg() + uint16(d)
	c.wz = addr
	return addr
}

func (c *CPU) condition(y uint8) bool {
	switch y {
	case 0:
		return c.f&FlagZ == 0
	case 1:
		return c.f&FlagZ != 0
	case 2:
		return c.f&FlagC == 0
	case 3:
		return c.f&FlagC != 0
	case 4:
		return c.f&FlagPV == 0
	case 5:
		return c.f&FlagPV != 0
	case 6:
		return c.f&FlagS == 0
	}
	return c.f&FlagS != 0
}

// ---------------------------------------------------------------------------
// Unprefixed opcodes (also used for DD/FD-prefixed ones through c.idx)

func (c *CPU) execute(op uint8, lastQ uint8) {
	x, y, z := op>>6, (op>>3)&7, op&7
	p, q := y>>1, y&1

	switch x {
	case 1:
		if op == 0x76 {
			c.halted = true
			c.pc--
			return
		}
		switch {
		case y == 6:
			addr := c.memOperand()
			c.write(addr, c.plainReg(z))
		case z == 6:
			addr := c.memOperand()
			c.setPlainReg(y, c.read(addr))
		default:
			c.setReg8(y, c.reg8(z))
		}
		return
	case 2:
		var v uint8
		if z == 6 {
			v = c.read(c.memOperand())
		} else {
			v = c.reg8(z)
		}
		c.alu(y, v)
		return
	case 0:
		c.executeX0(y, z, p, q, lastQ)
		return
	}
	c.executeX3(op, y, z, p, q)
}

func (c *CPU) executeX0(y, z, p, q uint8, lastQ uint8) {
	switch z {
	case 0:
		switch y {
		case 0: // NOP
		case 1: // EX AF,AF'
			c.a, c.a2 = c.a2, c.a
			c.f, c.f2 = c.f2, c.f
		case 2: // DJNZ e
			c.internal(1)
			e := int8(c.imm8())
			c.b--
			if c.b != 0 {
				c.internal(5)
				c.pc += uint16(e)
				c.wz = c.pc
			}
		case 3: // JR e
			e := int8(c.imm8())
			c.internal(5)
			c.pc += uint16(e)
			c.wz = c.pc
		default: // JR cc,e
			e := int8(c.imm8())
			if c.condition(y - 4) {
				c.internal(5)
				c.pc += uint16(e)
				c.wz = c.pc
			}
		}
	case 1:
		if q == 0 { // LD rp,nn
			c.setRP(p, c.imm16())
		} else { // ADD HL,rp
			c.internal(7)
			c.setHLReg(c.add16(c.hlReg(), c.rp(p)))
		}
	case 2:
		switch y {
		case 0: // LD (BC),A
			addr := pair(c.b, c.c)
			c.write(addr, c.a)
			c.wz = pair(c.a, lo(addr+1))
		case 1: // LD A,(BC)
			addr := pair(c.b, c.c)
			c.a = c.read(addr)
			c.wz = addr + 1
		case 2: // LD (DE),A
			addr := pair(c.d, c.e)
			c.write(addr, c.a)
			c.wz = pair(c.a, lo(addr+1))
		case 3: // LD A,(DE)
			addr := pair(c.d, c.e)
			c.a = c.read(addr)
			c.wz = addr + 1
		case 4: // LD (nn),HL
			addr := c.imm16()
			v := c.hlReg()
			c.write(addr, lo(v))
			c.write(addr+1, hi(v))
			c.wz = addr + 1
		case 5: // LD HL,(nn)
			addr := c.imm16()
			l := c.read(addr)
			h := c.read(addr + 1)
			c.setHLReg(pair(h, l))
			c.wz = addr + 1
		case 6: // LD (nn),A
			addr := c.imm16()
			c.write(addr, c.a)
			c.wz = pair(c.a, lo(addr+1))
		case 7: // LD A,(nn)
			addr := c.imm16()
			c.a = c.read(addr)
			c.wz = addr + 1
		}
	case 3: // INC rp / DEC rp
		c.internal(2)
		if q == 0 {
			c.setRP(p, c.rp(p)+1)
		} else {
			c.setRP(p, c.rp(p)-1)
		}
	case 4, 5: // INC r / DEC r
		if y == 6 {
			addr := c.memOperand()
			v := c.read(addr)
			c.internal(1)
			c.write(addr, c.incdec(v, z == 5))
		} else {
			c.setReg8(y, c.incdec(c.reg8(y), z == 5))
		}
	case 6: // LD r,n
		if y == 6 {
			if c.idx == 0 {
				c.write(pair(c.h, c.l), c.imm8())
			} else {
				d := int8(c.imm8())
				n := c.imm8()
				c.internal(2)
				addr := c.hlReg() + uint16(d)
				c.wz = addr
				c.write(addr, n)
			}
		} else {
			c.setReg8(y, c.imm8())
		}
	case 7:
		c.accumulatorOp(y, lastQ)
	}
}

func (c *CPU) accumulatorOp(y uint8, lastQ uint8) {
	keep := c.f & (FlagS | FlagZ | FlagPV)
	switch y {
	case 0: // RLCA
		c.a = c.a<<1 | c.a>>7
		c.setF(keep | c.a&(FlagY|FlagX) | c.a&FlagC)
	case 1: // RRCA
		carry := c.a & 1
		c.a = c.a>>1 | c.a<<7
		c.setF(keep | c.a&(FlagY|FlagX) | carry)
	case 2: // RLA
		carry := c.a >> 7
		c.a = c.a<<1 | c.f&FlagC
		c.setF(keep | c.a&(FlagY|FlagX) | carry)
	case 3: // RRA
		carry := c.a & 1
		c.a = c.a>>1 | (c.f&FlagC)<<7
		c.setF(keep | c.a&(FlagY|FlagX) | carry)
	case 4: // DAA
		c.daa()
	case 5: // CPL
		c.a = ^c.a
		c.setF(c.f&(FlagS|FlagZ|FlagPV|FlagC) | FlagH | FlagN | c.a&(FlagY|FlagX))
	case 6: // SCF
		xy := ((lastQ ^ c.f) | c.a) & (FlagY | FlagX)
		c.setF(keep | FlagC | xy)
	case 7: // CCF
		xy := ((lastQ ^ c.f) | c.a) & (FlagY | FlagX)
		f := keep | xy
		if c.f&FlagC != 0 {
			f |= FlagH
		} else {
			f |= FlagC
		}
		c.setF(f)
	}
}

func (c *CPU) executeX3(op, y, z, p, q uint8) {
	switch z {
	case 0: // RET cc
		c.internal(1)
		if c.condition(y) {
			c.pc = c.pop()
			c.wz = c.pc
		}
	case 1:
		if q == 0 { // POP rp2
			c.setRP2(p, c.pop())
			return
		}
		switch p {
		case 0: // RET
			c.pc = c.pop()
			c.wz = c.pc
		case 1: // EXX
			c.b, c.b2 = c.b2, c.b
			c.c, c.c2 = c.c2, c.c
			c.d, c.d2 = c.d2, c.d
			c.e, c.e2 = c.e2, c.e
			c.h, c.h2 = c.h2, c.h
			c.l, c.l2 = c.l2, c.l
		case 2: // JP (HL)
			c.pc = c.hlReg()
		case 3: // LD SP,HL
			c.internal(2)
			c.sp = c.hlReg()
		}
	case 2: // JP cc,nn
		addr := c.imm16()
		c.wz = addr
		if c.condition(y) {
			c.pc = addr
		}
	case 3:
		switch y {
		case 0: // JP nn
			c.pc = c.imm16()
			c.wz = c.pc
		case 1: // CB prefix
			if c.idx == 0 {
				c.executeCB(c.fetchOpcode())
			} else {
				c.executeIndexedCB()
			}
		case 2: // OUT (n),A
			n := c.imm8()
			c.out(pair(c.a, n), c.a)
			c.wz = pair(c.a, n+1)
		case 3: // IN A,(n)
			n := c.imm8()
			port := pair(c.a, n)
			c.a = c.in(port)
			c.wz = port + 1
		case 4: // EX (SP),HL
			l := c.read(c.sp)
			h := c.read(c.sp + 1)
			c.internal(1)
			v := c.hlReg()
			c.write(c.sp+1, hi(v))
			c.write(c.sp, lo(v))
			c.internal(2)
			c.setHLReg(pair(h, l))
			c.wz = pair(h, l)
		case 5: // EX DE,HL (never IX/IY)
			c.d, c.h = c.h, c.d
			c.e, c.l = c.l, c.e
		case 6: // DI
			c.iff1, c.iff2 = false, false
		case 7: // EI
			c.iff1, c.iff2 = true, true
			c.afterEI = true
		}
	case 4: // CALL cc,nn
		addr := c.imm16()
		c.wz = addr
		if c.condition(y) {
			c.internal(1)
			c.push(c.pc)
			c.pc = addr
		}
	case 5:
		if q == 0 { // PUSH rp2
			c.internal(1)
			c.push(c.rp2(p))
			return
		}
		switch p {
		case 0: // CALL nn
			addr := c.imm16()
			c.wz = addr
			c.internal(1)
			c.push(c.pc)
			c.pc = addr
		case 1: // DD prefix
			c.executeIndexed(1)
		case 2: // ED prefix
			c.idx = 0
			c.executeED(c.fetchOpcode())
		case 3: // FD prefix
			c.executeIndexed(2)
		}
	case 6: // ALU A,n
		c.alu(y, c.imm8())
	case 7: // RST
		c.internal(1)
		c.push(c.pc)
		c.pc = uint16(op & 0x38)
		c.wz = c.pc
	}
}

// executeIndexed handles DD/FD prefixes. Further DD/FD prefixes simply replace
// the active index register; ED cancels it.
func (c *CPU) executeIndexed(idx int) {
	for {
		c.idx = idx
		op := c.fetchOpcode()
		switch op {
		case 0xdd:
			idx = 1
			continue
		case 0xfd:
			idx = 2
			continue
		case 0xed:
			c.idx = 0
			c.executeED(c.fetchOpcode())
			return
		}
		c.execute(op, c.q)
		return
	}
}

// ---------------------------------------------------------------------------
// ALU

func (c *CPU) alu(op uint8, v uint8) {
	switch op {
	case 0:
		c.a = c.add8(c.a, v, 0)
	case 1:
		c.a = c.add8(c.a, v, c.f&FlagC)
	case 2:
		c.a = c.sub8(c.a, v, 0)
	case 3:
		c.a = c.sub8(c.a, v, c.f&FlagC)
	case 4:
		c.a &= v
		c.setF(sz53p[c.a] | FlagH)
	case 5:
		c.a ^= v
		c.setF(sz53p[c.a])
	case 6:
		c.a |= v
		c.setF(sz53p[c.a])
	case 7: // CP: undocumented flags come from the operand
		c.sub8(c.a, v, 0)
		c.setF(c.f&^(FlagY|FlagX) | v&(FlagY|FlagX))
	}
}

func (c *CPU) add8(a, v, carry uint8) uint8 {
	r := uint16(a) + uint16(v) + uint16(carry)
	res := uint8(r)
	f := sz53[res] | (a^v^res)&FlagH
	if (a^^v)&(a^res)&0x80 != 0 {
		f |= FlagPV
	}
	if r > 0xff {
		f |= FlagC
	}
	c.setF(f)
	return res
}

func (c *CPU) sub8(a, v, carry uint8) uint8 {
	r := int(a) - int(v) - int(carry)
	res := uint8(r)
	f := sz53[res] | (a^v^res)&FlagH | FlagN
	if (a^v)&(a^res)&0x80 != 0 {
		f |= FlagPV
	}
	if r < 0 {
		f |= FlagC
	}
	c.setF(f)
	return res
}

func (c *CPU) incdec(v uint8, dec bool) uint8 {
	f := c.f & FlagC
	var r uint8
	if dec {
		r = v - 1
		f |= FlagN
		if v&0x0f == 0 {
			f |= FlagH
		}
		if v == 0x80 {
			f |= FlagPV
		}
	} else {
		r = v + 1
		if r&0x0f == 0 {
			f |= FlagH
		}
		if v == 0x7f {
			f |= FlagPV
		}
	}
	c.setF(f | sz53[r])
	return r
}

func (c *CPU) add16(a, v uint16) uint16 {
	r := uint32(a) + uint32(v)
	res := uint16(r)
	f := c.f&(FlagS|FlagZ|FlagPV) | hi(res)&(FlagY|FlagX) | hi(a^v^res)&FlagH
	if r > 0xffff {
		f |= FlagC
	}
	c.setF(f)
	c.wz = a + 1
	return res
}

func (c *CPU) adc16(a, v uint16) uint16 {
	r := uint32(a) + uint32(v) + uint32(c.f&FlagC)
	res := uint16(r)
	f := hi(res)&(FlagS|FlagY|FlagX) | hi(a^v^res)&FlagH
	if res == 0 {
		f |= FlagZ
	}
	if (a^^v)&(a^res)&0x8000 != 0 {
		f |= FlagPV
	}
	if r > 0xffff {
		f |= FlagC
	}
	c.setF(f)
	c.wz = a + 1
	return res
}

func (c *CPU) sbc16(a, v uint16) uint16 {
	r := int(a) - int(v) - int(c.f&FlagC)
	res := uint16(r)
	f := hi(res)&(FlagS|FlagY|FlagX) | hi(a^v^res)&FlagH | FlagN
	if res == 0 {
		f |= FlagZ
	}
	if (a^v)&(a^res)&0x8000 != 0 {
		f |= FlagPV
	}
	if r < 0 {
		f |= FlagC
	}
	c.setF(f)
	c.wz = a + 1
	return res
}

func (c *CPU) daa() {
	a := c.a
	correction := uint8(0)
	carry := c.f & FlagC
	if c.f&FlagH != 0 || a&0x0f > 9 {
		correction |= 0x06
	}
	if carry != 0 || a > 0x99 {
		correction |= 0x60
		carry = FlagC
	}
	var half bool
	if c.f&FlagN != 0 {
		half = c.f&FlagH != 0 && a&0x0f < 6
		c.a = a - correction
	} else {
		half = a&0x0f > 9
		c.a = a + correction
	}
	f := sz53p[c.a] | c.f&FlagN | carry
	if half {
		f |= FlagH
	}
	c.setF(f)
}

// ---------------------------------------------------------------------------
// CB prefix

// rotate performs shift/rotate operation y on v and sets the flags.
func (c *CPU) rotate(y uint8, v uint8) uint8 {
	var r, carry uint8
	switch y {
	case 0: // RLC
		carry = v >> 7
		r = v<<1 | carry
	case 1: // RRC
		carry = v & 1
		r = v>>1 | carry<<7
	case 2: // RL
		carry = v >> 7
		r = v<<1 | c.f&FlagC
	case 3: // RR
		carry = v & 1
		r = v>>1 | (c.f&FlagC)<<7
	case 4: // SLA
		carry = v >> 7
		r = v << 1
	case 5: // SRA
		carry = v & 1
		r = v>>1 | v&0x80
	case 6: // SLL (undocumented)
		carry = v >> 7
		r = v<<1 | 1
	case 7: // SRL
		carry = v & 1
		r = v >> 1
	}
	c.setF(sz53p[r] | carry)
	return r
}

// bit sets the flags for BIT n,v; xy supplies the undocumented bits 3 and 5.
func (c *CPU) bit(n uint8, v uint8, xy uint8) {
	r := v & (1 << n)
	f := c.f&FlagC | FlagH | xy&(FlagY|FlagX)
	if r == 0 {
		f |= FlagZ | FlagPV
	}
	if r&0x80 != 0 {
		f |= FlagS
	}
	c.setF(f)
}

func (c *CPU) executeCB(op uint8) {
	x, y, z := op>>6, (op>>3)&7, op&7
	if z == 6 {
		addr := pair(c.h, c.l)
		v := c.read(addr)
		c.internal(1)
		switch x {
		case 0:
			c.write(addr, c.rotate(y, v))
		case 1:
			c.bit(y, v, hi(c.wz))
		case 2:
			c.write(addr, v&^(1<<y))
		case 3:
			c.write(addr, v|1<<y)
		}
		return
	}
	v := c.plainReg(z)
	switch x {
	case 0:
		c.setPlainReg(z, c.rotate(y, v))
	case 1:
		c.bit(y, v, v)
	case 2:
		c.setPlainReg(z, v&^(1<<y))
	case 3:
		c.setPlainReg(z, v|1<<y)
	}
}

// executeIndexedCB handles DD CB d op / FD CB d op. The opcode byte is read as
// an ordinary memory read, not an M1 cycle. Non-BIT operations also copy the
// result into register z when z is not 6 (undocumented).
func (c *CPU) executeIndexedCB() {
	d := int8(c.imm8())
	op := c.read(c.pc)
	c.pc++
	c.internal(2)
	addr := c.hlReg() + uint16(d)
	c.wz = addr

	x, y, z := op>>6, (op>>3)&7, op&7
	v := c.read(addr)
	c.internal(1)
	var r uint8
	switch x {
	case 0:
		r = c.rotate(y, v)
	case 1:
		c.bit(y, v, hi(addr))
		return
	case 2:
		r = v &^ (1 << y)
	case 3:
		r = v | 1<<y
	}
	c.write(addr, r)
	if z != 6 {
		c.setPlainReg(z, r)
	}
}

// ---------------------------------------------------------------------------
// ED prefix

func (c *CPU) executeED(op uint8) {
	x, y, z := op>>6, (op>>3)&7, op&7
	p, q := y>>1, y&1

	if x == 2 && z <= 3 && y >= 4 {
		c.blockOp(y, z)
		return
	}
	if x != 1 {
		return // undefined ED opcodes behave as two NOPs
	}

	switch z {
	case 0: // IN r,(C) (r=6: flags only)
		port := pair(c.b, c.c)
		v := c.in(port)
		c.wz = port + 1
		if y != 6 {
			c.setPlainReg(y, v)
		}
		c.setF(c.f&FlagC | sz53p[v])
	case 1: // OUT (C),r (r=6: outputs 0 on NMOS parts)
		port := pair(c.b, c.c)
		v := uint8(0)
		if y != 6 {
			v = c.plainReg(y)
		}
		c.out(port, v)
		c.wz = port + 1
	case 2:
		c.internal(7)
		hl := pair(c.h, c.l)
		var r uint16
		if q == 0 {
			r = c.sbc16(hl, c.rp(p))
		} else {
			r = c.adc16(hl, c.rp(p))
		}
		c.h, c.l = hi(r), lo(r)
	case 3:
		addr := c.imm16()
		if q == 0 { // LD (nn),rp
			v := c.rp(p)
			c.write(addr, lo(v))
			c.write(addr+1, hi(v))
		} else { // LD rp,(nn)
			l := c.read(addr)
			h := c.read(addr + 1)
			c.setRP(p, pair(h, l))
		}
		c.wz = addr + 1
	case 4: // NEG
		c.a = c.sub8(0, c.a, 0)
	case 5: // RETN / RETI
		c.iff1 = c.iff2
		c.pc = c.pop()
		c.wz = c.pc
	case 6: // IM
		c.im = [8]uint8{0, 0, 1, 2, 0, 0, 1, 2}[y]
	case 7:
		switch y {
		case 0: // LD I,A
			c.internal(1)
			c.i = c.a
		case 1: // LD R,A
			c.internal(1)
			c.r = c.a
		case 2: // LD A,I
			c.internal(1)
			c.a = c.i
			c.setIRFlags()
		case 3: // LD A,R
			c.internal(1)
			c.a = c.r
			c.setIRFlags()
		case 4: // RRD
			addr := pair(c.h, c.l)
			v := c.read(addr)
			c.internal(4)
			c.write(addr, c.a<<4|v>>4)
			c.a = c.a&0xf0 | v&0x0f
			c.setF(c.f&FlagC | sz53p[c.a])
			c.wz = addr + 1
		case 5: // RLD
			addr := pair(c.h, c.l)
			v := c.read(addr)
			c.internal(4)
			c.write(addr, v<<4|c.a&0x0f)
			c.a = c.a&0xf0 | v>>4
			c.setF(c.f&FlagC | sz53p[c.a])
			c.wz = addr + 1
		}
	}
}

func (c *CPU) setIRFlags() {
	f := c.f&FlagC | sz53[c.a]
	if c.iff2 {
		f |= FlagPV
	}
	c.setF(f)
}

// blockOp executes LDI/CPI/INI/OUTI and their decrementing and repeating
// variants. y: 4=I, 5=D, 6=IR, 7=DR; z: 0=LD, 1=CP, 2=IN, 3=OUT.
func (c *CPU) blockOp(y, z uint8) {
	dec := y&1 != 0
	repeat := y >= 6
	step := uint16(1)
	if dec {
		step = 0xffff
	}
	hl := pair(c.h, c.l)
	bc := pair(c.b, c.c)

	switch z {
	case 0: // LDI
		v := c.read(hl)
		de := pair(c.d, c.e)
		c.write(de, v)
		c.internal(2)
		hl += step
		de += step
		bc--
		c.h, c.l, c.d, c.e, c.b, c.c = hi(hl), lo(hl), hi(de), lo(de), hi(bc), lo(bc)
		n := v + c.a
		f := c.f&(FlagS|FlagZ|FlagC) | n&FlagX | (n&0x02)<<4
		if bc != 0 {
			f |= FlagPV
		}
		c.setF(f)
		if repeat && bc != 0 {
			c.internal(5)
			c.pc -= 2
			c.wz = c.pc + 1
		}
	case 1: // CPI
		v := c.read(hl)
		c.internal(5)
		hl += step
		bc--
		c.h, c.l, c.b, c.c = hi(hl), lo(hl), hi(bc), lo(bc)
		r := c.a - v
		h := (c.a ^ v ^ r) & FlagH
		n := r
		if h != 0 {
			n--
		}
		f := c.f&FlagC | FlagN | sz53[r]&(FlagS|FlagZ) | h | n&FlagX | (n&0x02)<<4
		if bc != 0 {
			f |= FlagPV
		}
		c.setF(f)
		c.wz += step
		if repeat && bc != 0 && r != 0 {
			c.internal(5)
			c.pc -= 2
			c.wz = c.pc + 1
		}
	case 2: // INI
		c.internal(1)
		v := c.in(bc)
		c.wz = bc + step
		c.write(hl, v)
		c.b--
		hl += step
		c.h, c.l = hi(hl), lo(hl)
		k := uint16(v) + uint16(c.c+uint8(step))
		c.blockIOFlags(v, k)
		if repeat && c.b != 0 {
			c.internal(5)
			c.pc -= 2
		}
	case 3: // OUTI
		c.internal(1)
		v := c.read(hl)
		c.b--
		port := pair(c.b, c.c)
		c.wz = port + step
		c.out(port, v)
		hl += step
		c.h, c.l = hi(hl), lo(hl)
		k := uint16(v) + uint16(c.l)
		c.blockIOFlags(v, k)
		if repeat && c.b != 0 {
			c.internal(5)
			c.pc -= 2
		}
	}
}

func (c *CPU) blockIOFlags(v uint8, k uint16) {
	f := sz53[c.b] | parityFlag(uint8(k)&7^c.b)
	if v&0x80 != 0 {
		f |= FlagN
	}
	if k > 0xff {
		f |= FlagH | FlagC
	}
	c.setF(f)
}
