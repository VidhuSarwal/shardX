import { useRef, useState } from 'react';
import { useFrame } from '@react-three/fiber';
import { Html, RoundedBox } from '@react-three/drei';
import * as THREE from 'three';
import { progressStore } from './store';
import { SHARD_POSITIONS, NODE_POSITIONS, DRIVE_POSITIONS } from './targets';

/**
 * Scales a group 0↔1 depending on whether the current chapter is in `chapters`.
 * The group is mounted only while on-screen (drei <Html> labels are DOM portals and
 * ignore ancestor scale/visibility); the state flip happens only at chapter edges.
 */
const useChapterGroup = (chapters: number[], reduced: boolean, enter = 0.15) => {
  const ref = useRef<THREE.Group>(null);
  const [mounted, setMounted] = useState(false);
  const mountedRef = useRef(false);
  useFrame((_, dt) => {
    const { chapter, t } = progressStore;
    const on = chapters.includes(chapter) && (chapters[0] !== chapter || t > enter);
    const g = ref.current;
    if (g) g.scale.setScalar(reduced ? (on ? 1 : 0.0001) : THREE.MathUtils.damp(g.scale.x, on ? 1 : 0.0001, 5, dt));
    if (on && !mountedRef.current) { mountedRef.current = true; setMounted(true); }
    else if (!on && mountedRef.current && (!g || g.scale.x < 0.01)) { mountedRef.current = false; setMounted(false); }
  });
  return [ref, mounted] as const;
};

const Label = ({ children, position }: { children: string; position: [number, number, number] }) => (
  <Html position={position} center distanceFactor={8} className="pointer-events-none select-none whitespace-nowrap rounded-md border border-border bg-background/70 px-2 py-1 font-mono text-[11px] text-foreground/90 backdrop-blur">
    {children}
  </Html>
);

const Prism = ({ position, color = '#33e0ff' }: { position: [number, number, number]; color?: string }) => (
  <mesh position={position}>
    <octahedronGeometry args={[0.32, 0]} />
    <meshStandardMaterial color={color} emissive={color} emissiveIntensity={0.6} metalness={0.2} roughness={0.3} />
  </mesh>
);

export const Accents = ({ reduced }: { reduced: boolean }) => {
  const [shards, shardsOn] = useChapterGroup([3, 4], reduced);
  const [nodes, nodesOn] = useChapterGroup([4], reduced);
  const [key, keyOn] = useChapterGroup([5], reduced);
  const [ring, ringOn] = useChapterGroup([6], reduced);
  const [drive, driveOn] = useChapterGroup([7], reduced);
  const retry = useRef<THREE.Mesh>(null);
  const ringMat = useRef<THREE.MeshStandardMaterial>(null);

  useFrame(({ clock }) => {
    // Chapter 4: one shard "fails", arcs back out, then re-lands (SQS retry).
    if (retry.current) {
      const { chapter, t } = progressStore;
      const k = chapter === 4 ? THREE.MathUtils.smoothstep(t, 0.3, 0.9) : 0;
      const bounce = Math.sin(k * Math.PI);
      retry.current.position.set(2.2 * (1 - k) + 0.4 * bounce, -1.0 * (1 - k) + 1.6 * bounce, 0.5 * bounce);
      retry.current.rotation.y = clock.elapsedTime * 2;
    }
    if (ringMat.current) {
      const { chapter, t } = progressStore;
      ringMat.current.emissiveIntensity = chapter === 6 ? 0.4 + t * 1.2 : 0.4;
    }
  });

  return (
    <>
      {shardsOn && <group ref={shards} scale={0.0001}>
        {SHARD_POSITIONS.map((p, i) => (
          <group key={i}>
            <Prism position={p} />
            <Label position={[p[0], p[1] - 0.6, p[2]]}>{`chunk_00${i + 1}.2xpfm`}</Label>
          </group>
        ))}
      </group>}
      {nodesOn && <group ref={nodes} scale={0.0001}>
        <mesh position={NODE_POSITIONS.s3}><cylinderGeometry args={[0.9, 0.9, 0.5, 48]} /><meshStandardMaterial color="#0b1a22" emissive="#33e0ff" emissiveIntensity={0.25} metalness={0.6} roughness={0.35} /></mesh>
        <Label position={[0, -0.75, 0]}>S3 · SSE-KMS</Label>
        <mesh position={NODE_POSITIONS.kms}><icosahedronGeometry args={[0.45, 1]} /><meshStandardMaterial color="#1a0b22" emissive="#b56bff" emissiveIntensity={0.5} /></mesh>
        <Label position={[NODE_POSITIONS.kms[0], NODE_POSITIONS.kms[1] - 0.7, NODE_POSITIONS.kms[2]]}>KMS CMK</Label>
        <mesh position={NODE_POSITIONS.dynamo}><boxGeometry args={[0.7, 0.7, 0.7]} /><meshStandardMaterial color="#0b1a22" emissive="#33e0ff" emissiveIntensity={0.35} /></mesh>
        <Label position={[NODE_POSITIONS.dynamo[0], NODE_POSITIONS.dynamo[1] - 0.7, NODE_POSITIONS.dynamo[2]]}>DynamoDB · sha256</Label>
        <mesh ref={retry}><octahedronGeometry args={[0.28, 0]} /><meshStandardMaterial color="#ffb02e" emissive="#ffb02e" emissiveIntensity={0.9} /></mesh>
        <Label position={[1.4, 1.2, 0.5]}>SQS retry</Label>
      </group>}
      {keyOn && <group ref={key} scale={0.0001}>
        <RoundedBox args={[2.4, 1.5, 0.06]} radius={0.06} position={[0, 0, 0.8]}>
          <meshStandardMaterial color="#0a0c14" emissive="#33e0ff" emissiveIntensity={0.15} metalness={0.7} roughness={0.25} />
        </RoundedBox>
        <Label position={[0, 0.45, 0.85]}>.2xpfm.key</Label>
        <Label position={[0, -0.1, 0.85]}>seed · chunk map · never uploaded</Label>
      </group>}
      {ringOn && <group ref={ring} scale={0.0001}>
        <mesh><torusGeometry args={[1.8, 0.06, 16, 96]} /><meshStandardMaterial ref={ringMat} color="#062015" emissive="#3dffa0" emissiveIntensity={0.4} /></mesh>
        <Label position={[0, 0, 0]}>100% · all shards verified</Label>
      </group>}
      {driveOn && <group ref={drive} scale={0.0001}>
        {DRIVE_POSITIONS.map((p, i) => (
          <group key={i}>
            <mesh position={p} rotation={[Math.PI / 2, 0, 0]}><cylinderGeometry args={[0.8, 0.8, 0.12, 48]} /><meshStandardMaterial color="#0b1a22" emissive="#33e0ff" emissiveIntensity={0.3} /></mesh>
            <Label position={[p[0], p[1] - 1.0, p[2]]}>{`Drive ${i + 1} · 15 GB`}</Label>
          </group>
        ))}
      </group>}
    </>
  );
};
