import React, { useState } from 'react';

interface EndpointFormProps {
  onSubmit: (url: string, headers?: Record<string, string>) => void;
  maxEndpoints?: number;
  currentCount: number;
}

export function EndpointForm({ onSubmit, maxEndpoints, currentCount }: EndpointFormProps) {
  const [url, setUrl] = useState('');
  const [headerKey, setHeaderKey] = useState('');
  const [headerValue, setHeaderValue] = useState('');
  const [headers, setHeaders] = useState<Record<string, string>>({});
  const [validationError, setValidationError] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setValidationError('');

    if (!url.trim()) {
      setValidationError('URL is required');
      return;
    }

    try {
      new URL(url);
    } catch {
      setValidationError('Please enter a valid URL (e.g., https://example.com/webhook)');
      return;
    }

    if (maxEndpoints && currentCount >= maxEndpoints) {
      setValidationError(`Maximum of ${maxEndpoints} endpoints reached`);
      return;
    }

    onSubmit(url, Object.keys(headers).length > 0 ? headers : undefined);
    setUrl('');
    setHeaders({});
  };

  const addHeader = () => {
    if (headerKey.trim() && headerValue.trim()) {
      setHeaders({ ...headers, [headerKey.trim()]: headerValue.trim() });
      setHeaderKey('');
      setHeaderValue('');
    }
  };

  const removeHeader = (key: string) => {
    const next = { ...headers };
    delete next[key];
    setHeaders(next);
  };

  return (
    <form className="waas-form" onSubmit={handleSubmit}>
      <div style={{ marginBottom: 12 }}>
        <label>Webhook URL</label>
        <input
          type="url"
          placeholder="https://example.com/webhooks"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          required
        />
      </div>

      <div style={{ marginBottom: 8 }}>
        <label>Custom Headers (optional)</label>
        <div style={{ display: 'flex', gap: 8 }}>
          <input
            placeholder="Header name"
            value={headerKey}
            onChange={(e) => setHeaderKey(e.target.value)}
            style={{ flex: 1 }}
          />
          <input
            placeholder="Header value"
            value={headerValue}
            onChange={(e) => setHeaderValue(e.target.value)}
            style={{ flex: 1 }}
          />
          <button type="button" className="waas-btn waas-btn--sm" onClick={addHeader}>
            Add
          </button>
        </div>
      </div>

      {Object.keys(headers).length > 0 && (
        <div style={{ marginBottom: 12 }}>
          {Object.entries(headers).map(([k, v]) => (
            <div key={k} className="waas-mono" style={{ fontSize: 12, display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0' }}>
              <span>{k}: {v}</span>
              <button type="button" style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--waas-error)' }} onClick={() => removeHeader(k)}>×</button>
            </div>
          ))}
        </div>
      )}

      {validationError && (
        <div style={{ color: 'var(--waas-error)', fontSize: 13, marginBottom: 8 }}>
          {validationError}
        </div>
      )}

      <div className="waas-form-actions">
        <button type="submit" className="waas-btn waas-btn--primary">
          Create Endpoint
        </button>
      </div>
    </form>
  );
}
