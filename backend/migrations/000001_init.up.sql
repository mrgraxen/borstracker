CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE sessions (
    id            UUID PRIMARY KEY,
    language      TEXT NOT NULL DEFAULT 'sv' CHECK (language IN ('sv', 'en')),
    sound_id      SMALLINT NOT NULL DEFAULT 1 CHECK (sound_id BETWEEN 1 AND 4),
    sound_enabled BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE watchlist_items (
    id         BIGSERIAL PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    symbol     TEXT NOT NULL,
    UNIQUE (session_id, symbol)
);

CREATE INDEX idx_watchlist_symbol ON watchlist_items(symbol);

CREATE TYPE alert_type AS ENUM (
    'absolute_below',
    'absolute_above',
    'pct_below_open',
    'pct_above_open'
);

CREATE TABLE alerts (
    id             BIGSERIAL PRIMARY KEY,
    session_id     UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    symbol         TEXT NOT NULL,
    alert_type     alert_type NOT NULL,
    threshold      NUMERIC NOT NULL,
    enabled        BOOLEAN NOT NULL DEFAULT true,
    cooldown_sec   INT NOT NULL DEFAULT 300,
    last_triggered TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_alerts_symbol_enabled ON alerts(symbol) WHERE enabled;

CREATE TABLE alert_events (
    id           BIGSERIAL PRIMARY KEY,
    session_id   UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    alert_id     BIGINT NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
    symbol       TEXT NOT NULL,
    price        NUMERIC NOT NULL,
    message      TEXT NOT NULL,
    triggered_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_alert_events_session ON alert_events(session_id, triggered_at DESC);

CREATE TABLE price_ticks (
    time     TIMESTAMPTZ NOT NULL,
    symbol   TEXT NOT NULL,
    price    NUMERIC NOT NULL,
    open     NUMERIC,
    high     NUMERIC,
    low      NUMERIC,
    volume   BIGINT,
    currency TEXT,
    stale    BOOLEAN NOT NULL DEFAULT false
);

SELECT create_hypertable('price_ticks', 'time', if_not_exists => TRUE);

CREATE INDEX idx_price_ticks_symbol_time ON price_ticks (symbol, time DESC);
