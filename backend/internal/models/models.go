package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Session struct {
	ID           string    `json:"id"`
	Language     string    `json:"language"`
	SoundID      int16     `json:"sound_id"`
	SoundEnabled bool      `json:"sound_enabled"`
	CreatedAt    time.Time `json:"created_at"`
	LastSeenAt   time.Time `json:"last_seen_at"`
}

type WatchlistItem struct {
	ID        int64
	SessionID string
	Symbol    string
}

type AlertType string

const (
	AlertAbsoluteBelow AlertType = "absolute_below"
	AlertAbsoluteAbove AlertType = "absolute_above"
	AlertPctBelowOpen  AlertType = "pct_below_open"
	AlertPctAboveOpen  AlertType = "pct_above_open"
)

type Alert struct {
	ID            int64           `json:"id"`
	SessionID     string          `json:"session_id"`
	Symbol        string          `json:"symbol"`
	AlertType     AlertType       `json:"alert_type"`
	Threshold     decimal.Decimal `json:"threshold"`
	Enabled       bool            `json:"enabled"`
	CooldownSec   int             `json:"cooldown_sec"`
	LastTriggered *time.Time      `json:"last_triggered,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

type AlertEvent struct {
	ID          int64           `json:"id"`
	SessionID   string          `json:"session_id"`
	AlertID     int64           `json:"alert_id"`
	Symbol      string          `json:"symbol"`
	Price       decimal.Decimal `json:"price"`
	Message     string          `json:"message"`
	TriggeredAt time.Time       `json:"triggered_at"`
}

type PriceTick struct {
	Time     time.Time
	Symbol   string
	Price    decimal.Decimal
	Open     *decimal.Decimal
	High     *decimal.Decimal
	Low      *decimal.Decimal
	Volume   *int64
	Currency string
	Stale    bool
}

type Quote struct {
	Symbol   string
	Price    decimal.Decimal
	Open     decimal.Decimal
	High     decimal.Decimal
	Low      decimal.Decimal
	Volume   int64
	Currency string
	Stale    bool
	Time     time.Time
}

type ChartPoint struct {
	Time  time.Time `json:"time"`
	Price float64   `json:"price"`
}
