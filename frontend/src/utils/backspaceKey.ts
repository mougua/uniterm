// Translate xterm.js's default backspace byte (ASCII DEL, 0x7F) into the byte
// sequence the remote end expects. xterm.js always emits 0x7F for the physical
// Backspace key, which works on most modern Linux/macOS PTYs but is silently
// dropped on Huawei/H3C/Cisco network gear and some serial consoles. Those
// devices expect 0x08 (ASCII BS) or ESC[3~ (VT220 Delete).
//
// Only applies to terminal-stream connection types. Non-terminal protocols
// (SFTP, FTP, database, RDP…) never pass through xterm's onData, but the
// defensive type check makes the function safe to call from any code path.
//
// `mode === undefined` falls back to the new default ('bs') to match the
// behavior the ConnectionForm applies to new connections; this is the
// intentional behavior change for issue #456 (MobaXterm ships the same
// default — "Backspace sends ^H" is on by default).
//
// EXCEPTION — Windows local terminals (issue: Backspace deletes the whole
// word): on Windows the local session runs through ConPTY, and conhost's
// input engine deliberately synthesizes a Ctrl+Backspace key event for an
// incoming 0x08 (^H) — the InputStateMachineEngine comment says it outright:
// "Process Ctrl+Bksp to delete whole words" (microsoft/terminal PR #3935).
// PowerShell's PSReadLine then binds Ctrl+Backspace to BackwardKillWord, so
// every plain Backspace press wipes the whole word. xterm's native 0x7F is
// safe: ConPTY translates 0x7F into a plain VK_BACK, which PSReadLine maps
// to BackwardDeleteChar (one character). Unix local PTYs (creack/pty) are
// unaffected — there ^H is a single-character erase, so 'bs' stays fine.
const TERMINAL_STREAM_TYPES = new Set([
  'ssh',
  'telnet',
  'serial',
  'mosh',
  'local',
  'k8s',
  'container',
])

export type BackspaceKeyMode = 'del' | 'bs' | 'vt220'

function detectWindows(): boolean {
  if (typeof navigator === 'undefined') return false
  const ua = navigator.userAgent || ''
  const platform = (navigator as { platform?: string }).platform || ''
  return /Windows|Win32|Win64/i.test(ua) || /^Win/i.test(platform)
}

export function applyBackspaceKey(
  data: string,
  mode: BackspaceKeyMode | undefined,
  connType: string | undefined,
  isWindows?: boolean,
): string {
  if (!connType || !TERMINAL_STREAM_TYPES.has(connType)) return data
  // Windows local terminal must keep xterm's native 0x7F (see file header);
  // explicit per-connection settings are respected for every other type.
  const windows = isWindows ?? detectWindows()
  if (connType === 'local' && windows) return data
  // Undefined → default to 'bs' for terminal-stream types (see file header).
  const effective: BackspaceKeyMode = mode ?? 'bs'
  if (effective === 'del') return data
  if (!data.includes('\x7f')) return data
  const replacement = effective === 'bs' ? '\x08' : '\x1b[3~'
  return data.split('\x7f').join(replacement)
}
