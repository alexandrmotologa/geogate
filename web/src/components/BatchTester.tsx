import React, { useState } from 'react';
import { Gauge, Play, Download, Search, RefreshCw, Cpu } from 'lucide-react';
import type { BatchLookupResponse } from '../types';

export const BatchTester: React.FC = () => {
  const [inputText, setInputText] = useState<string>(
    `8.8.8.8\n1.1.1.1\n88.198.24.1\n86.120.10.5\n133.242.1.1\n178.168.0.1\n142.250.0.1\n81.2.69.142\n212.27.48.10\n139.130.4.5`
  );
  const [isLoading, setIsLoading] = useState(false);
  const [response, setResponse] = useState<BatchLookupResponse | null>(null);
  const [filterText, setFilterText] = useState('');
  const [error, setError] = useState<string | null>(null);

  const runBatchLookup = async () => {
    setError(null);
    const ips = inputText
      .split(/[\r\n, ]+/)
      .map(s => s.trim())
      .filter(s => s.length > 0);

    if (ips.length === 0) {
      setError('Please provide at least one IP address.');
      return;
    }

    setIsLoading(true);
    try {
      const res = await fetch('/v1/batch/lookup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ips }),
      });

      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        throw new Error(errData.error || `HTTP error ${res.status}`);
      }

      const data: BatchLookupResponse = await res.json();
      setResponse(data);
    } catch (err: any) {
      setError(err.message || 'Batch lookup request failed.');
    } finally {
      setIsLoading(false);
    }
  };

  const loadPreset = (preset: 'scalers' | 'countries' | 'large') => {
    if (preset === 'scalers') {
      setInputText(`8.8.8.8\n8.8.4.4\n1.1.1.1\n1.0.0.1\n9.9.9.9\n208.67.222.222\n88.198.24.1\n133.242.0.1\n86.120.0.1\n178.168.0.1`);
    } else if (preset === 'countries') {
      setInputText(`8.8.8.8\n81.2.69.142\n88.198.24.1\n212.27.48.10\n133.242.0.1\n86.120.0.1\n178.168.0.1\n142.250.0.1\n139.130.4.5\n177.16.0.1\n103.27.0.1\n145.220.0.1\n80.58.0.1\n151.1.0.1\n83.1.0.1\n192.176.0.1\n193.134.0.1\n116.14.0.1\n211.234.0.1\n89.100.0.1`);
    } else if (preset === 'large') {
      // Generate 250 synthetic IP entries for high-throughput stress test
      const generated: string[] = [];
      for (let i = 1; i <= 250; i++) {
        generated.push(`88.198.${(i / 10) | 0}.${(i * 7) % 250 + 1}`);
      }
      setInputText(generated.join('\n'));
    }
  };

  const exportCSV = () => {
    if (!response) return;
    const headers = ['IP', 'Country Code', 'Country Name', 'City', 'EU', 'ASN', 'ASN Org', 'Latitude', 'Longitude'];
    const rows = response.results.map(r => [
      r.ip,
      r.country_code,
      `"${r.country_name}"`,
      `"${r.city}"`,
      r.is_eu ? '1' : '0',
      r.asn.number,
      `"${r.asn.organization}"`,
      r.latitude,
      r.longitude,
    ]);
    const csvContent = 'data:text/csv;charset=utf-8,' + [headers.join(','), ...rows.map(e => e.join(','))].join('\n');
    const encodedUri = encodeURI(csvContent);
    const link = document.createElement('a');
    link.setAttribute('href', encodedUri);
    link.setAttribute('download', `geogate_batch_${Date.now()}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  const exportJSON = () => {
    if (!response) return;
    const blob = new Blob([JSON.stringify(response, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `geogate_batch_${Date.now()}.json`;
    link.click();
    URL.revokeObjectURL(url);
  };

  const filteredResults = response?.results.filter(
    r =>
      r.ip.includes(filterText) ||
      r.country_name.toLowerCase().includes(filterText.toLowerCase()) ||
      r.country_code.toLowerCase().includes(filterText.toLowerCase()) ||
      r.city.toLowerCase().includes(filterText.toLowerCase()) ||
      r.asn.organization.toLowerCase().includes(filterText.toLowerCase())
  );

  return (
    <div className="bg-slate-900/60 rounded-2xl border border-slate-800/80 p-4 sm:p-5 backdrop-blur-sm flex flex-col min-h-[580px]">
      {/* Top Header */}
      <div className="flex flex-wrap items-center justify-between gap-3 pb-4 border-b border-slate-800/80">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-lg bg-indigo-500/20 text-indigo-400 flex items-center justify-center">
            <Gauge className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-slate-100">High-Throughput Batch Pipeline</h3>
            <p className="text-xs text-slate-400">Goroutine parallel worker pool resolving &gt; 150,000 IPs/sec</p>
          </div>
        </div>

        {/* Preset Action Buttons */}
        <div className="flex items-center gap-2">
          <span className="text-xs text-slate-500 font-medium">Presets:</span>
          <button
            onClick={() => loadPreset('scalers')}
            className="px-2.5 py-1 text-xs bg-slate-800 text-slate-300 hover:text-white rounded-lg border border-slate-700 transition-colors"
          >
            DNS & Scalers
          </button>
          <button
            onClick={() => loadPreset('countries')}
            className="px-2.5 py-1 text-xs bg-slate-800 text-slate-300 hover:text-white rounded-lg border border-slate-700 transition-colors"
          >
            20 Countries
          </button>
          <button
            onClick={() => loadPreset('large')}
            className="px-2.5 py-1 text-xs bg-indigo-950/80 text-indigo-300 hover:text-white rounded-lg border border-indigo-800/80 transition-colors"
          >
            250 Benchmark IPs
          </button>
        </div>
      </div>

      {/* Grid: Input Left, Telemetry & Output Right */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-4 flex-1 min-h-0">
        {/* Left Column: Textarea & Run */}
        <div className="flex flex-col h-full space-y-2">
          <label className="text-xs font-semibold text-slate-300 flex items-center justify-between">
            <span>Target IP Addresses (one per line)</span>
            <span className="text-[11px] text-slate-500 font-mono">
              {inputText.split(/[\r\n, ]+/).filter(Boolean).length} IPs
            </span>
          </label>
          <textarea
            value={inputText}
            onChange={e => setInputText(e.target.value)}
            className="w-full flex-1 bg-slate-950/80 border border-slate-800 rounded-xl p-3 text-xs font-mono text-slate-200 resize-none outline-none focus:border-indigo-500 transition-colors"
            placeholder="Paste list of IPv4 / IPv6 addresses..."
          />
          <button
            onClick={runBatchLookup}
            disabled={isLoading}
            className="w-full py-2.5 bg-indigo-600 hover:bg-indigo-500 disabled:bg-slate-800 text-white font-semibold rounded-xl text-xs flex items-center justify-center gap-2 shadow-lg transition-all"
          >
            {isLoading ? (
              <>
                <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                Resolving In-Memory...
              </>
            ) : (
              <>
                <Play className="w-3.5 h-3.5 fill-current" />
                Execute Parallel Batch Lookup
              </>
            )}
          </button>
          {error && <p className="text-xs text-rose-400 font-medium">{error}</p>}
        </div>

        {/* Right Columns: Metrics & Results Table */}
        <div className="md:col-span-2 flex flex-col h-full space-y-3 min-h-0">
          {/* Performance Stats Cards */}
          <div className="grid grid-cols-3 gap-3">
            <div className="bg-slate-950/70 border border-slate-800/80 rounded-xl p-3">
              <span className="text-[11px] text-slate-400 block">Total Resolved</span>
              <span className="text-lg font-bold text-slate-100 font-mono">
                {response ? response.total_processed.toLocaleString() : '—'}
              </span>
            </div>
            <div className="bg-slate-950/70 border border-slate-800/80 rounded-xl p-3">
              <span className="text-[11px] text-slate-400 block">Execution Latency</span>
              <span className="text-lg font-bold text-emerald-400 font-mono">
                {response ? `${response.duration_microseconds.toLocaleString()} μs` : '—'}
              </span>
            </div>
            <div className="bg-slate-950/70 border border-slate-800/80 rounded-xl p-3">
              <span className="text-[11px] text-slate-400 block">Throughput Rate</span>
              <span className="text-lg font-bold text-indigo-400 font-mono">
                {response ? `${Math.round(response.throughput_ips_per_sec).toLocaleString()} IPs/s` : '—'}
              </span>
            </div>
          </div>

          {/* Table Header with Search & Export */}
          <div className="flex items-center justify-between gap-2">
            <div className="relative flex-1">
              <Search className="w-3.5 h-3.5 text-slate-400 absolute left-2.5 top-1/2 -translate-y-1/2" />
              <input
                type="text"
                placeholder="Filter results..."
                value={filterText}
                onChange={e => setFilterText(e.target.value)}
                className="w-full bg-slate-950 border border-slate-800 rounded-lg pl-8 pr-3 py-1.5 text-xs text-slate-200 outline-none"
              />
            </div>
            <div className="flex gap-1.5">
              <button
                onClick={exportCSV}
                disabled={!response}
                className="px-2.5 py-1.5 bg-slate-800 hover:bg-slate-700 disabled:opacity-40 text-slate-300 hover:text-white rounded-lg text-xs font-medium flex items-center gap-1.5 transition-colors border border-slate-700"
              >
                <Download className="w-3.5 h-3.5" />
                CSV
              </button>
              <button
                onClick={exportJSON}
                disabled={!response}
                className="px-2.5 py-1.5 bg-slate-800 hover:bg-slate-700 disabled:opacity-40 text-slate-300 hover:text-white rounded-lg text-xs font-medium flex items-center gap-1.5 transition-colors border border-slate-700"
              >
                <Download className="w-3.5 h-3.5" />
                JSON
              </button>
            </div>
          </div>

          {/* Results Table */}
          <div className="flex-1 overflow-y-auto rounded-xl border border-slate-800/80 bg-slate-950/60 font-mono text-xs">
            {!response ? (
              <div className="h-full flex flex-col items-center justify-center text-center p-6 text-slate-500">
                <Cpu className="w-6 h-6 text-slate-600 mb-2" />
                <p>Run a batch lookup to inspect resolved records</p>
              </div>
            ) : (
              <table className="w-full border-collapse">
                <thead className="sticky top-0 bg-slate-900 border-b border-slate-800 text-slate-400 text-[11px]">
                  <tr>
                    <th className="text-left py-2 px-2.5">IP</th>
                    <th className="text-left py-2 px-2.5">COUNTRY</th>
                    <th className="text-left py-2 px-2.5">CITY</th>
                    <th className="text-left py-2 px-2.5">EU</th>
                    <th className="text-left py-2 px-2.5">ASN & ORG</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800/60">
                  {filteredResults?.map((r, idx) => (
                    <tr key={idx} className="hover:bg-slate-800/40">
                      <td className="py-2 px-2.5 font-bold text-slate-200">{r.ip}</td>
                      <td className="py-2 px-2.5">
                        <span className="px-1.5 py-0.5 rounded bg-indigo-950 border border-indigo-800 text-indigo-300 font-bold text-[10px] mr-1.5">
                          {r.country_code}
                        </span>
                        <span className="text-slate-300">{r.country_name}</span>
                      </td>
                      <td className="py-2 px-2.5 text-slate-300">{r.city || '—'}</td>
                      <td className="py-2 px-2.5">
                        {r.is_eu ? (
                          <span className="text-emerald-400 text-[11px] font-bold">YES</span>
                        ) : (
                          <span className="text-slate-500 text-[11px]">NO</span>
                        )}
                      </td>
                      <td className="py-2 px-2.5 text-slate-400 truncate max-w-[140px]" title={r.asn.organization}>
                        AS{r.asn.number} {r.asn.organization}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
