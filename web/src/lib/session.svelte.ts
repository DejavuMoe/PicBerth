import { api, setUnauthorizedHandler, type ApiError } from './api';
import type { SessionInfo } from './types';

export const session = $state({
  loaded: false,
  failure: null as ApiError | null,
  authenticated: false,
  setupRequired: false,
  user: null as SessionInfo['user'],
  limits: null as SessionInfo['limits'],
  version: '',
});

function apply(s: SessionInfo) {
  session.authenticated = s.authenticated;
  session.setupRequired = s.setupRequired;
  session.user = s.user;
  session.limits = s.limits;
  session.version = s.version;
  session.failure = null;
  session.loaded = true;
}

export async function loadSession(): Promise<void> {
  try {
    apply(await api.session());
  } catch (e) {
    session.failure = e as ApiError;
    session.loaded = true;
  }
}

/** Refresh in the background; keeps the current state when the request fails. */
export async function refreshSession(): Promise<void> {
  try {
    apply(await api.session());
  } catch {
    // keep what we have
  }
}

export async function logout(): Promise<void> {
  try {
    await api.logout();
  } finally {
    session.authenticated = false;
    session.user = null;
  }
}

// An expired session (401 anywhere) flips the flag; App's gate redirects to /login?next=.
setUnauthorizedHandler(() => {
  if (session.authenticated) {
    session.authenticated = false;
    session.user = null;
  }
});
