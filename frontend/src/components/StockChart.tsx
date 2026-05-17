import { useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import {
  createChart,
  IChartApi,
  ISeriesApi,
  LineData,
  Time,
  UTCTimestamp,
} from 'lightweight-charts';
import type { Alert } from '../api/client';

type ChartRange = '1d' | '1w' | '1m' | '3m' | '1y';

interface Props {
  symbol: string;
  range: ChartRange;
  points: { time: string; price: number }[];
  alerts: Alert[];
  livePrice?: number;
}

function toChartTime(iso: string, range: ChartRange): Time {
  const d = new Date(iso);
  if (range === '1d') {
    return Math.floor(d.getTime() / 1000) as UTCTimestamp;
  }
  const y = d.getUTCFullYear();
  const m = String(d.getUTCMonth() + 1).padStart(2, '0');
  const day = String(d.getUTCDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

function buildSeriesData(
  points: { time: string; price: number }[],
  range: ChartRange,
  livePrice?: number,
): LineData[] {
  const sorted = [...points]
    .filter((p) => Number.isFinite(p.price))
    .sort((a, b) => new Date(a.time).getTime() - new Date(b.time).getTime());

  const data: LineData[] = sorted.map((p) => ({
    time: toChartTime(p.time, range),
    value: p.price,
  }));

  if (livePrice != null && data.length > 0) {
    const last = data[data.length - 1];
    const liveTime =
      range === '1d'
        ? (Math.floor(Date.now() / 1000) as UTCTimestamp)
        : toChartTime(new Date().toISOString(), range);
    if (liveTime !== last.time) {
      data.push({ time: liveTime, value: livePrice });
    } else {
      data[data.length - 1] = { time: last.time, value: livePrice };
    }
  }

  return data;
}

export function StockChart({ symbol, range, points, alerts, livePrice }: Props) {
  const { i18n } = useTranslation();
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<'Line'> | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;
    const locale = i18n.language === 'sv' ? 'sv-SE' : 'en-US';
    const chart = createChart(containerRef.current, {
      height: 220,
      layout: { background: { color: '#0f1419' }, textColor: '#c8d1dc' },
      grid: { vertLines: { color: '#1e2833' }, horzLines: { color: '#1e2833' } },
      localization: { locale },
      rightPriceScale: { borderVisible: false },
      timeScale: {
        borderVisible: false,
        timeVisible: range === '1d',
        secondsVisible: false,
      },
    });
    const series = chart.addLineSeries({ color: '#3b82f6', lineWidth: 2 });
    chartRef.current = chart;
    seriesRef.current = series;

    const ro = new ResizeObserver(() => {
      if (containerRef.current) {
        chart.applyOptions({ width: containerRef.current.clientWidth });
      }
    });
    ro.observe(containerRef.current);

    return () => {
      ro.disconnect();
      chart.remove();
      chartRef.current = null;
      seriesRef.current = null;
    };
  }, [symbol, range, i18n.language]);

  useEffect(() => {
    if (!seriesRef.current) return;
    const data = buildSeriesData(points, range, livePrice);
    seriesRef.current.setData(data);
    chartRef.current?.timeScale().fitContent();
  }, [points, livePrice, range]);

  useEffect(() => {
    if (!seriesRef.current) return;
    const prices = points.map((p) => p.price);
    if (livePrice != null) prices.push(livePrice);
    const minP = prices.length ? Math.min(...prices) : 0;
    const maxP = prices.length ? Math.max(...prices) : 0;
    const pad = (maxP - minP) * 0.15 || maxP * 0.05 || 1;

    const lines = (alerts ?? [])
      .filter((a) => a.enabled)
      .map((a) => {
        const v = parseFloat(a.threshold);
        return seriesRef.current!.createPriceLine({
          price: v,
          color: '#f59e0b',
          lineWidth: 1,
          lineStyle: 2,
          // Avoid a stray "17" (etc.) on the price scale when the line is far from the quote
          axisLabelVisible: v >= minP - pad && v <= maxP + pad,
          title: a.alert_type,
        });
      });
    return () => lines.forEach((l) => seriesRef.current?.removePriceLine(l));
  }, [alerts, points, livePrice]);

  return <div ref={containerRef} className="chart-container" />;
}
