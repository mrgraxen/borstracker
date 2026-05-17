package alerts

import (
	"time"

	"github.com/graxe/borstracker/internal/models"
	"github.com/shopspring/decimal"
)

// Evaluate checks whether an alert should fire for the given quote.
func Evaluate(alert models.Alert, price, open decimal.Decimal, now time.Time) (bool, string) {
	if !alert.Enabled {
		return false, ""
	}
	if alert.LastTriggered != nil {
		elapsed := now.Sub(*alert.LastTriggered)
		if elapsed < time.Duration(alert.CooldownSec)*time.Second {
			return false, ""
		}
	}

	switch alert.AlertType {
	case models.AlertAbsoluteBelow:
		if price.LessThan(alert.Threshold) {
			return true, formatMsg(alert, price, "below")
		}
	case models.AlertAbsoluteAbove:
		if price.GreaterThan(alert.Threshold) {
			return true, formatMsg(alert, price, "above")
		}
	case models.AlertPctBelowOpen:
		if open.IsZero() {
			return false, ""
		}
		pct := pctChange(price, open)
		if pct.LessThan(alert.Threshold) {
			return true, formatPctMsg(alert, price, pct, "below")
		}
	case models.AlertPctAboveOpen:
		if open.IsZero() {
			return false, ""
		}
		pct := pctChange(price, open)
		if pct.GreaterThan(alert.Threshold) {
			return true, formatPctMsg(alert, price, pct, "above")
		}
	}
	return false, ""
}

func pctChange(price, open decimal.Decimal) decimal.Decimal {
	if open.IsZero() {
		return decimal.Zero
	}
	return price.Sub(open).Div(open).Mul(decimal.NewFromInt(100))
}

func formatMsg(a models.Alert, price decimal.Decimal, dir string) string {
	return a.Symbol + " " + dir + " " + a.Threshold.StringFixed(2) + " (now " + price.StringFixed(2) + ")"
}

func formatPctMsg(a models.Alert, price, pct decimal.Decimal, dir string) string {
	return a.Symbol + " " + dir + " " + a.Threshold.StringFixed(2) + "% from open (now " + price.StringFixed(2) + ", " + pct.StringFixed(2) + "%)"
}
