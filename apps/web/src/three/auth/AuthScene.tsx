import { useMemo, useRef } from 'react';
import { Canvas, useFrame } from '@react-three/fiber';
import { Instances, Instance } from '@react-three/drei';
import { EffectComposer, Bloom } from '@react-three/postprocessing';
import * as THREE from 'three';
import { useReducedMotion } from '@/hooks/useReducedMotion';

const COUNT = 180;

const Cloud = ({ reduced }: { reduced: boolean }) => {
  const group = useRef<THREE.Group>(null);
  const items = useMemo(() => Array.from({ length: COUNT }, (_, i) => ({
    pos: new THREE.Vector3((Math.random() - 0.5) * 10, (Math.random() - 0.5) * 8, (Math.random() - 0.5) * 6),
    rot: new THREE.Euler(Math.random() * Math.PI, Math.random() * Math.PI, 0),
    scale: 0.08 + Math.random() * 0.22,
    speed: 0.2 + Math.random() * 0.6,
    color: i % 5 === 0 ? '#b56bff' : '#33e0ff',
  })), []);
  useFrame(({ pointer, clock }, dt) => {
    if (!group.current || reduced) return;
    group.current.rotation.y = THREE.MathUtils.damp(group.current.rotation.y, pointer.x * 0.25, 2, dt);
    group.current.rotation.x = THREE.MathUtils.damp(group.current.rotation.x, -pointer.y * 0.2, 2, dt);
    // <Instances> renders one mesh whose children are the <Instance> objects.
    const inst = group.current.children[0];
    inst?.children.forEach((c, i) => {
      const it = items[i];
      if (!it) return;
      c.position.y = it.pos.y + Math.sin(clock.elapsedTime * it.speed + i) * 0.25;
      c.rotation.z += dt * 0.2 * it.speed;
    });
  });
  return (
    <group ref={group}>
      <Instances limit={COUNT}>
        <octahedronGeometry args={[1, 0]} />
        <meshStandardMaterial emissiveIntensity={0.8} metalness={0.3} roughness={0.4} />
        {items.map((it, i) => <Instance key={i} position={it.pos} rotation={it.rot} scale={it.scale} color={it.color} />)}
      </Instances>
    </group>
  );
};

const AuthScene = ({ className }: { className?: string }) => {
  const reduced = useReducedMotion();
  return (
    <div className={className} aria-hidden>
      <Canvas dpr={[1, 2]} frameloop={reduced ? 'demand' : 'always'} camera={{ position: [0, 0, 8], fov: 50 }} gl={{ antialias: true, alpha: true, powerPreference: 'high-performance' }}>
        <ambientLight intensity={0.5} />
        <pointLight position={[3, 3, 5]} intensity={25} color="#33e0ff" />
        <pointLight position={[-3, -2, 3]} intensity={15} color="#b56bff" />
        <Cloud reduced={reduced} />
        <EffectComposer><Bloom intensity={0.7} luminanceThreshold={0.3} mipmapBlur /></EffectComposer>
      </Canvas>
    </div>
  );
};
export default AuthScene;
