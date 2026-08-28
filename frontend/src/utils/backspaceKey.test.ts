import { describe, it, expect } from 'vitest'
import { applyBackspaceKey, type BackspaceKeyMode } from './backspaceKey'

const DEL = '\x7f'
const BS = '\x08'
const VT220 = '\x1b[3~'

// jsdom runs with a win32 user agent, which would make the runtime platform
// detection inside applyBackspaceKey treat every 'local' call as Windows.
// Pass the platform explicitly so these cases stay environment-independent.
const apply = (data: string, mode: BackspaceKeyMode | undefined, connType: string | undefined) =>
  applyBackspaceKey(data, mode, connType, false)

describe('applyBackspaceKey', () => {
  describe('connection type filtering', () => {
    it('passes through for non-terminal-stream types', () => {
      expect(applyBackspaceKey(`a${DEL}b`, 'bs', 'sftp')).toBe(`a${DEL}b`)
      expect(applyBackspaceKey(`a${DEL}b`, 'bs', 'ftp')).toBe(`a${DEL}b`)
      expect(applyBackspaceKey(`a${DEL}b`, 'bs', 'database')).toBe(`a${DEL}b`)
      expect(applyBackspaceKey(`a${DEL}b`, 'bs', 'rdp')).toBe(`a${DEL}b`)
      expect(applyBackspaceKey(`a${DEL}b`, 'bs', 'vnc')).toBe(`a${DEL}b`)
      expect(applyBackspaceKey(`a${DEL}b`, 'bs', 'monitor')).toBe(`a${DEL}b`)
    })

    it('passes through when connType is undefined', () => {
      expect(applyBackspaceKey(`a${DEL}b`, 'bs', undefined)).toBe(`a${DEL}b`)
    })

    it('applies for terminal-stream types', () => {
      for (const t of ['ssh', 'telnet', 'serial', 'mosh', 'local', 'k8s', 'container']) {
        expect(apply(`a${DEL}b`, 'bs', t)).toBe(`a${BS}b`)
        expect(apply(`a${DEL}b`, 'vt220', t)).toBe(`a${VT220}b`)
      }
    })

    it('keeps 0x7F for local terminals on Windows (ConPTY turns 0x08 into Ctrl+Backspace)', () => {
      // conhost synthesizes Ctrl+Backspace for 0x08 → PSReadLine BackwardKillWord
      // deletes the whole word. xterm's native 0x7F stays a single-char erase.
      expect(applyBackspaceKey(`a${DEL}b`, 'bs', 'local', true)).toBe(`a${DEL}b`)
      expect(applyBackspaceKey(`a${DEL}b`, undefined, 'local', true)).toBe(`a${DEL}b`)
      expect(applyBackspaceKey(`a${DEL}b`, 'vt220', 'local', true)).toBe(`a${DEL}b`)
    })

    it('still translates local terminals on non-Windows platforms', () => {
      expect(applyBackspaceKey(`a${DEL}b`, 'bs', 'local', false)).toBe(`a${BS}b`)
      expect(applyBackspaceKey(`a${DEL}b`, 'vt220', 'local', false)).toBe(`a${VT220}b`)
    })

    it('still translates remote terminal types on Windows', () => {
      // Windows ConPTY only affects the local shell; ssh/telnet/serial/etc.
      // talk to a remote PTY where ^H is a single-character erase.
      for (const t of ['ssh', 'telnet', 'serial', 'mosh', 'k8s', 'container']) {
        expect(applyBackspaceKey(`a${DEL}b`, 'bs', t, true)).toBe(`a${BS}b`)
      }
    })
  })

  describe('mode handling', () => {
    it('passes through when mode is "del"', () => {
      expect(apply(`a${DEL}b`, 'del', 'ssh')).toBe(`a${DEL}b`)
    })

    it('defaults to "bs" for terminal-stream types when mode is undefined', () => {
      expect(apply(`a${DEL}b`, undefined, 'ssh')).toBe(`a${BS}b`)
      expect(apply(`a${DEL}b`, undefined, 'telnet')).toBe(`a${BS}b`)
      expect(apply(`a${DEL}b`, undefined, 'serial')).toBe(`a${BS}b`)
    })

    it('replaces 0x7F with 0x08 when mode is "bs"', () => {
      expect(apply(`a${DEL}b`, 'bs', 'ssh')).toBe(`a${BS}b`)
    })

    it('replaces 0x7F with ESC[3~ when mode is "vt220"', () => {
      expect(apply(`a${DEL}b`, 'vt220', 'ssh')).toBe(`a${VT220}b`)
    })
  })

  describe('input shape', () => {
    it('returns data unchanged when no 0x7F is present', () => {
      const data = 'hello world\r\nls -la'
      expect(apply(data, 'bs', 'ssh')).toBe(data)
      expect(apply(data, 'vt220', 'ssh')).toBe(data)
      expect(apply(data, 'del', 'ssh')).toBe(data)
    })

    it('replaces every 0x7F in a multi-byte string', () => {
      expect(apply(`${DEL}${DEL}${DEL}`, 'bs', 'ssh')).toBe(`${BS}${BS}${BS}`)
      expect(apply(`a${DEL}b${DEL}c`, 'bs', 'ssh')).toBe(`a${BS}b${BS}c`)
    })

    it('replaces every 0x7F with VT220 sequence', () => {
      expect(apply(`a${DEL}b${DEL}c`, 'vt220', 'ssh')).toBe(
        `a${VT220}b${VT220}c`,
      )
    })

    it('handles empty string', () => {
      expect(apply('', 'bs', 'ssh')).toBe('')
    })

    it('does not touch other control characters', () => {
      // Tab (0x09) and Enter (\r) must pass through untouched.
      const data = `a\tb\rc${DEL}d`
      expect(apply(data, 'bs', 'ssh')).toBe(`a\tb\rc${BS}d`)
    })

    it('preserves escape sequences surrounding the backspace', () => {
      // e.g. arrow keys from xterm come in as ESC[A — only the bare 0x7F should
      // be translated, not the escape bytes.
      const data = `prefix${DEL}ESC${DEL}`
      expect(apply(data, 'bs', 'ssh')).toBe(`prefix${BS}ESC${BS}`)
    })
  })
})
