import React, { useState, useEffect } from 'react';
import { Activity, Pause, Play, Trash2, ArrowUpRight, RotateCcw, Check, AlertCircle, Clock } from 'lucide-react';
import type { RequestEvent, ReplayResponse } from '../types';

interface ProxyStreamProps {
  upstreamUrl: string;
}

export const ProxyStream: React.FC<ProxyStreamProps> = ({ upstreamUrl }) => {
  const [events, setEvents] = useState<RequestEvent[]>([]);
  const [isPaused, setIsPaused] = useState(false);
  const [isConnected, setIsConnected] = useState(false);
  const [selectedEvent, setSelectedEvent] = useState<RequestEvent | null>(null);

  // Replay modal state
  const [replayModalOpen, setReplayModalOpen] = useState(false);
  const [replayEvent, setReplayEvent] = useState<RequestEvent | null>(null);
  const [replayCountry, setReplayCountry] = useState('JP');
  const [replayConn, setReplayConn] = useState('residential');
  const [replayLatency, setReplayLatency] = useState('none');
  const [replayLoading, setReplayLoading] = useState(false);
  const [replayResult, setReplayResult] = useState<ReplayResponse | null>(null);

  useEffect(() => {
    let eventSource: EventSource | null = null;

    const connectSSE = () => {
      eventSource = new EventSource('/v1/events');

      eventSource.onopen = () => {
        setIsConnected(true);
      };

      eventSource.onmessage = e => {
        if (isPaused) return;
        try {
          const newEvent: RequestEvent = JSON.parse(e.data);
          setEvents(prev => [newEvent, ...prev.slice(0, 99)]);
        } catch (err) {
          console.error('Failed to parse SSE event:', err);
        }
      };

      eventSource.onerror = () => {
        setIsConnected(false);
        if (eventSource) {
          eventSource.close();
          setTimeout(connectSSE, 3000);
        }
      };
    };

    connectSSE();

    return () => {
      if (eventSource) eventSource.close();
    };
  }, [isPaused]);

  const clearEvents = () => {
    setEvents([]);
    setSelectedEvent(null);
  };

  const openReplayModal = (evt: RequestEvent) => {
    setReplayEvent(evt);
    setReplayCountry('JP');
    setReplayResult(null);
    setReplayModalOpen(true);
  };

  const executeReplay = async () => {
    if (!replayEvent) return;
    setReplayLoading(true);
    setReplayResult(null);

    try {
      const res = await fetch('/v1/proxy/replay', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          method: replayEvent.method,
          path: replayEvent.path,
          simulate_country: replayCountry,
          simulate_conn: replayConn !== 'residential' ? replayConn : undefined,
          simulate_latency: replayLatency !== 'none' ? replayLatency : undefined,
        }),
      });
      const data: ReplayResponse = await res.json();
      setReplayResult(data);
    } catch (err: unknown) {
      setReplayResult({
        success: false,
        status_code: 500,
        duration_ms: 0,
        simulated_country: replayCountry,
        injected_headers: {},
        error: err instanceof Error ? err.message : 'Network error during replay',
      });
    } finally {
      setReplayLoading(false);
    }
  };

  const getStatusBadge = (code: number) => {
    if (code >= 200 && code < 300) {
      return 'bg-emerald-950 text-emerald-400 border-emerald-800';
    }
    if (code >= 300 && code < 400) {
      return 'bg-cyan-950 text-cyan-400 border-cyan-800';
    }
    if (code >= 400 && code < 500) {
      return 'bg-amber-950 text-amber-400 border-amber-800';
    }
    return 'bg-rose-950 text-rose-400 border-rose-800';
  };

  return (
    <div className="bg-slate-900/60 rounded-2xl border border-slate-800/80 p-4 sm:p-5 backdrop-blur-sm flex flex-col min-h-[600px] relative">
      {/* Stream Controls Header */}
      <div className="flex flex-wrap items-center justify-between gap-3 pb-4 border-b border-slate-800/80">
        <div className="flex items-center gap-2.5">
          <div className="relative">
            <div className="w-8 h-8 rounded-lg bg-emerald-500/20 text-emerald-400 flex items-center justify-center">
              <Activity className="w-4 h-4" />
            </div>
            <span
              className={`absolute -top-0.5 -right-0.5 w-2.5 h-2.5 rounded-full ring-2 ring-slate-900 ${
                isConnected ? 'bg-emerald-400 animate-pulse' : 'bg-rose-500'
              }`}
            />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h3 className="text-sm font-semibold text-slate-100">Live Proxy Stream</h3>
              <span className="text-[11px] font-mono px-2 py-0.5 rounded-full bg-slate-800 text-slate-300">
                {events.length} captured
              </span>
            </div>
            <p className="text-xs text-slate-400">
              Forwarding to <span className="font-mono text-indigo-400">{upstreamUrl || 'http://localhost:3000'}</span>
            </p>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center gap-2">
          <button
            onClick={() => setIsPaused(!isPaused)}
            className={`px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-colors border ${
              isPaused
                ? 'bg-amber-600/20 text-amber-300 border-amber-700/60'
                : 'bg-slate-800 text-slate-300 border-slate-700 hover:bg-slate-700 hover:text-white'
            }`}
          >
            {isPaused ? <Play className="w-3.5 h-3.5" /> : <Pause className="w-3.5 h-3.5" />}
            {isPaused ? 'Resume' : 'Pause'}
          </button>
          <button
            onClick={clearEvents}
            className="p-1.5 rounded-lg text-slate-400 hover:text-rose-400 hover:bg-rose-500/10 border border-slate-800 transition-colors"
            title="Clear Stream"
          >
            <Trash2 className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Events Table / Stream */}
      <div className="max-h-[330px] overflow-y-auto mt-4 rounded-xl border border-slate-800/80 bg-slate-950/60 font-mono text-xs">
        {events.length === 0 ? (
          <div className="h-full flex flex-col items-center justify-center p-8 text-center text-slate-500 space-y-2">
            <Activity className="w-8 h-8 text-slate-600 animate-bounce" />
            <p className="text-sm font-medium text-slate-400">Waiting for requests through GeoGate...</p>
            <p className="text-xs max-w-sm">
              Make requests to <code className="text-indigo-400">http://localhost:8080/your-endpoint</code> to see live edge header injection.
            </p>
          </div>
        ) : (
          <table className="w-full border-collapse">
            <thead className="sticky top-0 bg-slate-900 border-b border-slate-800 text-slate-400 text-[11px] select-none">
              <tr>
                <th className="text-left py-2.5 px-3">METHOD</th>
                <th className="text-left py-2.5 px-3">PATH</th>
                <th className="text-left py-2.5 px-3">LOCATION</th>
                <th className="text-left py-2.5 px-3">STATUS</th>
                <th className="text-right py-2.5 px-3">LATENCY</th>
                <th className="text-right py-2.5 px-3">ACTIONS</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60">
              {events.map(evt => (
                <tr
                  key={evt.id}
                  onClick={() => setSelectedEvent(evt)}
                  className={`hover:bg-slate-800/40 cursor-pointer transition-colors ${
                    selectedEvent?.id === evt.id ? 'bg-indigo-600/10' : ''
                  }`}
                >
                  <td className="py-2.5 px-3 font-bold">
                    <span
                      className={`px-1.5 py-0.5 rounded text-[10px] ${
                        evt.method === 'GET'
                          ? 'bg-blue-950 text-blue-400 border border-blue-800'
                          : evt.method === 'POST'
                          ? 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                          : 'bg-purple-950 text-purple-400 border border-purple-800'
                      }`}
                    >
                      {evt.method}
                    </span>
                  </td>
                  <td className="py-2.5 px-3 text-slate-200 truncate max-w-[180px]" title={evt.path}>
                    {evt.path}
                  </td>
                  <td className="py-2.5 px-3 text-slate-300">
                    <div className="flex items-center gap-1.5">
                      <span className="px-1.5 py-0.5 rounded bg-indigo-950 border border-indigo-800 text-indigo-300 font-bold text-[10px]">
                        {evt.country_code}
                      </span>
                      <span className="truncate max-w-[80px] text-slate-400">{evt.city}</span>
                      {evt.simulated && (
                        <span className="text-[9px] px-1 rounded bg-amber-950/80 text-amber-400 border border-amber-800">
                          SIM
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="py-2.5 px-3">
                    <span className={`px-1.5 py-0.5 rounded border text-[10px] font-bold ${getStatusBadge(evt.status_code)}`}>
                      {evt.status_code}
                    </span>
                  </td>
                  <td className="py-2.5 px-3 text-right text-slate-400 text-[11px]">
                    <div className="flex items-center justify-end gap-1">
                      {evt.injected_latency && (
                        <span className="flex items-center gap-0.5 text-amber-400 text-[10px]" title={`Chaos injected: ${evt.injected_latency}`}>
                          <Clock className="w-3 h-3" />
                        </span>
                      )}
                      <span>{evt.duration_ms.toFixed(1)} ms</span>
                    </div>
                  </td>
                  <td className="py-2.5 px-3 text-right">
                    <div className="flex items-center justify-end gap-1.5">
                      <button
                        onClick={e => {
                          e.stopPropagation();
                          openReplayModal(evt);
                        }}
                        className="p-1 rounded text-slate-400 hover:text-indigo-300 hover:bg-indigo-500/10 transition-colors"
                        title="Replay as another country..."
                      >
                        <RotateCcw className="w-3.5 h-3.5" />
                      </button>
                      <button
                        onClick={e => {
                          e.stopPropagation();
                          setSelectedEvent(evt);
                        }}
                        className="p-1 rounded text-indigo-400 hover:text-indigo-300 hover:bg-indigo-500/10 transition-colors"
                        title="Inspect Header Diff"
                      >
                        <ArrowUpRight className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Selected Event: Header Diff & Inspector Drawer */}
      {selectedEvent && (
        <div className="mt-3 p-3.5 bg-slate-950 border border-indigo-500/30 rounded-xl font-mono text-xs space-y-2">
          <div className="flex items-center justify-between pb-2 border-b border-slate-800">
            <div className="flex items-center gap-2">
              <span className="text-emerald-400 font-bold">Header Diff Viewer</span>
              <span className="text-slate-400 text-[11px]">
                ({selectedEvent.method} {selectedEvent.path})
              </span>
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => openReplayModal(selectedEvent)}
                className="text-xs px-2 py-0.5 rounded bg-indigo-600 text-white font-sans hover:bg-indigo-500 flex items-center gap-1"
              >
                <RotateCcw className="w-3 h-3" />
                Replay as Country...
              </button>
              <button
                onClick={() => setSelectedEvent(null)}
                className="text-slate-400 hover:text-white text-xs px-2 py-0.5 rounded bg-slate-800 font-sans"
              >
                Close
              </button>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 max-h-40 overflow-y-auto">
            {/* Edge Injected Headers */}
            <div className="space-y-1">
              <span className="text-[10px] uppercase font-bold text-emerald-400 tracking-wider">
                Injected Edge Headers (+GeoGate)
              </span>
              <div className="space-y-1 bg-slate-900/60 p-2 rounded-lg border border-emerald-900/40">
                {Object.entries(selectedEvent.headers).map(([k, v]) => (
                  <div key={k} className="flex items-center justify-between text-[11px]">
                    <span className="text-emerald-400 font-medium">+{k}:</span>
                    <span className="text-slate-200 truncate ml-2">{v}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* Original Request Headers */}
            <div className="space-y-1">
              <span className="text-[10px] uppercase font-bold text-slate-400 tracking-wider">
                Original Client Headers
              </span>
              <div className="space-y-1 bg-slate-900/40 p-2 rounded-lg border border-slate-800">
                {selectedEvent.original_headers && Object.keys(selectedEvent.original_headers).length > 0 ? (
                  Object.entries(selectedEvent.original_headers).map(([k, v]) => (
                    <div key={k} className="flex items-center justify-between text-[11px]">
                      <span className="text-slate-400">{k}:</span>
                      <span className="text-slate-300 truncate ml-2">{v}</span>
                    </div>
                  ))
                ) : (
                  <div className="text-[11px] text-slate-500 italic">No custom client headers</div>
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Replay as Country Modal */}
      {replayModalOpen && replayEvent && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-lg p-5 shadow-2xl space-y-4">
            <div className="flex items-center justify-between pb-3 border-b border-slate-800">
              <div className="flex items-center gap-2">
                <RotateCcw className="w-4 h-4 text-indigo-400" />
                <h3 className="font-bold text-sm text-white">Replay Request as New Persona</h3>
              </div>
              <button
                onClick={() => setReplayModalOpen(false)}
                className="text-slate-400 hover:text-white text-xs px-2 py-1 rounded bg-slate-800"
              >
                Close
              </button>
            </div>

            <div className="text-xs text-slate-300 font-mono bg-slate-950 p-2.5 rounded-xl border border-slate-800">
              <span className="text-indigo-400 font-bold">{replayEvent.method}</span> {replayEvent.path}
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-[11px] font-semibold text-slate-400 block mb-1">Target Country</label>
                <select
                  value={replayCountry}
                  onChange={e => setReplayCountry(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-800 rounded-lg px-2.5 py-1.5 text-xs text-white"
                >
                  <option value="JP">Japan (Tokyo) - AS9370</option>
                  <option value="DE">Germany (Berlin) - Hetzner</option>
                  <option value="RO">Romania (Bucharest) - RDS</option>
                  <option value="US">United States (New York) - Verizon</option>
                  <option value="GB">United Kingdom (London) - BT</option>
                  <option value="BR">Brazil (Sao Paulo) - Claro</option>
                  <option value="IN">India (Mumbai) - Jio</option>
                  <option value="SG">Singapore - StarHub</option>
                  <option value="AU">Australia (Sydney) - Telstra</option>
                  <option value="CH">Switzerland (Zurich) - Swisscom</option>
                </select>
              </div>

              <div>
                <label className="text-[11px] font-semibold text-slate-400 block mb-1">Connection Type</label>
                <select
                  value={replayConn}
                  onChange={e => setReplayConn(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-800 rounded-lg px-2.5 py-1.5 text-xs text-white"
                >
                  <option value="residential">Residential</option>
                  <option value="datacenter">Datacenter (Cloud)</option>
                  <option value="vpn">VPN Exit Node</option>
                  <option value="tor">Tor Onion Exit</option>
                </select>
              </div>
            </div>

            <div>
              <label className="text-[11px] font-semibold text-slate-400 block mb-1">Simulated Latency</label>
              <select
                value={replayLatency}
                onChange={e => setReplayLatency(e.target.value)}
                className="w-full bg-slate-950 border border-slate-800 rounded-lg px-2.5 py-1.5 text-xs text-white"
              >
                <option value="none">0ms (Instant Local)</option>
                <option value="auto">Auto Distance / Physical Haversine</option>
                <option value="100ms">100ms (Transatlantic)</option>
                <option value="250ms">250ms (Transpacific)</option>
              </select>
            </div>

            <button
              onClick={executeReplay}
              disabled={replayLoading}
              className="w-full py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-xs flex items-center justify-center gap-2 transition-all shadow-md disabled:opacity-50"
            >
              {replayLoading ? (
                <span>Replaying to Upstream...</span>
              ) : (
                <>
                  <Play className="w-3.5 h-3.5" />
                  <span>Send Replay Request</span>
                </>
              )}
            </button>

            {replayResult && (
              <div
                className={`p-3 rounded-xl border font-mono text-xs space-y-1.5 ${
                  replayResult.success
                    ? 'bg-emerald-950/40 border-emerald-800/80 text-emerald-300'
                    : 'bg-rose-950/40 border-rose-800/80 text-rose-300'
                }`}
              >
                <div className="flex items-center justify-between font-bold">
                  <div className="flex items-center gap-1.5">
                    {replayResult.success ? (
                      <Check className="w-3.5 h-3.5 text-emerald-400" />
                    ) : (
                      <AlertCircle className="w-3.5 h-3.5 text-rose-400" />
                    )}
                    <span>
                      {replayResult.success ? 'Upstream Responded OK' : 'Replay Failed'} (HTTP {replayResult.status_code})
                    </span>
                  </div>
                  <span>{replayResult.duration_ms.toFixed(1)} ms</span>
                </div>
                {replayResult.upstream_response && (
                  <div className="text-[11px] text-slate-300 bg-slate-950/80 p-2 rounded-lg border border-slate-800 max-h-24 overflow-y-auto">
                    {replayResult.upstream_response}
                  </div>
                )}
                {replayResult.error && (
                  <div className="text-[11px] text-rose-400">{replayResult.error}</div>
                )}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
