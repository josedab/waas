import { useState } from 'react';
import { DeliveryTrace, TraceStage, PayloadDiff } from '@/services/debugger';

const STATUS_COLORS: Record<string, string> = {
  ok: 'bg-green-500',
  success: 'bg-green-500',
  error: 'bg-red-500',
  failed: 'bg-red-500',
  skipped: 'bg-gray-400',
  pending: 'bg-yellow-500',
  replayed: 'bg-blue-500',
};

const STAGE_ICONS: Record<string, string> = {
  received: '📥',
  validation: '✅',
  transform: '🔄',
  pre_transform: '🔄',
  post_transform: '✨',
  routing: '🔀',
  delivery: '📤',
  response: '📨',
  retry: '🔁',
  dlq: '☠️',
};

interface TraceTimelineProps {
  trace: DeliveryTrace;
  onStageClick: (stage: TraceStage) => void;
  selectedStage?: TraceStage | null;
}

export function TraceTimeline({ trace, onStageClick, selectedStage }: TraceTimelineProps) {
  return (
    <div className="bg-gray-900 rounded-lg p-4">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-white font-semibold">{trace.delivery_id}</h3>
          <p className="text-gray-400 text-sm">{trace.endpoint_url || trace.endpoint_id}</p>
        </div>
        <div className="flex items-center gap-2">
          <span className={`px-2 py-1 rounded text-xs font-medium text-white ${STATUS_COLORS[trace.final_status] || 'bg-gray-500'}`}>
            {trace.final_status}
          </span>
          <span className="text-gray-400 text-sm">{trace.total_duration_ms}ms</span>
        </div>
      </div>

      <div className="relative">
        {/* Timeline bar */}
        <div className="absolute left-4 top-0 bottom-0 w-0.5 bg-gray-700" />

        {trace.stages.map((stage, idx) => {
          const isSelected = selectedStage?.name === stage.name && selectedStage?.timestamp === stage.timestamp;
          return (
            <div
              key={`${stage.name}-${idx}`}
              className={`relative pl-10 py-3 cursor-pointer rounded transition-colors ${
                isSelected ? 'bg-gray-800' : 'hover:bg-gray-800/50'
              }`}
              onClick={() => onStageClick(stage)}
            >
              {/* Timeline dot */}
              <div className={`absolute left-2.5 top-4 w-3 h-3 rounded-full border-2 border-gray-900 ${
                STATUS_COLORS[stage.status] || 'bg-gray-500'
              }`} />

              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span>{STAGE_ICONS[stage.name] || '⚙️'}</span>
                  <span className="text-white font-medium capitalize">{stage.name.replace(/_/g, ' ')}</span>
                  <span className={`px-1.5 py-0.5 rounded text-xs ${
                    stage.status === 'ok' || stage.status === 'success'
                      ? 'bg-green-500/20 text-green-400'
                      : stage.status === 'error' || stage.status === 'failed'
                      ? 'bg-red-500/20 text-red-400'
                      : 'bg-gray-500/20 text-gray-400'
                  }`}>
                    {stage.status}
                  </span>
                </div>
                <span className="text-gray-500 text-sm">{stage.duration_ms}ms</span>
              </div>

              {stage.error && (
                <p className="text-red-400 text-sm mt-1 truncate">{stage.error}</p>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

interface StageDetailProps {
  stage: TraceStage;
  diff?: PayloadDiff | null;
}

export function StageDetail({ stage, diff }: StageDetailProps) {
  const [activeTab, setActiveTab] = useState<'input' | 'output' | 'diff' | 'meta'>('input');

  return (
    <div className="bg-gray-900 rounded-lg p-4">
      <div className="flex items-center gap-2 mb-4">
        <span>{STAGE_ICONS[stage.name] || '⚙️'}</span>
        <h3 className="text-white font-semibold capitalize">{stage.name.replace(/_/g, ' ')}</h3>
        <span className={`px-2 py-0.5 rounded text-xs font-medium ${
          stage.status === 'ok' || stage.status === 'success'
            ? 'bg-green-500/20 text-green-400'
            : 'bg-red-500/20 text-red-400'
        }`}>
          {stage.status}
        </span>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 mb-3 border-b border-gray-700 pb-2">
        {(['input', 'output', 'diff', 'meta'] as const).map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-3 py-1 rounded text-sm capitalize ${
              activeTab === tab
                ? 'bg-blue-600 text-white'
                : 'text-gray-400 hover:text-white hover:bg-gray-800'
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* Content */}
      <div className="bg-gray-950 rounded p-3 max-h-96 overflow-auto">
        {activeTab === 'input' && (
          <pre className="text-green-400 text-sm font-mono whitespace-pre-wrap">
            {formatJSON(stage.input)}
          </pre>
        )}
        {activeTab === 'output' && (
          <pre className="text-blue-400 text-sm font-mono whitespace-pre-wrap">
            {formatJSON(stage.output)}
          </pre>
        )}
        {activeTab === 'diff' && diff && (
          <div className="space-y-2">
            {diff.identical ? (
              <p className="text-gray-400 text-sm">No changes detected</p>
            ) : (
              diff.diffs.map((d, i) => (
                <div key={i} className="border-b border-gray-800 pb-2">
                  <span className="text-gray-400 text-xs">{d.path}</span>
                  <div className={`text-sm ${
                    d.type === 'added' ? 'text-green-400' :
                    d.type === 'removed' ? 'text-red-400' : 'text-yellow-400'
                  }`}>
                    {d.type === 'added' && <span>+ {d.new_value}</span>}
                    {d.type === 'removed' && <span>- {d.old_value}</span>}
                    {d.type === 'changed' && (
                      <>
                        <div className="text-red-400">- {d.old_value}</div>
                        <div className="text-green-400">+ {d.new_value}</div>
                      </>
                    )}
                  </div>
                </div>
              ))
            )}
          </div>
        )}
        {activeTab === 'meta' && (
          <div className="space-y-1">
            <div className="text-sm"><span className="text-gray-500">Duration:</span> <span className="text-white">{stage.duration_ms}ms</span></div>
            <div className="text-sm"><span className="text-gray-500">Timestamp:</span> <span className="text-white">{stage.timestamp}</span></div>
            {stage.metadata && Object.entries(stage.metadata).map(([k, v]) => (
              <div key={k} className="text-sm"><span className="text-gray-500">{k}:</span> <span className="text-white">{v}</span></div>
            ))}
          </div>
        )}
        {activeTab === 'diff' && !diff && (
          <p className="text-gray-400 text-sm">Select a trace to view diffs</p>
        )}
      </div>

      {stage.error && (
        <div className="mt-3 bg-red-500/10 border border-red-500/30 rounded p-3">
          <p className="text-red-400 text-sm font-mono">{stage.error}</p>
        </div>
      )}
    </div>
  );
}

interface ReplayPanelProps {
  deliveryId: string;
  onReplay: (request: { delivery_id: string; payload_override?: string; header_override?: Record<string, string>; endpoint_override?: string }) => void;
  isReplaying: boolean;
}

export function ReplayPanel({ deliveryId, onReplay, isReplaying }: ReplayPanelProps) {
  const [payloadOverride, setPayloadOverride] = useState('');
  const [endpointOverride, setEndpointOverride] = useState('');

  return (
    <div className="bg-gray-900 rounded-lg p-4">
      <h3 className="text-white font-semibold mb-3">🔁 Replay Delivery</h3>

      <div className="space-y-3">
        <div>
          <label className="text-gray-400 text-sm block mb-1">Payload Override (optional)</label>
          <textarea
            className="w-full bg-gray-950 text-green-400 border border-gray-700 rounded p-2 text-sm font-mono h-24 focus:border-blue-500 focus:outline-none"
            placeholder='{"key": "value"}'
            value={payloadOverride}
            onChange={e => setPayloadOverride(e.target.value)}
          />
        </div>
        <div>
          <label className="text-gray-400 text-sm block mb-1">Endpoint Override (optional)</label>
          <input
            className="w-full bg-gray-950 text-white border border-gray-700 rounded p-2 text-sm focus:border-blue-500 focus:outline-none"
            placeholder="endpoint-id"
            value={endpointOverride}
            onChange={e => setEndpointOverride(e.target.value)}
          />
        </div>
        <button
          onClick={() => onReplay({
            delivery_id: deliveryId,
            ...(payloadOverride && { payload_override: payloadOverride }),
            ...(endpointOverride && { endpoint_override: endpointOverride }),
          })}
          disabled={isReplaying}
          className="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 text-white font-medium py-2 px-4 rounded transition-colors"
        >
          {isReplaying ? 'Replaying...' : '▶ Replay Delivery'}
        </button>
      </div>
    </div>
  );
}

function formatJSON(str: string): string {
  if (!str) return '(empty)';
  try {
    return JSON.stringify(JSON.parse(str), null, 2);
  } catch {
    return str;
  }
}
