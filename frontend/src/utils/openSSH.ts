import type { ConnectionConfig } from '../types/session'

const recentPrefix = 'openssh:'

export function isOpenSSHHost(host: string): boolean {
  return /^[a-z0-9_][a-z0-9_.-]*$/i.test(host)
}

export function openSSHConfig(host: string): ConnectionConfig {
  return {
    id: recentPrefix + host,
    name: host,
    type: 'local',
    host,
    port: 22,
    user: '',
    authType: 'agent',
    openSSHHost: host,
  }
}

export function openSSHRecentConfig(id: string): ConnectionConfig | undefined {
  if (!id.startsWith(recentPrefix)) return
  const host = id.slice(recentPrefix.length)
  if (isOpenSSHHost(host)) return openSSHConfig(host)
}
