import { useEffect, useRef } from 'react';

export interface PriceUpdate {
  type: 'price';
  symbol: string;
  price: number;
  open: number;
  stale: boolean;
  currency?: string;
  ts: string;
}

export interface AlertUpdate {
  type: 'alert';
  alertId: number;
  symbol: string;
  price: number;
  message: string;
}

export type WSMessage = PriceUpdate | AlertUpdate;

function wsURL(): string {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${window.location.host}/api/v1/ws`;
}

export function useWebSocket(
  onPrice: (msg: PriceUpdate) => void,
  onAlert: (msg: AlertUpdate) => void,
) {
  const onPriceRef = useRef(onPrice);
  const onAlertRef = useRef(onAlert);
  onPriceRef.current = onPrice;
  onAlertRef.current = onAlert;

  useEffect(() => {
    let ws: WebSocket | null = null;
    let timer: ReturnType<typeof setTimeout>;

    const connect = () => {
      ws = new WebSocket(wsURL());
      ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(ev.data as string) as WSMessage;
          if (msg.type === 'price') onPriceRef.current(msg);
          if (msg.type === 'alert') onAlertRef.current(msg);
        } catch {
          /* ignore */
        }
      };
      ws.onclose = () => {
        timer = setTimeout(connect, 3000);
      };
    };

    connect();
    return () => {
      clearTimeout(timer);
      ws?.close();
    };
  }, []);
}
