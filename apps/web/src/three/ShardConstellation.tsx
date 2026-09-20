import { useMemo, useState } from 'react';
import { Canvas, useFrame } from '@react-three/fiber';
import { Html, OrbitControls } from '@react-three/drei';
import * as THREE from 'three';
import type { ShardRecord } from '@/lib/api';
import { groupShardsByTarget, formatBytes } from '@/lib/fileDetail';
import { useReducedMotion } from '@/hooks/useReducedMotion';

const STATUS_COLOR: Record<string, string> = { verified: '#3dffa0', pending: '#ffb02e', corrupted: '#ff4d4d', missing: '#7a2a2a' };

const Shard = ({ s, position }: { s: ShardRecord; position: [number, number, number] }) => {
  const [hover, setHover] = useState(false);
  const color = STATUS_COLOR[s.status] ?? '#888';
  return (
    <mesh position={position} scale={hover ? 1.35 : 1}
      onPointerOver={(e) => { e.stopPropagation(); setHover(true); }}
      onPointerOut={() => setHover(false)}>
      <octahedronGeometry args={[0.22, 0]} />
      <meshStandardMaterial color={color} emissive={color} emissiveIntensity={s.status === 'pending' ? 1.2 : 0.6} />
      {hover && (
        <Html center distanceFactor={6} className="pointer-events-none whitespace-nowrap rounded-md border border-border bg-background/90 px-2 py-1 font-mono text-[11px]">
          <div>shard #{s.shard_id} · {s.status}</div>
          <div>{formatBytes(s.size)}</div>
          <div className="text-muted-foreground">sha256 {s.sha256.slice(0, 16)}…</div>
        </Html>
      )}
    </mesh>
  );
};

const Rings = ({ shards, driveNames, reduced }: { shards: ShardRecord[]; driveNames: Record<string, string>; reduced: boolean }) => {
  const targets = useMemo(() => groupShardsByTarget(shards, driveNames), [shards, driveNames]);
  const groups = useMemo(() => targets.map((t, ti) => {
    const R = 1.2 + ti * 1.1;
    return { label: t.label, R, items: t.shards.map((s, i) => {
      const a = (i / t.shards.length) * Math.PI * 2;
      return { s, position: [Math.cos(a) * R, (ti % 2 ? 0.3 : -0.3), Math.sin(a) * R] as [number, number, number] };
    }) };
  }), [targets]);
  const ref = useState(() => new THREE.Group())[0];
  useFrame((_, dt) => { if (!reduced) ref.rotation.y += dt * 0.08; });
  return (
    <primitive object={ref}>
      {groups.map((g, gi) => (
        <group key={gi}>
          <mesh rotation={[Math.PI / 2, 0, 0]}><torusGeometry args={[g.R, 0.006, 8, 128]} /><meshBasicMaterial color="#1e2433" /></mesh>
          <Html position={[g.R + 0.35, 0, 0]} center className="pointer-events-none whitespace-nowrap font-mono text-[10px] text-muted-foreground">{g.label}</Html>
          {g.items.map(({ s, position }) => <Shard key={s.shard_id} s={s} position={position} />)}
        </group>
      ))}
    </primitive>
  );
};

const ShardConstellation = ({ shards, driveNames }: { shards: ShardRecord[]; driveNames: Record<string, string> }) => {
  const reduced = useReducedMotion();
  return (
    <div className="h-[420px] w-full rounded-2xl glass overflow-hidden" aria-hidden>
      <Canvas dpr={[1, 2]} frameloop={reduced ? 'demand' : 'always'} camera={{ position: [0, 3.5, 6], fov: 45 }} gl={{ antialias: true, alpha: true }}>
        <ambientLight intensity={0.6} />
        <pointLight position={[4, 5, 4]} intensity={30} color="#33e0ff" />
        <Rings shards={shards} driveNames={driveNames} reduced={reduced} />
        <OrbitControls enablePan={false} enableZoom={false} maxPolarAngle={Math.PI / 2.2} minPolarAngle={Math.PI / 4} />
      </Canvas>
    </div>
  );
};
export default ShardConstellation;
