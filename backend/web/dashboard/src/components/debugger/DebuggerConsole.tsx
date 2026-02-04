import { useState, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { debuggerService, DeliveryTrace, TraceStage, PayloadDiff } from '@/services/debugger';
import { TraceTimeline, StageDetail, ReplayPanel } from './TraceViewer';

export function DebuggerConsole() {
  const queryClient = useQueryClient();
  const [selectedTrace, setSelectedTrace] = useState<DeliveryTrace | null>(null);
  const [selectedStage, setSelectedStage] = useState<TraceStage | null>(null);
  const [diff, setDiff] = useState<PayloadDiff | null>(null);
  const [searchEndpoint, setSearchEndpoint] = useState('');

  const { data: tracesData, isLoading } = useQuery({
    queryKey: ['debugger-traces', searchEndpoint],
    queryFn: () => debuggerService.listTraces(searchEndpoint || undefined),
  });

  const replayMutation = useMutation({
    mutationFn: debuggerService.replay,
    onSuccess: (newTrace) => {
      queryClient.invalidateQueries({ queryKey: ['debugger-traces'] });
      setSelectedTrace(newTrace);
    },
  });

  const handleTraceSelect = useCallback(async (trace: DeliveryTrace) => {
    setSelectedTrace(trace);
    setSelectedStage(null);
    try {
      const diffResult = await debuggerService.getDiff(trace.delivery_id);
      setDiff(diffResult);
    } catch {
      setDiff(null);
    }
  }, []);

  const handleStageClick = useCallback((stage: TraceStage) => {
    setSelectedStage(stage);
  }, []);

  const traces = tracesData?.traces || [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">🔍 Webhook Debugger</h1>
          <p className="text-gray-400">Inspect delivery lifecycle and replay failed deliveries</p>
        </div>
      </div>

      {/* Search */}
      <div className="flex gap-3">
        <input
          className="flex-1 bg-gray-900 text-white border border-gray-700 rounded-lg px-4 py-2 focus:border-blue-500 focus:outline-none"
          placeholder="Filter by endpoint ID..."
          value={searchEndpoint}
          onChange={e => setSearchEndpoint(e.target.value)}
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Trace List */}
        <div className="lg:col-span-1 space-y-2 max-h-[calc(100vh-280px)] overflow-y-auto">
          {isLoading ? (
            <div className="text-gray-400 text-center py-8">Loading traces...</div>
          ) : traces.length === 0 ? (
            <div className="text-gray-400 text-center py-8">No delivery traces found</div>
          ) : (
            traces.map(trace => (
              <div
                key={trace.id}
                onClick={() => handleTraceSelect(trace)}
                className={`bg-gray-900 rounded-lg p-3 cursor-pointer border transition-colors ${
                  selectedTrace?.id === trace.id
                    ? 'border-blue-500 bg-gray-800'
                    : 'border-gray-800 hover:border-gray-600'
                }`}
              >
                <div className="flex items-center justify-between mb-1">
                  <span className="text-white text-sm font-mono truncate max-w-[200px]">
                    {trace.delivery_id}
                  </span>
                  <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${
                    trace.final_status === 'success' || trace.final_status === 'ok'
                      ? 'bg-green-500/20 text-green-400'
                      : trace.final_status === 'failed' || trace.final_status === 'error'
                      ? 'bg-red-500/20 text-red-400'
                      : 'bg-yellow-500/20 text-yellow-400'
                  }`}>
                    {trace.final_status}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-gray-500 text-xs">{trace.stages.length} stages</span>
                  <span className="text-gray-500 text-xs">{trace.total_duration_ms}ms</span>
                </div>
              </div>
            ))
          )}
        </div>

        {/* Trace Detail */}
        <div className="lg:col-span-2 space-y-4">
          {selectedTrace ? (
            <>
              <TraceTimeline
                trace={selectedTrace}
                onStageClick={handleStageClick}
                selectedStage={selectedStage}
              />

              {selectedStage && (
                <StageDetail stage={selectedStage} diff={diff} />
              )}

              <ReplayPanel
                deliveryId={selectedTrace.delivery_id}
                onReplay={replayMutation.mutate}
                isReplaying={replayMutation.isPending}
              />
            </>
          ) : (
            <div className="bg-gray-900 rounded-lg p-12 text-center">
              <div className="text-4xl mb-3">🔍</div>
              <h3 className="text-white font-semibold mb-1">Select a delivery trace</h3>
              <p className="text-gray-400 text-sm">
                Click on a trace from the list to inspect its delivery lifecycle
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default DebuggerConsole;
