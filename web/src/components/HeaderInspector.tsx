import React, { useState } from 'react';
import { Copy, Check, Code2, Cookie, Zap } from 'lucide-react';
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

  const copyToClipboard = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  const toggleBrowserSessionSimulation = () => {
    if (!record) return;
    if (cookieActive) {
      document.cookie = 'geogate_simulate_country=; path=/; max-age=0';
      setCookieActive(false);
    } else {
      document.cookie = `geogate_simulate_country=${record.country_code}; path=/; max-age=86400`;
      setCookieActive(true);
    }
  };

  const countryCode = record?.country_code || 'US';

  const snippets = {
    curl: `# Run cURL with GeoGate Edge Header Override
curl -i -H "X-Simulate-Country: ${countryCode}" \\
  ${upstreamUrl}/api/pricing

# Or using query parameter override
curl -i "${upstreamUrl}/api/pricing?simulate_country=${countryCode}"`,

    nextjs: `// Next.js App Router (middleware.ts)
import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

export function middleware(request: NextRequest) {
  // Read edge geolocation injected transparently by GeoGate
  const country = request.headers.get('cf-ipcountry') || 'US'
  const city = request.headers.get('x-geo-city') || 'Unknown'
  const isEU = request.headers.get('x-geo-eu') === '1'
  const asn = request.headers.get('x-asn')

  if (isEU) {
    // Enforce EU GDPR or VAT compliance
  }

  return NextResponse.next()
}`,

    express: `// Express.js / Node.js
app.use((req, res, next) => {
  req.geo = {
    country: req.headers['cf-ipcountry'] || 'US',
    city: req.headers['x-geo-city'],
    latitude: parseFloat(req.headers['x-geo-latitude'] || '0'),
    longitude: parseFloat(req.headers['x-geo-longitude'] || '0'),
    isEU: req.headers['x-geo-eu'] === '1',
    asn: req.headers['x-asn'],
    currency: req.headers['x-geo-currency'] || 'USD',
  }
  next()
})`,

    fastapi: `# Python FastAPI
from fastapi import FastAPI, Header
from typing import Optional

app = FastAPI()

@app.get("/pricing")
async def get_pricing(
    country: str = Header("US", alias="cf-ipcountry"),
    city: Optional[str] = Header(None, alias="x-geo-city"),
    is_eu: str = Header("0", alias="x-geo-eu"),
):
    return {
        "country": country,
        "city": city,
        "apply_eu_vat": is_eu == "1"
    }`,
  };

  return (
    <div className="bg-slate-900/60 rounded-2xl border border-slate-800/80 p-5 backdrop-blur-sm flex flex-col h-full">
      {/* Top Header */}
      <div className="flex items-center justify-between pb-4 border-b border-slate-800/80">
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

      {/* Header Key-Value Table */}
      <div className="my-4 overflow-y-auto max-h-56 rounded-xl border border-slate-800 bg-slate-950/70 p-2 space-y-1 font-mono text-xs">
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
