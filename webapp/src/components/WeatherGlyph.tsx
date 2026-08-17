type WeatherGlyphProps = {
  icon?: string;
  className?: string;
};

export function WeatherGlyph({ icon, className = '' }: WeatherGlyphProps) {
  const kind = iconKind(icon);
  return (
    <span className={`weather-glyph weather-glyph-${kind} ${className}`} aria-hidden="true">
      <svg viewBox="0 0 64 64" focusable="false">
        {kind === 'moon' && <Moon />}
        {kind === 'cloud' && <CloudOnly />}
        {kind === 'rain' && <Rain />}
        {kind === 'sun' && <Sun />}
        {kind === 'suncloud' && <SunCloud />}
      </svg>
    </span>
  );
}

function iconKind(icon?: string) {
  if (!icon) return 'suncloud';
  if (icon.includes('🌙')) return 'moon';
  if (icon.includes('🌧') || icon.includes('☔')) return 'rain';
  if (icon.includes('☁')) return 'cloud';
  if (icon.includes('☀')) return 'sun';
  return 'suncloud';
}

function Sun() {
  return (
    <>
      <circle className="wg-sun" cx="32" cy="32" r="13" />
      {Array.from({ length: 8 }).map((_, i) => {
        const angle = (i * Math.PI) / 4;
        const x1 = 32 + Math.cos(angle) * 21;
        const y1 = 32 + Math.sin(angle) * 21;
        const x2 = 32 + Math.cos(angle) * 27;
        const y2 = 32 + Math.sin(angle) * 27;
        return <path key={i} className="wg-ray" d={`M${x1.toFixed(1)} ${y1.toFixed(1)} L${x2.toFixed(1)} ${y2.toFixed(1)}`} />;
      })}
    </>
  );
}

function Moon() {
  return <path className="wg-moon" d="M43.8 42.7A19.5 19.5 0 0 1 24.2 13a20.4 20.4 0 1 0 19.6 29.7Z" />;
}

function CloudOnly() {
  return <Cloud />;
}

function Rain() {
  return (
    <>
      <Cloud />
      <path className="wg-rain" d="M24 46l-3 6M34 46l-3 6M44 46l-3 6" />
    </>
  );
}

function SunCloud() {
  return (
    <>
      <g transform="translate(-8 -8) scale(.82)">
        <Sun />
      </g>
      <Cloud />
    </>
  );
}

function Cloud() {
  return <path className="wg-cloud" d="M20.2 45.2h25.4a10 10 0 0 0 0-20 13.2 13.2 0 0 0-25.4 3.9 8.1 8.1 0 0 0 0 16.1Z" />;
}
