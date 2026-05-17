import { useEffect, useRef } from 'react';
import { createChart, IChartApi, ISeriesApi, LineData, Time } from 'lightweight-charts';
import type { Alert } from '../api/client';

interface Props {
  symbol: string;
  points: { time: string; price: number }[];
  alerts: Alert[];
  livePrice?: number;
}

export function StockChart({ symbol, points, alerts, livePrice }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<'Line'> | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;
    const chart = createChart(containerRef.current, {
      height: 220,
      layout: { background: { color: '#0f1419' }, textColor: '#c8d1dc' },
      grid: { vertLines: { color: '#1e2833' }, horzLines: { color: '#1e2833' } },
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
  }, [symbol]);

  useEffect(() => {
    if (!seriesRef.current) return;
    const data: LineData[] = points.map((p) => ({
      time: (new Date(p.time).getTime() / 1000) as Time,
      value: p.price,
    }));
    if (livePrice != null && data.length > 0) {
      data.push({ time: (Date.now() / 1000) as Time, value: livePrice });
    }
    seriesRef.current.setData(data);
    chartRef.current?.timeScale().fitContent();
  }, [points, livePrice]);

  useEffect(() => {
    if (!seriesRef.current) return;
    const markers = alerts
      .filter((a) => a.enabled)
      .map((a) => {
        const v = parseFloat(a.threshold);
        return { price: v, color: '#f59e0b', title: a.alert_type };
      });
    // lightweight-charts v4 uses price lines
    const lines = markers.map((m) =>
      seriesRef.current!.createPriceLine({
        price: m.price,
        color: m.color,
        lineWidth: 1,
        lineStyle: 2,
        axisLabelVisible: true,
        title: m.title,
      }),
    );
    return () => lines.forEach((l) => seriesRef.current?.removePriceLine(l));
  }, [alerts]);

  return <div ref={containerRef} className="chart-container" />;
}
