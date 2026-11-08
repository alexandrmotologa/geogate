import React, { useState } from 'react';
import { Shield, CheckCircle2, XCircle, Play } from 'lucide-react';
import type { GeofencePolicy, GeofenceEvaluationResponse } from '../types';

export const GeofenceMatrix: React.FC = () => {
  const [policyName, setPolicyName] = useState('EU & Trusted Partners');
  const [allowedCountries, setAllowedCountries] = useState('US, CA, GB, DE, FR, RO');
  const [blockedCountries, setBlockedCountries] = useState('RU, CN');
  const [requireEU, setRequireEU] = useState(false);
  const [blockedASNs, setBlockedASNs] = useState('');
  const [testIPs, setTestIPs] = useState('8.8.8.8\n88.198.24.1\n86.120.10.5\n202.96.128.1\n95.173.136.1\n133.242.0.1');

  const [isLoading, setIsLoading] = useState(false);
  const [evaluation, setEvaluation] = useState<GeofenceEvaluationResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const applyPreset = (preset: 'eu' | 'embargo' | 'na_uk') => {
    if (preset === 'eu') {
      setPolicyName('Strict EU GDPR & VAT Boundary');
      setRequireEU(true);
      setAllowedCountries('');
      setBlockedCountries('RU, CN');
      setBlockedASNs('');
    } else if (preset === 'embargo') {
      setPolicyName('High-Risk Embargo Filter');
      setRequireEU(false);
      setAllowedCountries('');
      setBlockedCountries('RU, CN, IR, KP');
      setBlockedASNs('');
    } else if (preset === 'na_uk') {
      setPolicyName('North America & UK Only');
      setRequireEU(false);
      setAllowedCountries('US, CA, GB');
      setBlockedCountries('');
      setBlockedASNs('');
    }
  };

  const evaluateGeofence = async () => {
    setError(null);
    const ips = testIPs
      .split(/[\r\n, ]+/)
      .map(s => s.trim())
      .filter(Boolean);

    if (ips.length === 0) {
      setError('Please provide at least one IP to test.');
      return;
    }

    const policy: GeofencePolicy = {
      name: policyName,
      require_eu: requireEU,
      allowed_countries: allowedCountries
        ? allowedCountries.split(',').map(s => s.trim().toUpperCase()).filter(Boolean)
        : [],
      blocked_countries: blockedCountries
        ? blockedCountries.split(',').map(s => s.trim().toUpperCase()).filter(Boolean)
        : [],
      blocked_asns: blockedASNs
        ? blockedASNs.split(',').map(s => parseInt(s.trim(), 10)).filter(n => !isNaN(n))
        : [],
    };

    setIsLoading(true);
    try {
      const res = await fetch('/v1/geofence/evaluate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ policy, ips }),
      });

      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        throw new Error(errData.error || `HTTP error ${res.status}`);
      }

      const data: GeofenceEvaluationResponse = await res.json();
      setEvaluation(data);
    } catch (err: any) {
      setError(err.message || 'Geofence evaluation request failed.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="bg-slate-900/60 rounded-2xl border border-slate-800/80 p-5 backdrop-blur-sm flex flex-col h-[580px]">
      {/* Top Header */}
      <div className="flex flex-wrap items-center justify-between gap-3 pb-4 border-b border-slate-800/80">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-lg bg-emerald-500/20 text-emerald-400 flex items-center justify-center">
            <Shield className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-slate-100">Geofencing Rule Matrix</h3>
            <p className="text-xs text-slate-400">Simulate country access policies, EU compliance, and ASN filters</p>
          </div>
        </div>

        {/* Presets */}
        <div className="flex items-center gap-2">
          <span className="text-xs text-slate-500 font-medium">Presets:</span>
          <button
            onClick={() => applyPreset('eu')}
            className="px-2.5 py-1 text-xs bg-slate-800 text-slate-300 hover:text-white rounded-lg border border-slate-700 transition-colors"
          >
            Strict EU Only
          </button>
          <button
            onClick={() => applyPreset('embargo')}
            className="px-2.5 py-1 text-xs bg-slate-800 text-slate-300 hover:text-white rounded-lg border border-slate-700 transition-colors"
          >
            Sanction Filter
          </button>
          <button
            onClick={() => applyPreset('na_uk')}
            className="px-2.5 py-1 text-xs bg-slate-800 text-slate-300 hover:text-white rounded-lg border border-slate-700 transition-colors"
          >
            US/CA/GB Only
          </button>
        </div>
      </div>

      {/* Grid: Policy Settings & Evaluator */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-4 flex-1 min-h-0">
        {/* Left Column: Policy Config */}
        <div className="flex flex-col space-y-3 overflow-y-auto pr-1">
          <div>
            <label className="text-[11px] font-semibold text-slate-400 block mb-1">Policy Name</label>
            <input
              type="text"
              value={policyName}
              onChange={e => setPolicyName(e.target.value)}
              className="w-full bg-slate-950 border border-slate-800 rounded-lg px-2.5 py-1.5 text-xs text-slate-200 outline-none focus:border-indigo-500"
            />
          </div>

          <div>
            <label className="text-[11px] font-semibold text-slate-400 block mb-1">
              Allowed Countries (comma-separated ISO)
            </label>
            <input
              type="text"
              placeholder="e.g. US, CA, GB (empty = allow all)"
              value={allowedCountries}
              onChange={e => setAllowedCountries(e.target.value)}
              className="w-full bg-slate-950 border border-slate-800 rounded-lg px-2.5 py-1.5 text-xs text-slate-200 outline-none focus:border-indigo-500 font-mono"
            />
          </div>

          <div>
            <label className="text-[11px] font-semibold text-slate-400 block mb-1">
              Blocked Countries (comma-separated ISO)
            </label>
            <input
              type="text"
              placeholder="e.g. RU, CN, IR"
              value={blockedCountries}
              onChange={e => setBlockedCountries(e.target.value)}
              className="w-full bg-slate-950 border border-slate-800 rounded-lg px-2.5 py-1.5 text-xs text-slate-200 outline-none focus:border-indigo-500 font-mono"
            />
          </div>

          <div>
            <label className="text-[11px] font-semibold text-slate-400 block mb-1">
              Blocked ASNs (comma-separated numbers)
            </label>
            <input
              type="text"
              placeholder="e.g. 12389, 4134"
              value={blockedASNs}
              onChange={e => setBlockedASNs(e.target.value)}
              className="w-full bg-slate-950 border border-slate-800 rounded-lg px-2.5 py-1.5 text-xs text-slate-200 outline-none focus:border-indigo-500 font-mono"
            />
          </div>

          {/* EU Toggle */}
          <div className="flex items-center justify-between p-2.5 rounded-xl bg-slate-950 border border-slate-800">
            <div>
              <span className="text-xs font-semibold text-slate-200 block">Require EU / EEA</span>
              <span className="text-[10px] text-slate-400">Strictly enforce European Union jurisdiction</span>
            </div>
            <button
              onClick={() => setRequireEU(!requireEU)}
              className={`w-10 h-6 rounded-full p-1 transition-colors ${
                requireEU ? 'bg-indigo-600' : 'bg-slate-800'
              }`}
            >
              <div
                className={`w-4 h-4 rounded-full bg-white transition-transform ${
                  requireEU ? 'translate-x-4' : 'translate-x-0'
                }`}
              />
            </button>
          </div>

          {/* Test Candidate IPs */}
          <div className="flex-1 flex flex-col space-y-1">
            <label className="text-[11px] font-semibold text-slate-400">Test Candidate IPs</label>
            <textarea
              value={testIPs}
              onChange={e => setTestIPs(e.target.value)}
              rows={4}
              className="w-full bg-slate-950 border border-slate-800 rounded-lg p-2 text-xs font-mono text-slate-200 outline-none resize-none focus:border-indigo-500"
            />
          </div>

          <button
            onClick={evaluateGeofence}
            disabled={isLoading}
            className="w-full py-2 bg-emerald-600 hover:bg-emerald-500 text-white font-semibold rounded-xl text-xs flex items-center justify-center gap-2 shadow-lg transition-all"
          >
            <Play className="w-3.5 h-3.5 fill-current" />
            Evaluate Geofence Rules
          </button>
          {error && <p className="text-xs text-rose-400">{error}</p>}
        </div>

        {/* Right Column: Verdict Table */}
        <div className="md:col-span-2 flex flex-col h-full space-y-3 min-h-0">
          {/* Summary Badges */}
          <div className="flex items-center gap-3">
            <div className="flex-1 bg-slate-950/70 border border-slate-800 rounded-xl p-3 flex items-center justify-between">
              <div>
                <span className="text-[11px] text-slate-400 block">Allowed Traffic</span>
                <span className="text-lg font-bold text-emerald-400 font-mono">
                  {evaluation ? evaluation.allowed_count : '—'}
                </span>
              </div>
              <CheckCircle2 className="w-6 h-6 text-emerald-500/30" />
            </div>

            <div className="flex-1 bg-slate-950/70 border border-slate-800 rounded-xl p-3 flex items-center justify-between">
              <div>
                <span className="text-[11px] text-slate-400 block">Blocked Traffic</span>
                <span className="text-lg font-bold text-rose-400 font-mono">
                  {evaluation ? evaluation.blocked_count : '—'}
                </span>
              </div>
              <XCircle className="w-6 h-6 text-rose-500/30" />
            </div>
          </div>

          {/* Verdicts Table */}
          <div className="flex-1 overflow-y-auto rounded-xl border border-slate-800/80 bg-slate-950/60 font-mono text-xs">
            {!evaluation ? (
              <div className="h-full flex flex-col items-center justify-center text-center p-6 text-slate-500">
                <Shield className="w-6 h-6 text-slate-600 mb-2" />
                <p>Run evaluation to inspect policy verdicts</p>
              </div>
            ) : (
              <table className="w-full border-collapse">
                <thead className="sticky top-0 bg-slate-900 border-b border-slate-800 text-slate-400 text-[11px]">
                  <tr>
                    <th className="text-left py-2 px-3">IP</th>
                    <th className="text-left py-2 px-3">VERDICT</th>
                    <th className="text-left py-2 px-3">LOCATION</th>
                    <th className="text-left py-2 px-3">TRIGGERED RULE & REASON</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800/60">
                  {evaluation.results.map((item, idx) => (
                    <tr key={idx} className="hover:bg-slate-800/40">
                      <td className="py-2.5 px-3 font-bold text-slate-200">{item.ip}</td>
                      <td className="py-2.5 px-3">
                        <span
                          className={`px-2 py-0.5 rounded text-[10px] font-bold border ${
                            item.allowed
                              ? 'bg-emerald-950 text-emerald-400 border-emerald-800'
                              : 'bg-rose-950 text-rose-400 border-rose-800'
                          }`}
                        >
                          {item.allowed ? 'PASS' : 'BLOCKED'}
                        </span>
                      </td>
                      <td className="py-2.5 px-3 text-slate-300">
                        {item.record ? (
                          <div className="flex items-center gap-1.5">
                            <span className="font-bold text-indigo-400">{item.record.country_code}</span>
                            <span className="text-slate-400 truncate max-w-[90px]">{item.record.city}</span>
                          </div>
                        ) : (
                          '—'
                        )}
                      </td>
                      <td className="py-2.5 px-3 text-slate-300 text-[11px]">
                        <span className="font-semibold text-slate-200 mr-1.5">[{item.matched_rule}]</span>
                        <span className="text-slate-400">{item.reason}</span>
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
