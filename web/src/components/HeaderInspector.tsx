import React, { useState } from 'react';
import { Copy, Check, Code2, Cookie, Zap, Shield, Clock, Percent } from 'lucide-react';
import type { GeoRecord } from '../types';

interface HeaderInspectorProps {
  record: GeoRecord | null;
  headers: Record<string, string>;
  upstreamUrl?: string;
}

export const HeaderInspector: React.FC<HeaderInspectorProps> = ({ record, headers, upstreamUrl = 'http://localhost:8080' }) => {
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [activeSnippetTab, setActiveSnippetTab] = useState<'curl' | 'nextjs' | 'express' | 'fastapi'>('curl');
  const [cookieActive, setCookieActive] = useState(false);
  const [selectedConn, setSelectedConn] = useState<'residential' | 'datacenter' | 'vpn' | 'tor'>('residential');
  const [selectedLatency, setSelectedLatency] = useState<'none' | 'auto' | '100ms' | '250ms'>('none');

  const copyToClipboard = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  const toggleBrowserSessionSimulation = () => {
    if (!record) return;
    if (cookieActive) {
      document.cookie = 'geogate_simulate_country=; path=/; max-age=0';
      document.cookie = 'geogate_simulate_conn=; path=/; max-age=0';
      document.cookie = 'geogate_simulate_latency=; path=/; max-age=0';
      setCookieActive(false);
    } else {
      document.cookie = `geogate_simulate_country=${record.country_code}; path=/; max-age=86400`;
      if (selectedConn !== 'residential') {
        document.cookie = `geogate_simulate_conn=${selectedConn}; path=/; max-age=86400`;
      }
      if (selectedLatency !== 'none') {
        document.cookie = `geogate_simulate_latency=${selectedLatency}; path=/; max-age=86400`;
      }
      setCookieActive(true);
    }
  };

  const updateConnType = (conn: 'residential' | 'datacenter' | 'vpn' | 'tor') => {
    setSelectedConn(conn);
    if (cookieActive) {
      if (conn === 'residential') {
        document.cookie = 'geogate_simulate_conn=; path=/; max-age=0';
      } else {
        document.cookie = `geogate_simulate_conn=${conn}; path=/; max-age=86400`;
      }
    }
  };

  const updateLatency = (lat: 'none' | 'auto' | '100ms' | '250ms') => {
    setSelectedLatency(lat);
    if (cookieActive) {
      if (lat === 'none') {
        document.cookie = 'geogate_simulate_latency=; path=/; max-age=0';
      } else {
        document.cookie = `geogate_simulate_latency=${lat}; path=/; max-age=86400`;
      }
    }
  };

  const countryCode = record?.country_code || 'US';
  const pppDiscount = record?.ppp_discount || 0;
  const pppFactor = record?.ppp_factor || 1.0;

  const buildQueryParamSuffix = () => {
    const params = new URLSearchParams();
    params.set('simulate_country', countryCode);
    if (selectedConn !== 'residential') params.set('simulate_conn', selectedConn);
    if (selectedLatency !== 'none') params.set('simulate_latency', selectedLatency);
    return params.toString();
  };

  const snippets = {
    curl: `# Run cURL with GeoGate Edge Header Override & Intelligence Persona
curl -i -H "X-Simulate-Country: ${countryCode}" \\${selectedConn !== 'residential' ? `\n  -H "X-Simulate-Conn: ${selectedConn}" \\` : ''}${selectedLatency !== 'none' ? `\n  -H "X-Simulate-Latency: ${selectedLatency}" \\` : ''}
  ${upstreamUrl}/api/pricing

# Or using query parameter override
curl -i "${upstreamUrl}/api/pricing?${buildQueryParamSuffix()}"`,

    nextjs: `// Next.js App Router (middleware.ts)
import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

export function middleware(request: NextRequest) {
  // Read edge geolocation & security intelligence injected transparently by GeoGate
  const country = request.headers.get('cf-ipcountry') || 'US'
  const isEU = request.headers.get('x-geo-eu') === '1'
  const pppDiscount = parseInt(request.headers.get('x-geo-ppp-discount') || '0', 10)
  const isTor = request.headers.get('x-geo-is-tor') === '1'
  const isDatacenter = request.headers.get('x-geo-is-datacenter') === '1'

  // Block high-risk anonymous traffic
  if (isTor) {
    return new NextResponse('Anonymous Tor proxies disallowed', { status: 403 })
  }

  // Forward calculated PPP discount to application context
  const response = NextResponse.next()
  response.headers.set('x-app-applied-discount', pppDiscount.toString())
  return response
}`,

    express: `// Express.js / Node.js
app.use((req, res, next) => {
  req.geo = {
    country: req.headers['cf-ipcountry'] || 'US',
    city: req.headers['x-geo-city'],
    isEU: req.headers['x-geo-eu'] === '1',
    pppFactor: parseFloat(req.headers['x-geo-ppp-factor'] || '1.0'),
    pppDiscount: parseInt(req.headers['x-geo-ppp-discount'] || '0', 10),
    connType: req.headers['x-geo-connection-type'] || 'residential',
    isDatacenter: req.headers['x-geo-is-datacenter'] === '1',
    isTor: req.headers['x-geo-is-tor'] === '1',
  }
  next()
})`,

    fastapi: `# Python FastAPI
from fastapi import FastAPI, Header, HTTPException
from typing import Optional

app = FastAPI()

@app.get("/pricing")
async def get_pricing(
    country: str = Header("US", alias="cf-ipcountry"),
    ppp_discount: int = Header(0, alias="x-geo-ppp-discount"),
    is_tor: str = Header("0", alias="x-geo-is-tor"),
):
    if is_tor == "1":
        raise HTTPException(status_code=403, detail="Tor traffic blocked")
    base_price = 100.0
    discounted_price = base_price * (1 - ppp_discount / 100.0)
    return {
        "country": country,
        "base_price": base_price,
        "ppp_discount_percent": ppp_discount,
        "final_price": round(discounted_price, 2)
    }`,
  };

  return (
    <div className="bg-slate-900/60 rounded-2xl border border-slate-800/80 p-5 backdrop-blur-sm flex flex-col h-full space-y-4">
      {/* Top Header */}
      <div className="flex flex-wrap items-center justify-between gap-3 pb-3 border-b border-slate-800/80">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-lg bg-indigo-500/20 text-indigo-400 flex items-center justify-center">
            <Zap className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-slate-100">Edge Header Synthesizer</h3>
            <p className="text-xs text-slate-400">Standard Cloudflare & CDN headers injected into upstream</p>
          </div>
        </div>

        {/* Session Cookie Toggle */}
        <button
          onClick={toggleBrowserSessionSimulation}
          className={`px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-all shadow-sm ${
            cookieActive
              ? 'bg-emerald-600 text-white ring-2 ring-emerald-400/50'
              : 'bg-slate-800 text-slate-300 hover:bg-slate-700 hover:text-white'
          }`}
          title="Sets geogate_simulate_country cookie for seamless browser testing"
        >
          <Cookie className="w-3.5 h-3.5" />
          {cookieActive ? `Simulating ${countryCode} in Browser` : 'Simulate in Browser Session'}
        </button>
      </div>

      {/* Simulation Persona Controls: Chaos Latency & Security Connection Type */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3 p-3 rounded-xl bg-slate-950/70 border border-slate-800">
        {/* Connection Persona Selector */}
        <div>
          <label className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider flex items-center gap-1.5 mb-1.5">
            <Shield className="w-3.5 h-3.5 text-indigo-400" />
            Connection Persona
          </label>
          <div className="grid grid-cols-4 gap-1">
            {(['residential', 'datacenter', 'vpn', 'tor'] as const).map(type => (
              <button
                key={type}
                onClick={() => updateConnType(type)}
                className={`px-2 py-1 text-[11px] font-medium rounded-lg border transition-colors capitalize ${
                  selectedConn === type
                    ? 'bg-indigo-600 border-indigo-500 text-white font-semibold'
                    : 'bg-slate-900 border-slate-800 text-slate-400 hover:bg-slate-800 hover:text-slate-200'
                }`}
              >
                {type}
              </button>
            ))}
          </div>
        </div>

        {/* Chaos Latency Selector */}
        <div>
          <label className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider flex items-center gap-1.5 mb-1.5">
            <Clock className="w-3.5 h-3.5 text-amber-400" />
            Chaos Latency Injection
          </label>
          <div className="grid grid-cols-4 gap-1">
            {(['none', 'auto', '100ms', '250ms'] as const).map(lat => (
              <button
                key={lat}
                onClick={() => updateLatency(lat)}
                className={`px-2 py-1 text-[11px] font-medium rounded-lg border transition-colors ${
                  selectedLatency === lat
                    ? 'bg-amber-600 border-amber-500 text-white font-semibold'
                    : 'bg-slate-900 border-slate-800 text-slate-400 hover:bg-slate-800 hover:text-slate-200'
                }`}
              >
                {lat === 'none' ? '0ms' : lat === 'auto' ? 'Auto Dist' : lat}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Purchasing Power Parity (PPP) Banner */}
      {pppDiscount > 0 && (
        <div className="flex items-center justify-between px-3.5 py-2.5 rounded-xl bg-gradient-to-r from-emerald-950/60 to-slate-900 border border-emerald-800/60 text-xs">
          <div className="flex items-center gap-2">
            <div className="w-6 h-6 rounded-md bg-emerald-500/20 text-emerald-400 flex items-center justify-center">
              <Percent className="w-3.5 h-3.5" />
            </div>
            <div>
              <span className="font-bold text-emerald-400">-{pppDiscount}% PPP Discount Recommended</span>
              <span className="text-slate-400 ml-2">Factor: {pppFactor}x for {countryCode}</span>
            </div>
          </div>
          <span className="text-[11px] text-slate-300 font-mono">
            $100 &rarr; <strong className="text-emerald-300">${(100 * (1 - pppDiscount / 100)).toFixed(2)}</strong>
          </span>
        </div>
      )}

      {/* Header Key-Value Table */}
      <div className="overflow-y-auto max-h-48 rounded-xl border border-slate-800 bg-slate-950/70 p-2 space-y-1 font-mono text-xs">
        {Object.entries(headers).map(([key, val]) => (
          <div
            key={key}
            className="flex items-center justify-between p-2 rounded-lg hover:bg-slate-800/40 transition-colors group"
          >
            <div className="flex items-center gap-2">
              <span className="text-indigo-400 font-semibold">{key}:</span>
              <span className="text-emerald-400 font-medium">{val}</span>
            </div>
            <button
              onClick={() => copyToClipboard(val, key)}
              className="opacity-0 group-hover:opacity-100 p-1 text-slate-400 hover:text-white rounded transition-opacity"
              title="Copy header value"
            >
              {copiedKey === key ? (
                <Check className="w-3.5 h-3.5 text-emerald-400" />
              ) : (
                <Copy className="w-3.5 h-3.5" />
              )}
            </button>
          </div>
        ))}
      </div>

      {/* Code Snippets Section */}
      <div className="mt-auto pt-3 border-t border-slate-800/80">
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center gap-1.5">
            <Code2 className="w-4 h-4 text-slate-400" />
            <span className="text-xs font-semibold text-slate-300">Integration Recipes</span>
          </div>

          {/* Snippet Tabs */}
          <div className="flex bg-slate-950 p-0.5 rounded-lg border border-slate-800 text-[11px]">
            {(['curl', 'nextjs', 'express', 'fastapi'] as const).map(tab => (
              <button
                key={tab}
                onClick={() => setActiveSnippetTab(tab)}
                className={`px-2.5 py-1 rounded-md font-mono transition-colors ${
                  activeSnippetTab === tab
                    ? 'bg-indigo-600 text-white font-semibold'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                {tab.toUpperCase()}
              </button>
            ))}
          </div>
        </div>

        {/* Code Snippet Box */}
        <div className="relative rounded-xl bg-slate-950 border border-slate-800 p-3 font-mono text-xs text-slate-300 overflow-x-auto">
          <button
            onClick={() => copyToClipboard(snippets[activeSnippetTab], 'snippet')}
            className="absolute top-2.5 right-2.5 p-1.5 bg-slate-900 border border-slate-700/60 rounded-md text-slate-400 hover:text-white transition-colors"
            title="Copy Code"
          >
            {copiedKey === 'snippet' ? (
              <Check className="w-3.5 h-3.5 text-emerald-400" />
            ) : (
              <Copy className="w-3.5 h-3.5" />
            )}
          </button>
          <pre className="text-[11px] leading-relaxed select-all">
            {snippets[activeSnippetTab]}
          </pre>
        </div>
      </div>
    </div>
  );
};
