import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { api, Alert, AlertEvent, Settings, WatchlistRow } from './api/client';
import { playAlertSound } from './audio/sounds';
import { StockChart } from './components/StockChart';
import { SymbolSearch } from './components/SymbolSearch';
import { useWebSocket, PriceUpdate, AlertUpdate } from './ws/useWebSocket';

type ChartRange = '1d' | '1w' | '1m' | '3m' | '1y';

export default function App() {
  const { t, i18n } = useTranslation();
  const [settings, setSettings] = useState<Settings | null>(null);
  const [watchlist, setWatchlist] = useState<WatchlistRow[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [history, setHistory] = useState<AlertEvent[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [chartRange, setChartRange] = useState<ChartRange>('1d');
  const [chartPoints, setChartPoints] = useState<{ time: string; price: number }[]>([]);
  const [livePrices, setLivePrices] = useState<Record<string, number>>({});
  const [toasts, setToasts] = useState<string[]>([]);
  const [gdprOk, setGdprOk] = useState(() => localStorage.getItem('gdpr_ack') === '1');

  const [newAlert, setNewAlert] = useState({
    alertType: 'absolute_below',
    threshold: '',
    cooldownSec: '300',
  });

  const refresh = useCallback(async () => {
    try {
      const [s, w, a, h] = await Promise.all([
        api.getSettings(),
        api.getWatchlist(),
        api.getAlerts(),
        api.getHistory(),
      ]);
      setSettings(s);
      i18n.changeLanguage(s.language);
      setWatchlist(w.items ?? []);
      setAlerts(a.alerts ?? []);
      setHistory(h.events ?? []);
      setSelected((prev) => prev ?? w.items?.[0]?.symbol ?? null);
    } catch {
      setWatchlist((prev) => prev ?? []);
      setAlerts((prev) => prev ?? []);
      setHistory((prev) => prev ?? []);
    }
  }, [i18n]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    if (!selected) return;
    void api.getChart(selected, chartRange).then((c) => setChartPoints(c.points));
  }, [selected, chartRange]);

  const onPrice = useCallback(
    (msg: PriceUpdate) => {
      setLivePrices((prev) => ({ ...prev, [msg.symbol]: msg.price }));
      setWatchlist((prev) =>
        prev.map((row) =>
          row.symbol === msg.symbol
            ? { ...row, price: msg.price, open: msg.open, stale: msg.stale }
            : row,
        ),
      );
      if (selected === msg.symbol) {
        setChartPoints((prev) => {
          const next = [...prev, { time: msg.ts, price: msg.price }];
          return next.slice(-500);
        });
      }
    },
    [selected],
  );

  const onAlert = useCallback(
    (msg: AlertUpdate) => {
      setToasts((prev) => [msg.message, ...prev].slice(0, 5));
      void refresh();
      if (settings?.sound_enabled) {
        void playAlertSound(settings.sound_id);
      }
    },
    [refresh, settings],
  );

  useWebSocket(onPrice, onAlert);

  const changeLanguage = async (lang: string) => {
    i18n.changeLanguage(lang);
    const updated = await api.patchSettings({ language: lang });
    setSettings(updated);
  };

  const symbolAlerts = (alerts ?? []).filter((a) => a.symbol === selected);
  const alertTypes = ['absolute_below', 'absolute_above', 'pct_below_open', 'pct_above_open'];

  return (
    <div className="app">
      <header className="header">
        <h1>{t('appTitle')}</h1>
        <div className="header-controls">
          <label>
            {t('language')}
            <select
              value={settings?.language ?? 'sv'}
              onChange={(e) => void changeLanguage(e.target.value)}
            >
              <option value="sv">Svenska</option>
              <option value="en">English</option>
            </select>
          </label>
          <label>
            {t('sound')}
            <select
              value={settings?.sound_id ?? 1}
              onChange={(e) =>
                void api.patchSettings({ soundId: Number(e.target.value) }).then(setSettings)
              }
            >
              {[1, 2, 3, 4].map((n) => (
                <option key={n} value={n}>
                  {n}
                </option>
              ))}
            </select>
          </label>
          <button
            type="button"
            onClick={() =>
              void api
                .patchSettings({ soundEnabled: !settings?.sound_enabled })
                .then(setSettings)
            }
          >
            {settings?.sound_enabled ? t('soundOn') : t('soundOff')}
          </button>
        </div>
      </header>

      {!gdprOk && (
        <div className="gdpr-banner">
          <p>{t('gdpr')}</p>
          <button
            type="button"
            onClick={() => {
              localStorage.setItem('gdpr_ack', '1');
              setGdprOk(true);
            }}
          >
            {t('gdprDismiss')}
          </button>
        </div>
      )}

      <main className="layout">
        <aside className="sidebar">
          <SymbolSearch onAdded={() => void refresh()} />

          {watchlist.length === 0 ? (
            <p className="muted">{t('noWatchlist')}</p>
          ) : (
            <ul className="watchlist">
              {(watchlist ?? []).map((row) => (
                <li key={row.symbol} className={selected === row.symbol ? 'active' : ''}>
                  <button type="button" onClick={() => setSelected(row.symbol)}>
                    <span className="sym">{row.symbol}</span>
                    <span className="price">
                      {row.price != null ? row.price.toFixed(2) : '—'} {row.currency ?? ''}
                    </span>
                    {row.stale && <span className="badge">{t('stale')}</span>}
                  </button>
                  <button
                    type="button"
                    className="remove"
                    onClick={() => void api.removeSymbol(row.symbol).then(refresh)}
                  >
                    {t('remove')}
                  </button>
                </li>
              ))}
            </ul>
          )}

          <section className="history">
            <h3>{t('history')}</h3>
            <ul>
              {(history ?? []).map((ev) => (
                <li key={ev.id}>
                  <small>{new Date(ev.triggered_at).toLocaleString()}</small>
                  <div>{ev.message}</div>
                </li>
              ))}
            </ul>
          </section>
        </aside>

        <section className="content">
          {toasts.length > 0 && (
            <div className="toasts">
              {toasts.map((msg, i) => (
                <div key={`${msg}-${i}`} className="toast">
                  {msg}
                </div>
              ))}
            </div>
          )}

          {selected ? (
            <>
              <div className="chart-header">
                <h2>{selected}</h2>
                <div className="ranges">
                  {(['1d', '1w', '1m', '3m', '1y'] as ChartRange[]).map((r) => (
                    <button
                      key={r}
                      type="button"
                      className={chartRange === r ? 'active' : ''}
                      onClick={() => setChartRange(r)}
                    >
                      {t(`ranges.${r}`)}
                    </button>
                  ))}
                </div>
              </div>
              <StockChart
                symbol={selected}
                range={chartRange}
                points={chartPoints}
                alerts={symbolAlerts}
                livePrice={livePrices[selected]}
              />

              <section className="alerts-panel">
                <h3>{t('alerts')}</h3>
                <form
                  className="alert-form"
                  onSubmit={(e) => {
                    e.preventDefault();
                    void api
                      .createAlert({
                        symbol: selected,
                        alertType: newAlert.alertType,
                        threshold: parseFloat(newAlert.threshold),
                        cooldownSec: parseInt(newAlert.cooldownSec, 10),
                      })
                      .then(refresh);
                  }}
                >
                  <select
                    value={newAlert.alertType}
                    onChange={(e) => setNewAlert((a) => ({ ...a, alertType: e.target.value }))}
                  >
                    {alertTypes.map((k) => (
                      <option key={k} value={k}>
                        {t(`types.${k}`)}
                      </option>
                    ))}
                  </select>
                  <input
                    type="number"
                    step="any"
                    placeholder={t('threshold')}
                    value={newAlert.threshold}
                    onChange={(e) => setNewAlert((a) => ({ ...a, threshold: e.target.value }))}
                    required
                  />
                  <input
                    type="number"
                    placeholder={t('cooldown')}
                    value={newAlert.cooldownSec}
                    onChange={(e) => setNewAlert((a) => ({ ...a, cooldownSec: e.target.value }))}
                  />
                  <button type="submit">{t('createAlert')}</button>
                </form>

                {symbolAlerts.length === 0 ? (
                  <p className="muted">{t('noAlerts')}</p>
                ) : (
                  <ul className="alert-list">
                    {symbolAlerts.map((a) => (
                      <li key={a.id}>
                        <span>
                          {t(`types.${a.alert_type}`)} @ {a.threshold}
                        </span>
                        <label>
                          <input
                            type="checkbox"
                            checked={a.enabled}
                            onChange={(e) =>
                              void api.patchAlert(a.id, { enabled: e.target.checked }).then(refresh)
                            }
                          />
                          {t('enabled')}
                        </label>
                        <button
                          type="button"
                          onClick={() => void api.deleteAlert(a.id).then(refresh)}
                        >
                          {t('remove')}
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
              </section>
            </>
          ) : (
            <p className="muted">{t('noWatchlist')}</p>
          )}
        </section>
      </main>
    </div>
  );
}
