import React, { useState, useEffect, useRef } from 'react';
import { Globe, MapPin, ZoomIn, ZoomOut, RotateCcw, Search, Compass, Navigation } from 'lucide-react';
import type { CityEntry, CountryInfo } from '../types';

interface WorldMapProps {
  cities: CityEntry[];
  countries: CountryInfo[];
  selectedCity: CityEntry | null;
  selectedCountry: string;
  onSelectCity: (city: CityEntry) => void;
  onSelectCountry: (countryCode: string) => void;
}

export const WorldMap: React.FC<WorldMapProps> = ({
  cities,
  countries,
  selectedCity,
  selectedCountry: _selectedCountry,
  onSelectCity,
  onSelectCountry,
}) => {
  const [viewMode, setViewMode] = useState<'3d' | '2d'>('3d');
  const [searchQuery, setSearchQuery] = useState('');
  const [rotation, setRotation] = useState({ x: 15, y: -20 });
  const [zoom, setZoom] = useState(1);
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });
  const canvasRef = useRef<HTMLCanvasElement | null>(null);

  // Filtered search results
  const filteredCities = searchQuery.trim()
    ? cities.filter(
        c =>
          c.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          c.country_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          c.country_code.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : [];

  const filteredCountries = searchQuery.trim()
    ? countries.filter(
        c =>
          c.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          c.code.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : [];

  // When selectedCity changes, smoothly rotate globe towards it
  useEffect(() => {
    if (selectedCity && viewMode === '3d') {
      // Longitude maps to rotation.y, Latitude to rotation.x
      const targetY = -selectedCity.longitude;
      const targetX = selectedCity.latitude * 0.5;
      setRotation({ x: targetX, y: targetY });
    }
  }, [selectedCity, viewMode]);

  // Handle Drag / Rotate for 3D globe
  const handleMouseDown = (e: React.MouseEvent) => {
    setIsDragging(true);
    setDragStart({ x: e.clientX, y: e.clientY });
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (!isDragging) return;
    const deltaX = e.clientX - dragStart.x;
    const deltaY = e.clientY - dragStart.y;
    setDragStart({ x: e.clientX, y: e.clientY });

    setRotation(prev => ({
      x: Math.max(-60, Math.min(60, prev.x - deltaY * 0.4)),
      y: (prev.y + deltaX * 0.4) % 360,
    }));
  };

  const handleMouseUp = () => setIsDragging(false);

  // Canvas drawing for 3D Globe / 2D Map
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    let animationFrameId: number | undefined;
    const width = canvas.width;
    const height = canvas.height;
    const centerX = width / 2;
    const centerY = height / 2;
    const radius = Math.min(width, height) * 0.38 * zoom;

    const render = () => {
      ctx.clearRect(0, 0, width, height);

      if (viewMode === '3d') {
        // Draw 3D Globe Sphere
        const grad = ctx.createRadialGradient(
          centerX - radius * 0.3,
          centerY - radius * 0.3,
          radius * 0.1,
          centerX,
          centerY,
          radius
        );
        grad.addColorStop(0, '#1e293b');
        grad.addColorStop(0.7, '#0f172a');
        grad.addColorStop(1, '#020617');

        ctx.beginPath();
        ctx.arc(centerX, centerY, radius, 0, Math.PI * 2);
        ctx.fillStyle = grad;
        ctx.fill();

        // Atmospheric glow rim
        ctx.lineWidth = 2.5;
        ctx.strokeStyle = '#38bdf8';
        ctx.shadowColor = '#38bdf8';
        ctx.shadowBlur = 18;
        ctx.stroke();
        ctx.shadowBlur = 0;

        // Draw Graticule Lines (latitudes & longitudes)
        ctx.lineWidth = 0.75;
        ctx.strokeStyle = 'rgba(148, 163, 184, 0.18)';

        // Parallels
        for (let lat = -60; lat <= 60; lat += 30) {
          ctx.beginPath();
          let first = true;
          for (let lon = 0; lon <= 360; lon += 5) {
            const radLat = (lat * Math.PI) / 180;
            const radLon = ((lon + rotation.y) * Math.PI) / 180;
            const radRotX = (rotation.x * Math.PI) / 180;

            // 3D sphere coordinate projection
            let x3 = radius * Math.cos(radLat) * Math.sin(radLon);
            let y3 = radius * Math.sin(radLat);
            let z3 = radius * Math.cos(radLat) * Math.cos(radLon);

            // Pitch rotation (x axis)
            const yRot = y3 * Math.cos(radRotX) - z3 * Math.sin(radRotX);
            const zRot = y3 * Math.sin(radRotX) + z3 * Math.cos(radRotX);

            if (zRot > 0) {
              const screenX = centerX + x3;
              const screenY = centerY - yRot;
              if (first) {
                ctx.moveTo(screenX, screenY);
                first = false;
              } else {
                ctx.lineTo(screenX, screenY);
              }
            } else {
              first = true;
            }
          }
          ctx.stroke();
        }

        // Meridians
        for (let lon = 0; lon < 360; lon += 30) {
          ctx.beginPath();
          let first = true;
          for (let lat = -90; lat <= 90; lat += 5) {
            const radLat = (lat * Math.PI) / 180;
            const radLon = ((lon + rotation.y) * Math.PI) / 180;
            const radRotX = (rotation.x * Math.PI) / 180;

            let x3 = radius * Math.cos(radLat) * Math.sin(radLon);
            let y3 = radius * Math.sin(radLat);
            let z3 = radius * Math.cos(radLat) * Math.cos(radLon);

            const yRot = y3 * Math.cos(radRotX) - z3 * Math.sin(radRotX);
            const zRot = y3 * Math.sin(radRotX) + z3 * Math.cos(radRotX);

            if (zRot > 0) {
              const screenX = centerX + x3;
              const screenY = centerY - yRot;
              if (first) {
                ctx.moveTo(screenX, screenY);
                first = false;
              } else {
                ctx.lineTo(screenX, screenY);
              }
            } else {
              first = true;
            }
          }
          ctx.stroke();
        }

        // Plot Cities on 3D Globe
        cities.forEach(city => {
          const radLat = (city.latitude * Math.PI) / 180;
          const radLon = ((city.longitude + rotation.y) * Math.PI) / 180;
          const radRotX = (rotation.x * Math.PI) / 180;

          let x3 = radius * Math.cos(radLat) * Math.sin(radLon);
          let y3 = radius * Math.sin(radLat);
          let z3 = radius * Math.cos(radLat) * Math.cos(radLon);

          const yRot = y3 * Math.cos(radRotX) - z3 * Math.sin(radRotX);
          const zRot = y3 * Math.sin(radRotX) + z3 * Math.cos(radRotX);

          // Only draw if city is on the visible front hemisphere (zRot > 0)
          if (zRot > 0) {
            const screenX = centerX + x3;
            const screenY = centerY - yRot;
            const isSelected = selectedCity?.name === city.name;

            // Ping circle
            ctx.beginPath();
            ctx.arc(screenX, screenY, isSelected ? 8 : 4.5, 0, Math.PI * 2);
            ctx.fillStyle = isSelected ? '#10b981' : '#38bdf8';
            ctx.shadowColor = isSelected ? '#10b981' : '#0284c7';
            ctx.shadowBlur = isSelected ? 14 : 6;
            ctx.fill();
            ctx.shadowBlur = 0;

            if (isSelected) {
              ctx.lineWidth = 2;
              ctx.strokeStyle = '#ffffff';
              ctx.stroke();

              // City label tag
              ctx.font = '600 12px sans-serif';
              ctx.fillStyle = '#ffffff';
              ctx.fillText(`${city.name} (${city.country_code})`, screenX + 10, screenY - 4);
            }
          }
        });
      } else {
        // Draw 2D Equirectangular Flat Map
        ctx.fillStyle = '#090d16';
        ctx.fillRect(40, 40, width - 80, height - 80);
        ctx.strokeStyle = '#1e293b';
        ctx.strokeRect(40, 40, width - 80, height - 80);

        // Grid
        ctx.lineWidth = 0.5;
        ctx.strokeStyle = 'rgba(148, 163, 184, 0.15)';
        for (let x = 40; x <= width - 40; x += (width - 80) / 12) {
          ctx.beginPath();
          ctx.moveTo(x, 40);
          ctx.lineTo(x, height - 40);
          ctx.stroke();
        }
        for (let y = 40; y <= height - 40; y += (height - 80) / 6) {
          ctx.beginPath();
          ctx.moveTo(40, y);
          ctx.lineTo(width - 40, y);
          ctx.stroke();
        }

        // Plot Cities on 2D map
        cities.forEach(city => {
          // Equirectangular mapping: Lon [-180, 180] -> [40, width - 40]
          // Lat [90, -90] -> [40, height - 40]
          const screenX = 40 + ((city.longitude + 180) / 360) * (width - 80);
          const screenY = 40 + ((90 - city.latitude) / 180) * (height - 80);
          const isSelected = selectedCity?.name === city.name;

          ctx.beginPath();
          ctx.arc(screenX, screenY, isSelected ? 7 : 3.5, 0, Math.PI * 2);
          ctx.fillStyle = isSelected ? '#10b981' : '#38bdf8';
          ctx.fill();

          if (isSelected) {
            ctx.font = '600 11px sans-serif';
            ctx.fillStyle = '#ffffff';
            ctx.fillText(`${city.name}`, screenX + 8, screenY - 2);
          }
        });
      }
    };

    render();
    return () => {
      if (animationFrameId) cancelAnimationFrame(animationFrameId);
    };
  }, [viewMode, rotation, zoom, cities, selectedCity]);

  // Click on canvas to find nearest city
  const handleCanvasClick = (e: React.MouseEvent<HTMLCanvasElement>) => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const rect = canvas.getBoundingClientRect();
    const clickX = e.clientX - rect.left;
    const clickY = e.clientY - rect.top;

    const width = canvas.width;
    const height = canvas.height;
    const centerX = width / 2;
    const centerY = height / 2;
    const radius = Math.min(width, height) * 0.38 * zoom;

    let closestCity: CityEntry | null = null;
    let minDistance = 25; // hit radius in pixels

    if (viewMode === '3d') {
      cities.forEach(city => {
        const radLat = (city.latitude * Math.PI) / 180;
        const radLon = ((city.longitude + rotation.y) * Math.PI) / 180;
        const radRotX = (rotation.x * Math.PI) / 180;

        let x3 = radius * Math.cos(radLat) * Math.sin(radLon);
        let y3 = radius * Math.sin(radLat);
        let z3 = radius * Math.cos(radLat) * Math.cos(radLon);

        const yRot = y3 * Math.cos(radRotX) - z3 * Math.sin(radRotX);
        const zRot = y3 * Math.sin(radRotX) + z3 * Math.cos(radRotX);

        if (zRot > 0) {
          const screenX = centerX + x3;
          const screenY = centerY - yRot;
          const dist = Math.hypot(screenX - clickX, screenY - clickY);
          if (dist < minDistance) {
            minDistance = dist;
            closestCity = city;
          }
        }
      });
    } else {
      cities.forEach(city => {
        const screenX = 40 + ((city.longitude + 180) / 360) * (width - 80);
        const screenY = 40 + ((90 - city.latitude) / 180) * (height - 80);
        const dist = Math.hypot(screenX - clickX, screenY - clickY);
        if (dist < minDistance) {
          minDistance = dist;
          closestCity = city;
        }
      });
    }

    if (closestCity) {
      onSelectCity(closestCity);
    }
  };

  return (
    <div className="relative w-full h-[580px] bg-slate-900/60 rounded-2xl border border-slate-800/80 overflow-hidden flex flex-col backdrop-blur-sm">
      {/* Top Map Controls Bar */}
      <div className="absolute top-4 left-4 right-4 z-20 flex flex-wrap items-center justify-between gap-3 pointer-events-none">
        {/* Search Bar */}
        <div className="relative w-72 pointer-events-auto">
          <div className="flex items-center bg-slate-950/90 border border-slate-700/80 rounded-xl px-3 py-2 shadow-xl backdrop-blur-md">
            <Search className="w-4 h-4 text-slate-400 mr-2 flex-shrink-0" />
            <input
              type="text"
              placeholder="Search city, country or code..."
              value={searchQuery}
              onChange={e => setSearchQuery(e.target.value)}
              className="w-full bg-transparent text-sm text-slate-100 placeholder-slate-400 outline-none"
            />
          </div>

          {/* Search Dropdown Results */}
          {searchQuery.trim() && (
            <div className="absolute top-full left-0 right-0 mt-2 bg-slate-950/95 border border-slate-800 rounded-xl shadow-2xl max-h-60 overflow-y-auto z-30 p-1">
              {filteredCities.slice(0, 6).map(city => (
                <button
                  key={city.name}
                  onClick={() => {
                    onSelectCity(city);
                    setSearchQuery('');
                  }}
                  className="w-full text-left px-3 py-2 text-xs hover:bg-indigo-600/20 hover:text-indigo-300 rounded-lg flex items-center justify-between transition-colors"
                >
                  <span className="font-semibold text-slate-200">{city.name}</span>
                  <span className="text-slate-400 font-mono text-[10px]">
                    {city.country_name} ({city.country_code})
                  </span>
                </button>
              ))}
              {filteredCountries.slice(0, 4).map(country => (
                <button
                  key={country.code}
                  onClick={() => {
                    onSelectCountry(country.code);
                    setSearchQuery('');
                  }}
                  className="w-full text-left px-3 py-2 text-xs hover:bg-emerald-600/20 hover:text-emerald-300 rounded-lg flex items-center justify-between transition-colors"
                >
                  <span className="font-semibold text-slate-200">{country.name}</span>
                  <span className="text-slate-400 font-mono text-[10px]">
                    Capital: {country.capital} [{country.code}]
                  </span>
                </button>
              ))}
            </div>
          )}
        </div>

        {/* View Mode & Camera Action Buttons */}
        <div className="flex items-center gap-2 pointer-events-auto">
          <div className="flex bg-slate-950/80 p-1 border border-slate-800 rounded-xl">
            <button
              onClick={() => setViewMode('3d')}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium flex items-center gap-1.5 transition-all ${
                viewMode === '3d'
                  ? 'bg-indigo-600 text-white shadow-md'
                  : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              <Globe className="w-3.5 h-3.5" />
              3D Globe
            </button>
            <button
              onClick={() => setViewMode('2d')}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium flex items-center gap-1.5 transition-all ${
                viewMode === '2d'
                  ? 'bg-indigo-600 text-white shadow-md'
                  : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              <Navigation className="w-3.5 h-3.5" />
              2D Flat
            </button>
          </div>

          <div className="flex bg-slate-950/80 p-1 border border-slate-800 rounded-xl">
            <button
              onClick={() => setZoom(prev => Math.min(2.0, prev + 0.15))}
              title="Zoom In"
              className="p-1.5 text-slate-400 hover:text-white rounded-lg hover:bg-slate-800"
            >
              <ZoomIn className="w-4 h-4" />
            </button>
            <button
              onClick={() => setZoom(prev => Math.max(0.6, prev - 0.15))}
              title="Zoom Out"
              className="p-1.5 text-slate-400 hover:text-white rounded-lg hover:bg-slate-800"
            >
              <ZoomOut className="w-4 h-4" />
            </button>
            <button
              onClick={() => {
                setRotation({ x: 15, y: -20 });
                setZoom(1);
              }}
              title="Reset View"
              className="p-1.5 text-slate-400 hover:text-white rounded-lg hover:bg-slate-800"
            >
              <RotateCcw className="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>

      {/* Interactive Canvas Rendering Surface */}
      <canvas
        ref={canvasRef}
        width={960}
        height={580}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseUp}
        onClick={handleCanvasClick}
        className={`w-full h-full cursor-grab active:cursor-grabbing block select-none ${
          viewMode === '3d' ? '' : 'cursor-crosshair'
        }`}
      />

      {/* Floating Info Overlay for Currently Selected City */}
      {selectedCity && (
        <div className="absolute bottom-4 left-4 z-20 bg-slate-950/90 border border-indigo-500/40 rounded-xl p-3 shadow-2xl backdrop-blur-md flex items-center gap-3 max-w-md">
          <div className="w-9 h-9 rounded-lg bg-indigo-600/20 border border-indigo-500/30 flex items-center justify-center flex-shrink-0 text-indigo-400">
            <MapPin className="w-5 h-5" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-bold text-sm text-white">{selectedCity.name}</span>
              <span className="text-xs px-1.5 py-0.5 rounded bg-indigo-950 border border-indigo-800 text-indigo-300 font-mono">
                {selectedCity.country_code}
              </span>
              {selectedCity.is_eu && (
                <span className="text-[10px] px-1 py-0.5 rounded bg-emerald-950 border border-emerald-800 text-emerald-400 font-semibold">
                  EU Member
                </span>
              )}
              {selectedCity.ppp_discount !== undefined && selectedCity.ppp_discount > 0 && (
                <span className="text-[10px] px-1.5 py-0.5 rounded bg-amber-950 border border-amber-800 text-amber-300 font-semibold">
                  -{selectedCity.ppp_discount}% PPP
                </span>
              )}
            </div>
            <div className="text-xs text-slate-400 mt-0.5 flex flex-wrap gap-x-3">
              <span>Timezone: {selectedCity.timezone}</span>
              <span>Currency: {selectedCity.currency}</span>
              {selectedCity.ppp_factor !== undefined && (
                <span>PPP: {selectedCity.ppp_factor}x</span>
              )}
              <span className="font-mono text-slate-300">AS{selectedCity.asn}</span>
            </div>
          </div>
        </div>
      )}

      {/* Drag Instruction Cue */}
      <div className="absolute bottom-4 right-4 z-10 text-[11px] text-slate-400/80 flex items-center gap-1.5 bg-slate-950/60 px-3 py-1.5 rounded-lg border border-slate-800/60 pointer-events-none">
        <Compass className="w-3.5 h-3.5 text-indigo-400" />
        <span>Click any pin or drag globe to rotate</span>
      </div>
    </div>
  );
};
