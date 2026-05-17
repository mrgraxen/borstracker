const API_BASE = '/api/v1';

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error((body as { error?: string }).error ?? res.statusText);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return res.json() as Promise<T>;
}

export interface Settings {
  id: string;
  language: string;
  sound_id: number;
  sound_enabled: boolean;
}

export interface WatchlistRow {
  symbol: string;
  price?: number;
  open?: number;
  currency?: string;
  stale: boolean;
}

export interface Alert {
  id: number;
  symbol: string;
  alert_type: string;
  threshold: string;
  enabled: boolean;
  cooldown_sec: number;
}

export interface SymbolSearchResult {
  symbol: string;
  name: string;
  exchange: string;
  exchangeDisplay: string;
  venue: string;
  quoteType: string;
}

export interface AlertEvent {
  id: number;
  symbol: string;
  price: string;
  message: string;
  triggered_at: string;
}

export const api = {
  getSettings: () => request<Settings>('/settings'),
  patchSettings: (body: Partial<{ language: string; soundId: number; soundEnabled: boolean }>) =>
    request<Settings>('/settings', { method: 'PATCH', body: JSON.stringify(body) }),
  searchSymbols: (q: string) =>
    request<{ results: SymbolSearchResult[] }>(
      `/symbols/search?q=${encodeURIComponent(q)}`,
    ),
  getWatchlist: () => request<{ items: WatchlistRow[] }>('/watchlist'),
  addSymbol: (symbol: string) =>
    request<{ symbol: string }>('/watchlist', { method: 'POST', body: JSON.stringify({ symbol }) }),
  removeSymbol: (symbol: string) =>
    request<void>(`/watchlist/${encodeURIComponent(symbol)}`, { method: 'DELETE' }),
  getAlerts: (symbol?: string) =>
    request<{ alerts: Alert[] }>(symbol ? `/alerts?symbol=${encodeURIComponent(symbol)}` : '/alerts'),
  createAlert: (body: {
    symbol: string;
    alertType: string;
    threshold: number;
    cooldownSec?: number;
  }) => request<Alert>('/alerts', { method: 'POST', body: JSON.stringify(body) }),
  patchAlert: (id: number, body: { enabled?: boolean; threshold?: number; cooldownSec?: number }) =>
    request<Alert>(`/alerts/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
  deleteAlert: (id: number) => request<void>(`/alerts/${id}`, { method: 'DELETE' }),
  getHistory: () => request<{ events: AlertEvent[] }>('/alerts/history'),
  getChart: (symbol: string, range: string) =>
    request<{ points: { time: string; price: number }[] }>(
      `/chart/${encodeURIComponent(symbol)}?range=${range}`,
    ),
};
