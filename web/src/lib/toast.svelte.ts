export interface Toast {
  id: number;
  message: string;
  action?: { label: string; run: () => void };
}

interface ToastOptions {
  action?: Toast['action'];
  /** Milliseconds; 0 keeps the toast until dismissed. */
  duration?: number;
}

const MAX = 3;
const DEFAULT_DURATION = 4000;

export const toasts = $state<Toast[]>([]);

/** Shared flag: a fixed bottom bar (selection / unsaved changes) is on screen, toasts sit above it. */
export const bottomBar = $state({ active: false });

let nextId = 1;
const timers = new Map<number, ReturnType<typeof setTimeout>>();
const durations = new Map<number, number>();

function arm(id: number) {
  const ms = durations.get(id) ?? DEFAULT_DURATION;
  if (ms <= 0) return;
  clearTimeout(timers.get(id));
  timers.set(id, setTimeout(() => dismiss(id), ms));
}

export function toast(message: string, options: ToastOptions = {}): number {
  const id = nextId++;
  durations.set(id, options.duration ?? DEFAULT_DURATION);
  toasts.push({ id, message, action: options.action });
  while (toasts.length > MAX) {
    const oldest = toasts[0];
    if (oldest) dismiss(oldest.id);
  }
  arm(id);
  return id;
}

export function dismiss(id: number) {
  clearTimeout(timers.get(id));
  timers.delete(id);
  durations.delete(id);
  const i = toasts.findIndex((t) => t.id === id);
  if (i >= 0) toasts.splice(i, 1);
}

/** Pause auto-dismiss while the pointer or keyboard focus is on a toast. */
export function hold(id: number) {
  clearTimeout(timers.get(id));
  timers.delete(id);
}

export function release(id: number) {
  if (toasts.some((t) => t.id === id)) arm(id);
}
