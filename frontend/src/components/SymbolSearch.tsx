import {
  FormEvent,
  KeyboardEvent,
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
} from 'react';
import { useTranslation } from 'react-i18next';
import { api, SymbolSearchResult } from '../api/client';

interface Props {
  onAdded: () => void;
}

export function SymbolSearch({ onAdded }: Props) {
  const { t } = useTranslation();
  const listId = useId();
  const wrapperRef = useRef<HTMLDivElement>(null);

  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SymbolSearchResult[]>([]);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [highlight, setHighlight] = useState(0);
  const [error, setError] = useState<string | null>(null);

  const addSymbol = useCallback(
    async (symbol: string) => {
      setError(null);
      try {
        await api.addSymbol(symbol);
        setQuery('');
        setResults([]);
        setOpen(false);
        onAdded();
      } catch (e) {
        setError(e instanceof Error ? e.message : t('searchError'));
      }
    },
    [onAdded, t],
  );

  useEffect(() => {
    const q = query.trim();
    if (q.length < 2) {
      setResults([]);
      setOpen(false);
      setLoading(false);
      return;
    }

    const timer = setTimeout(() => {
      setLoading(true);
      void api
        .searchSymbols(q)
        .then((res) => {
          setResults(res.results);
          setOpen(res.results.length > 0);
          setHighlight(0);
        })
        .catch(() => {
          setResults([]);
          setOpen(false);
          setError(t('searchError'));
        })
        .finally(() => setLoading(false));
    }, 300);

    return () => clearTimeout(timer);
  }, [query, t]);

  useEffect(() => {
    const onDocClick = (e: MouseEvent) => {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', onDocClick);
    return () => document.removeEventListener('mousedown', onDocClick);
  }, []);

  const selectResult = (item: SymbolSearchResult) => {
    void addSymbol(item.symbol);
  };

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (open && results[highlight]) {
      selectResult(results[highlight]);
      return;
    }
    if (query.trim()) {
      void addSymbol(query.trim().toUpperCase());
    }
  };

  const onKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (!open || results.length === 0) return;
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setHighlight((h) => (h + 1) % results.length);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setHighlight((h) => (h - 1 + results.length) % results.length);
    } else if (e.key === 'Escape') {
      setOpen(false);
    }
  };

  return (
    <div className="symbol-search" ref={wrapperRef}>
      <form onSubmit={onSubmit} className="add-form">
        <div className="symbol-search-input-wrap">
          <input
            type="text"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setError(null);
            }}
            onFocus={() => results.length > 0 && setOpen(true)}
            onKeyDown={onKeyDown}
            placeholder={t('symbolPlaceholder')}
            autoComplete="off"
            role="combobox"
            aria-expanded={open}
            aria-controls={listId}
            aria-autocomplete="list"
          />
          {loading && <span className="symbol-search-spinner" aria-hidden />}
        </div>
        <button type="submit">{t('add')}</button>
      </form>

      {open && results.length > 0 && (
        <ul className="symbol-search-results" id={listId} role="listbox">
          {results.map((item, i) => (
            <li
              key={item.symbol}
              role="option"
              aria-selected={i === highlight}
              className={i === highlight ? 'active' : ''}
              onMouseEnter={() => setHighlight(i)}
              onMouseDown={(e) => {
                e.preventDefault();
                selectResult(item);
              }}
            >
              <div className="symbol-search-row-main">
                <span className="symbol-search-symbol">{item.symbol}</span>
                <span className="symbol-search-name">{item.name}</span>
              </div>
              {item.venue ? (
                <span className="symbol-search-venue" title={t('exchange')}>
                  {item.venue}
                </span>
              ) : null}
            </li>
          ))}
        </ul>
      )}

      {error && <p className="symbol-search-error">{error}</p>}
    </div>
  );
}
