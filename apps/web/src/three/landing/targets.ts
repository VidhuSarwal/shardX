export type TargetName = 'slab' | 'grid' | 'noise' | 'shards' | 'nodes' | 'key' | 'ring' | 'drive';

export const SHARD_POSITIONS: [number, number, number][] = [
  [-2.2, 1.0, 0], [0, 1.4, -0.5], [2.2, 1.0, 0], [-2.2, -1.0, 0], [0, -1.4, -0.5], [2.2, -1.0, 0],
];
export const NODE_POSITIONS = { s3: [0, 0, 0] as [number, number, number], kms: [-2.8, 1.6, -1] as [number, number, number], dynamo: [2.8, 1.6, -1] as [number, number, number] };
export const DRIVE_POSITIONS: [number, number, number][] = [[-2.4, 0, 0], [0, 0.6, -0.4], [2.4, 0, 0]];

/** from → to per chapter (index = chapter). */
export const CHAPTER_TARGETS: [TargetName, TargetName][] = [
  ['slab', 'slab'], ['slab', 'grid'], ['grid', 'noise'], ['noise', 'shards'], ['shards', 'nodes'],
  ['nodes', 'key'], ['key', 'ring'], ['ring', 'drive'], ['drive', 'slab'],
];

// mulberry32 — tiny seeded PRNG so tests are deterministic.
const rng = (seed: number) => () => {
  seed |= 0; seed = (seed + 0x6d2b79f5) | 0;
  let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
  t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
  return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
};

const fill = (n: number, f: (i: number, out: Float32Array) => void) => {
  const out = new Float32Array(n * 3);
  for (let i = 0; i < n; i++) f(i * 3, out);
  return out;
};

const cluster = (n: number, r: () => number, centers: [number, number, number][], radius: number) =>
  fill(n, (o, out) => {
    const c = centers[Math.floor(r() * centers.length)];
    const u = r() * 2 - 1, phi = r() * Math.PI * 2, rad = Math.cbrt(r()) * radius;
    const s = Math.sqrt(1 - u * u);
    out[o] = c[0] + rad * s * Math.cos(phi); out[o + 1] = c[1] + rad * s * Math.sin(phi); out[o + 2] = c[2] + rad * u;
  });

export const buildTargets = (n: number, seed = 1): Record<TargetName, Float32Array> => {
  const r = rng(seed);
  const slab = fill(n, (o, out) => { out[o] = (r() - 0.5) * 3; out[o + 1] = (r() - 0.5) * 2; out[o + 2] = (r() - 0.5) * 0.15; });
  const grid = fill(n, (o, out) => {
    const col = Math.floor(r() * 5), row = Math.floor(r() * 4);
    out[o] = -1.6 + col * 0.8 + (r() - 0.5) * 0.6; out[o + 1] = 1.2 - row * 0.8 + (r() - 0.5) * 0.6; out[o + 2] = (r() - 0.5) * 0.1;
  });
  const noise = fill(n, (o, out) => {
    const u = r() * 2 - 1, phi = r() * Math.PI * 2, rad = 1.5 + r() * 3;
    const s = Math.sqrt(1 - u * u);
    out[o] = rad * s * Math.cos(phi); out[o + 1] = rad * s * Math.sin(phi); out[o + 2] = rad * u;
  });
  const shards = cluster(n, r, SHARD_POSITIONS, 0.45);
  const nodes = cluster(n, r, [NODE_POSITIONS.s3, NODE_POSITIONS.s3, NODE_POSITIONS.s3, NODE_POSITIONS.kms, NODE_POSITIONS.dynamo], 0.7);
  const key = fill(n, (o, out) => { out[o] = (r() - 0.5) * 2.2; out[o + 1] = (r() - 0.5) * 1.4; out[o + 2] = 0.8 + (r() - 0.5) * 0.05; });
  const ring = fill(n, (o, out) => {
    const a = r() * Math.PI * 2, b = r() * Math.PI * 2, R = 1.8, tube = 0.25;
    out[o] = (R + tube * Math.cos(b)) * Math.cos(a); out[o + 1] = (R + tube * Math.cos(b)) * Math.sin(a); out[o + 2] = tube * Math.sin(b);
  });
  const drive = cluster(n, r, DRIVE_POSITIONS, 0.8);
  return { slab, grid, noise, shards, nodes, key, ring, drive };
};
