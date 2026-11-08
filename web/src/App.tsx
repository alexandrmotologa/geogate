import React, { useState, useEffect } from 'react';
import {
  Globe2,
  Activity,
  Gauge,
  Shield,
  Server,
  Cpu,
} from 'lucide-react';
import { WorldMap } from './components/WorldMap';
import { HeaderInspector } from './components/HeaderInspector';
import { ProxyStream } from './components/ProxyStream';
import { BatchTester } from './components/BatchTester';
import { GeofenceMatrix } from './components/GeofenceMatrix';
import type { CityEntry, CountryInfo, EngineStatus, GeoRecord } from './types';

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'map' | 'stream' | 'batch' | 'geofence'>('map');
  const [cities, setCities] = useState<CityEntry[]>([]);
  const [countries, setCountries] = useState<CountryInfo[]>([]);
  const [status, setStatus] = useState<EngineStatus | null>(null);

  const [selectedCity, setSelectedCity] = useState<CityEntry | null>(null);
  const [selectedCountry, setSelectedCountry] = useState<string>('JP');
  const [activeRecord, setActiveRecord] = useState<GeoRecord | null>(null);
  const [activeHeaders, setActiveHeaders] = useState<Record<string, string>>({});

  // Fetch initial metadata and engine status
  useEffect(() => {
    fetch('/v1/cities')
      .then(res => res.json())
      .then(data => {
        if (data.cities) {
          setCities(data.cities);
          // Default to Tokyo or first city
          const tokyo = data.cities.find((c: CityEntry) => c.name === 'Tokyo') || data.cities[0];
          if (tokyo) {
            setSelectedCity(tokyo);
            setSelectedCountry(tokyo.country_code);
          }
        }
      })
      .catch(err => console.error('Error fetching cities:', err));

    fetch('/v1/countries')
      .then(res => res.json())
      .then(data => {
        if (data.countries) setCountries(data.countries);
      })
      .catch(err => console.error('Error fetching countries:', err));

    const fetchStatus = () => {
      fetch('/v1/status')
        .then(res => res.json())
        .then(data => setStatus(data))
        .catch(err => console.error('Error fetching status:', err));
    };

    fetchStatus();
    const interval = setInterval(fetchStatus, 4000);
    return () => clearInterval(interval);
  }, []);

  // Whenever selectedCity or selectedCountry changes, fetch simulated headers
  useEffect(() => {
    let url = '/v1/headers/simulate';
    if (selectedCity) {
      url += `?city=${encodeURIComponent(selectedCity.name)}`;
    } else if (selectedCountry) {
      url += `?country=${encodeURIComponent(selectedCountry)}`;
    }

    fetch(url)
      .then(res => res.json())
      .then(data => {
        if (data.record) setActiveRecord(data.record);
        if (data.headers) setActiveHeaders(data.headers);
      })
      .catch(err => console.error('Error simulating headers:', err));
  }, [selectedCity, selectedCountry]);

  const handleSelectCity = (city: CityEntry) => {
    setSelectedCity(city);
    setSelectedCountry(city.country_code);
  };

  const handleSelectCountry = (code: string) => {
    setSelectedCountry(code);
    const matchCity = cities.find(c => c.country_code === code);
    if (matchCity) {
      setSelectedCity(matchCity);
    } else {
      setSelectedCity(null);
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col selection:bg-indigo-600 selection:text-white">
      {/* Top Navbar */}
      <header className="border-b border-slate-800/80 bg-slate-950/80 backdrop-blur-md sticky top-0 z-40 px-6 py-3.5">
        <div className="max-w-7xl mx-auto flex flex-wrap items-center justify-between gap-4">
          {/* Logo & Brand Title */}
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-indigo-600 to-cyan-500 flex items-center justify-center text-white shadow-lg shadow-indigo-500/20">
              <Globe2 className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-extrabold text-base tracking-tight text-white">GeoGate</span>
                <span className="text-[10px] font-bold px-2 py-0.5 rounded-full bg-indigo-950/80 text-indigo-300 border border-indigo-800 font-mono">
                  v1.0.0
                </span>
                <span className="text-[10px] font-bold px-2 py-0.5 rounded-full bg-emerald-950/80 text-emerald-400 border border-emerald-800 flex items-center gap-1 font-mono">
                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
                  &lt; 35MB RAM
                </span>
              </div>
              <p className="text-xs text-slate-400">Self-Hosted Local IP Geolocation & Edge Geofencing Gateway</p>
            </div>
          </div>

          {/* Runtime Metrics Telemetry */}
          {status && (
            <div className="flex items-center gap-3 text-xs font-mono">
              <div className="hidden sm:flex items-center gap-2 bg-slate-900/90 border border-slate-800 rounded-lg px-3 py-1.5">
                <Cpu className="w-3.5 h-3.5 text-indigo-400" />
                <span className="text-slate-400">RAM:</span>
                <span className="text-emerald-400 font-bold">{status.memory_alloc_mb.toFixed(1)} MB</span>
              </div>
              <div className="hidden sm:flex items-center gap-2 bg-slate-900/90 border border-slate-800 rounded-lg px-3 py-1.5">
                <Activity className="w-3.5 h-3.5 text-cyan-400" />
                <span className="text-slate-400">Lookups:</span>
                <span className="text-slate-200 font-bold">{status.total_lookups.toLocaleString()}</span>
              </div>
              <div className="flex items-center gap-2 bg-slate-900/90 border border-slate-800 rounded-lg px-3 py-1.5">
                <Server className="w-3.5 h-3.5 text-amber-400" />
                <span className="text-slate-400">Upstream:</span>
                <span className="text-indigo-300 font-bold truncate max-w-[140px]">{status.upstream_url}</span>
              </div>
            </div>
          )}
        </div>
      </header>

      {/* Main Container */}
      <main className="max-w-7xl mx-auto w-full px-6 py-6 flex-1 flex flex-col space-y-5">
        {/* Navigation Tabs Bar */}
        <div className="flex flex-wrap items-center justify-between border-b border-slate-800/80 pb-3 gap-3">
          <div className="flex items-center gap-2 overflow-x-auto py-1 max-w-full">
            <button
              onClick={() => setActiveTab('map')}
              className={`px-3 sm:px-4 py-2 rounded-xl text-xs font-semibold flex items-center gap-2 transition-all whitespace-nowrap flex-shrink-0 ${
                activeTab === 'map'
                  ? 'bg-indigo-600 text-white shadow-lg shadow-indigo-600/20'
                  : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900'
              }`}
            >
              <Globe2 className="w-4 h-4" />
              World Map Studio
            </button>
            <button
              onClick={() => setActiveTab('stream')}
              className={`px-3 sm:px-4 py-2 rounded-xl text-xs font-semibold flex items-center gap-2 transition-all whitespace-nowrap flex-shrink-0 ${
                activeTab === 'stream'
                  ? 'bg-indigo-600 text-white shadow-lg shadow-indigo-600/20'
                  : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900'
              }`}
            >
              <Activity className="w-4 h-4" />
              Live Proxy Stream
            </button>
            <button
              onClick={() => setActiveTab('batch')}
              className={`px-3 sm:px-4 py-2 rounded-xl text-xs font-semibold flex items-center gap-2 transition-all whitespace-nowrap flex-shrink-0 ${
                activeTab === 'batch'
                  ? 'bg-indigo-600 text-white shadow-lg shadow-indigo-600/20'
                  : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900'
              }`}
            >
              <Gauge className="w-4 h-4" />
              Batch Pipeline
            </button>
            <button
              onClick={() => setActiveTab('geofence')}
              className={`px-3 sm:px-4 py-2 rounded-xl text-xs font-semibold flex items-center gap-2 transition-all whitespace-nowrap flex-shrink-0 ${
                activeTab === 'geofence'
                  ? 'bg-indigo-600 text-white shadow-lg shadow-indigo-600/20'
                  : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900'
              }`}
            >
              <Shield className="w-4 h-4" />
              Geofence Matrix
            </button>
          </div>

          <div className="hidden lg:flex items-center gap-2 text-xs text-slate-400 font-mono">
            <span>Query override:</span>
            <code className="text-indigo-400 bg-slate-900 px-2 py-0.5 rounded border border-slate-800">
              ?simulate_country={selectedCountry}
            </code>
          </div>
        </div>

        {/* Tab Views */}
        {activeTab === 'map' && (
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-5 flex-1">
            {/* World Map: 7 cols on large screens */}
            <div className="lg:col-span-7 flex flex-col">
              <WorldMap
                cities={cities}
                countries={countries}
                selectedCity={selectedCity}
                selectedCountry={selectedCountry}
                onSelectCity={handleSelectCity}
                onSelectCountry={handleSelectCountry}
              />
            </div>

            {/* Edge Header Synthesizer: 5 cols */}
            <div className="lg:col-span-5 flex flex-col">
              <HeaderInspector
                record={activeRecord}
                headers={activeHeaders}
                upstreamUrl={status?.upstream_url || 'http://localhost:3000'}
              />
            </div>
          </div>
        )}

        {activeTab === 'stream' && (
          <ProxyStream upstreamUrl={status?.upstream_url || 'http://localhost:3000'} />
        )}

        {activeTab === 'batch' && <BatchTester />}

        {activeTab === 'geofence' && <GeofenceMatrix />}
      </main>

      {/* Footer */}
      <footer className="border-t border-slate-800/60 py-4 px-6 text-center text-xs text-slate-500 font-mono">
        GeoGate Daemon • MaxMind MMDB In-Memory Radix Engine • Latency &lt; 8μs • Single Native Binary
      </footer>
    </div>
  );
};

export default App;
