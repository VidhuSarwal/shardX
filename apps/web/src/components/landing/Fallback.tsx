const UPLOAD_GRID = Array.from({ length: 20 }, (_, i) => ({ x: 40 + (i % 5) * 52, y: 30 + Math.floor(i / 5) * 45 }));
const OBFUSCATE_DOTS = Array.from({ length: 60 }, (_, i) => ({ x: (i * 53 + (i % 7) * 11) % 320, y: (i * 37 + (i % 5) * 23) % 200 }));
const SHARD_CENTERS = Array.from({ length: 6 }, (_, i) => 30 + i * 45);

const renderChapter = (id: string) => {
  switch (id) {
    case 'hero':
    case 'cta':
      return <rect x="60" y="50" width="200" height="100" rx="16" fill="none" stroke="currentColor" strokeWidth="2" />;
    case 'upload':
      return UPLOAD_GRID.map((p, i) => (
        <rect key={i} x={p.x} y={p.y} width="32" height="32" rx="4" fill="none" stroke="currentColor" strokeWidth="1.5" />
      ));
    case 'obfuscate':
      return OBFUSCATE_DOTS.map((p, i) => <circle key={i} cx={p.x} cy={p.y} r="1.5" fill="currentColor" />);
    case 'shard':
      return SHARD_CENTERS.map((cx, i) => (
        <rect key={i} x={cx - 18} y="82" width="36" height="36" rx="4" fill="none" stroke="currentColor" strokeWidth="2" transform={`rotate(45 ${cx} 100)`} />
      ));
    case 'store':
      return (
        <>
          <ellipse cx="90" cy="60" rx="30" ry="10" fill="none" stroke="currentColor" strokeWidth="2" />
          <path d="M60 60 L60 140 A30 10 0 0 0 120 140 L120 60" fill="none" stroke="currentColor" strokeWidth="2" />
          <rect x="200" y="40" width="60" height="40" rx="4" fill="none" stroke="currentColor" strokeWidth="2" />
          <rect x="200" y="120" width="60" height="40" rx="4" fill="none" stroke="currentColor" strokeWidth="2" />
          <line x1="120" y1="80" x2="200" y2="60" stroke="currentColor" strokeWidth="2" />
          <line x1="120" y1="110" x2="200" y2="140" stroke="currentColor" strokeWidth="2" />
        </>
      );
    case 'keyfile':
      return (
        <>
          <rect x="60" y="40" width="200" height="120" rx="12" fill="none" stroke="currentColor" strokeWidth="2" />
          <circle cx="110" cy="100" r="18" fill="none" stroke="currentColor" strokeWidth="2" />
          <line x1="128" y1="100" x2="200" y2="100" stroke="currentColor" strokeWidth="2" />
          <line x1="180" y1="100" x2="180" y2="115" stroke="currentColor" strokeWidth="2" />
          <line x1="195" y1="100" x2="195" y2="112" stroke="currentColor" strokeWidth="2" />
        </>
      );
    case 'integrity':
      return (
        <>
          <circle cx="160" cy="100" r="50" fill="none" stroke="currentColor" strokeWidth="2" />
          <path d="M135 100 L153 118 L188 78" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
        </>
      );
    case 'drive':
      return (
        <>
          <circle cx="130" cy="100" r="45" fill="none" stroke="currentColor" strokeWidth="2" />
          <circle cx="175" cy="80" r="45" fill="none" stroke="currentColor" strokeWidth="2" />
          <circle cx="175" cy="120" r="45" fill="none" stroke="currentColor" strokeWidth="2" />
        </>
      );
    default:
      return null;
  }
};

export const ChapterFallback = ({ id }: { id: string }) => (
  <svg viewBox="0 0 320 200" className="mx-auto h-auto w-full max-w-sm text-primary" role="img" aria-label={`${id} illustration`}>
    {renderChapter(id)}
  </svg>
);
