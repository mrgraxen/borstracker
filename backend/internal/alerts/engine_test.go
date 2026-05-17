package alerts

import (
	"testing"
	"time"

	"github.com/graxe/borstracker/internal/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dec(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func baseAlert(t models.AlertType, threshold string) models.Alert {
	return models.Alert{
		ID:          1,
		SessionID:   "sess",
		Symbol:      "AAPL",
		AlertType:   t,
		Threshold:   dec(threshold),
		Enabled:     true,
		CooldownSec: 300,
	}
}

func TestEvaluate_AbsoluteBelow(t *testing.T) {
	a := baseAlert(models.AlertAbsoluteBelow, "200")
	ok, msg := Evaluate(a, dec("199.5"), dec("210"), time.Now())
	assert.True(t, ok)
	assert.Contains(t, msg, "AAPL")

	ok, _ = Evaluate(a, dec("200"), dec("210"), time.Now())
	assert.False(t, ok)
}

func TestEvaluate_AbsoluteAbove(t *testing.T) {
	a := baseAlert(models.AlertAbsoluteAbove, "200")
	ok, _ := Evaluate(a, dec("201"), dec("190"), time.Now())
	assert.True(t, ok)

	ok, _ = Evaluate(a, dec("199"), dec("190"), time.Now())
	assert.False(t, ok)
}

func TestEvaluate_PctBelowOpen(t *testing.T) {
	a := baseAlert(models.AlertPctBelowOpen, "-3")
	open := dec("100")
	ok, _ := Evaluate(a, dec("96"), open, time.Now())
	assert.True(t, ok)

	ok, _ = Evaluate(a, dec("97"), open, time.Now())
	assert.False(t, ok)
}

func TestEvaluate_PctAboveOpen(t *testing.T) {
	a := baseAlert(models.AlertPctAboveOpen, "5")
	open := dec("100")
	ok, _ := Evaluate(a, dec("106"), open, time.Now())
	assert.True(t, ok)

	ok, _ = Evaluate(a, dec("104"), open, time.Now())
	assert.False(t, ok)
}

func TestEvaluate_Disabled(t *testing.T) {
	a := baseAlert(models.AlertAbsoluteBelow, "200")
	a.Enabled = false
	ok, _ := Evaluate(a, dec("100"), dec("100"), time.Now())
	assert.False(t, ok)
}

func TestEvaluate_Cooldown(t *testing.T) {
	a := baseAlert(models.AlertAbsoluteBelow, "200")
	now := time.Now()
	a.LastTriggered = ptrTime(now.Add(-1 * time.Minute))
	ok, _ := Evaluate(a, dec("100"), dec("100"), now)
	assert.False(t, ok)

	a.LastTriggered = ptrTime(now.Add(-10 * time.Minute))
	ok, _ = Evaluate(a, dec("100"), dec("100"), now)
	assert.True(t, ok)
}

func TestEvaluate_ZeroOpen(t *testing.T) {
	a := baseAlert(models.AlertPctBelowOpen, "-3")
	ok, _ := Evaluate(a, dec("50"), dec("0"), time.Now())
	assert.False(t, ok)
}

func TestEvaluate_BoundaryExact(t *testing.T) {
	a := baseAlert(models.AlertAbsoluteBelow, "200")
	ok, _ := Evaluate(a, dec("200"), dec("200"), time.Now())
	assert.False(t, ok, "exact threshold should not trigger below")

	a2 := baseAlert(models.AlertAbsoluteAbove, "200")
	ok, _ = Evaluate(a2, dec("200"), dec("200"), time.Now())
	assert.False(t, ok)
}

func TestPctChange(t *testing.T) {
	pct := pctChange(dec("105"), dec("100"))
	require.True(t, pct.Equal(dec("5")))
}
