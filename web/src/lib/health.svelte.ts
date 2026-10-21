// A cheap, shared view of /status for the header lamp. Refreshed on demand and every few minutes.

import { api } from './api';
import type { Status } from './types';

export type Lamp = 'off' | 'ok' | 'warn' | 'danger';

export const health = $state({ status: null as Status | null, lamp: 'off' as Lamp, label: '正在读取状态' });

function judge(s: Status): { lamp: Lamp; label: string } {
  if (s.images.broken > 0 || (s.lastCheck && !s.lastCheck.ok)) {
    return { lamp: 'danger', label: '一致性检查发现问题' };
  }
  if (s.warnings.length > 0) return { lamp: 'warn', label: `需要注意 · ${s.warnings.length} 项` };
  return { lamp: 'ok', label: '一切正常' };
}

let inflight: Promise<void> | null = null;

export function refreshHealth(): Promise<void> {
  inflight ??= api
    .status()
    .then((s) => {
      health.status = s;
      Object.assign(health, judge(s));
    })
    .catch(() => {
      // keep the last known state; the lamp only reflects what was actually read
    })
    .finally(() => {
      inflight = null;
    });
  return inflight;
}

/** Start periodic refresh; returns the stop function. */
export function watchHealth(everyMs = 180_000): () => void {
  void refreshHealth();
  const t = setInterval(() => void refreshHealth(), everyMs);
  return () => clearInterval(t);
}
