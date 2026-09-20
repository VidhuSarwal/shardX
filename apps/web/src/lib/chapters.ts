export const CHAPTERS = [
  { id: 'hero', label: 'Noise' },
  { id: 'upload', label: 'Upload' },
  { id: 'obfuscate', label: 'Obfuscate' },
  { id: 'shard', label: 'Shard' },
  { id: 'store', label: 'Store' },
  { id: 'keyfile', label: 'Key File' },
  { id: 'integrity', label: 'Integrity' },
  { id: 'drive', label: 'Drive mode' },
  { id: 'cta', label: 'Begin' },
] as const;

/** Maps global scroll progress p∈[0,1] to { chapter, t∈[0,1] within that chapter }. */
export const chapterProgress = (p: number, n: number = CHAPTERS.length) => {
  const clamped = Math.min(1, Math.max(0, p));
  if (clamped === 1) return { chapter: n - 1, t: 1 };
  const scaled = clamped * n;
  const chapter = Math.floor(scaled);
  return { chapter, t: scaled - chapter };
};
