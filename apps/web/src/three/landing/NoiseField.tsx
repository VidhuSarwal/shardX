import { useEffect, useMemo, useRef } from 'react';
import { useFrame, useThree } from '@react-three/fiber';
import * as THREE from 'three';
import { progressStore } from './store';
import { buildTargets, CHAPTER_TARGETS } from './targets';

const vert = /* glsl */ `
  attribute vec3 aFrom;
  attribute vec3 aTo;
  attribute float aSeed;
  uniform float uT;
  uniform float uTime;
  uniform float uSwirl;
  uniform float uSize;
  varying float vSeed;
  varying float vT;
  void main() {
    // per-particle delay so the morph ripples instead of snapping
    float d = clamp((uT - aSeed * 0.35) / 0.65, 0.0, 1.0);
    float e = d * d * (3.0 - 2.0 * d);
    vec3 p = mix(aFrom, aTo, e);
    // swirl during transit, strongest mid-morph
    float mid = 4.0 * e * (1.0 - e);
    float a = uTime * 0.6 + aSeed * 6.2831;
    p += uSwirl * mid * vec3(cos(a), sin(a * 1.3), sin(a)) * 0.6;
    vSeed = aSeed; vT = e;
    vec4 mv = modelViewMatrix * vec4(p, 1.0);
    gl_PointSize = uSize * (1.0 + 0.5 * aSeed) * (8.0 / -mv.z);
    gl_Position = projectionMatrix * mv;
  }
`;
const frag = /* glsl */ `
  uniform vec3 uColorA;
  uniform vec3 uColorB;
  uniform float uMix;
  varying float vSeed;
  varying float vT;
  void main() {
    vec2 c = gl_PointCoord - 0.5;
    float r = dot(c, c);
    if (r > 0.25) discard;
    float alpha = smoothstep(0.25, 0.0, r) * (0.55 + 0.45 * vSeed);
    vec3 col = mix(uColorA, uColorB, clamp(uMix + (vSeed - 0.5) * 0.3, 0.0, 1.0));
    gl_FragColor = vec4(col, alpha);
  }
`;

const CYAN = new THREE.Color('#33e0ff');
const VIOLET = new THREE.Color('#b56bff');
const GREEN = new THREE.Color('#3dffa0');
/** Colour blend per chapter (0 = cyan, 1 = violet). Integrity chapter goes green via uColorB swap. */
const MIX_BY_CHAPTER = [0, 0.1, 1, 0.7, 0.4, 0.2, 0, 0.5, 0];

export const NoiseField = ({ count, reduced }: { count: number; reduced: boolean }) => {
  const targets = useMemo(() => buildTargets(count), [count]);
  const geo = useMemo(() => {
    const g = new THREE.BufferGeometry();
    const seeds = new Float32Array(count);
    for (let i = 0; i < count; i++) seeds[i] = Math.random();
    g.setAttribute('position', new THREE.BufferAttribute(targets.slab.slice(), 3));
    g.setAttribute('aFrom', new THREE.BufferAttribute(targets.slab.slice(), 3));
    g.setAttribute('aTo', new THREE.BufferAttribute(targets.slab.slice(), 3));
    g.setAttribute('aSeed', new THREE.BufferAttribute(seeds, 1));
    g.boundingSphere = new THREE.Sphere(new THREE.Vector3(), 10);
    return g;
  }, [count, targets]);
  const mat = useMemo(() => new THREE.ShaderMaterial({
    vertexShader: vert, fragmentShader: frag, transparent: true, depthWrite: false, blending: THREE.AdditiveBlending,
    uniforms: { uT: { value: 0 }, uTime: { value: 0 }, uSwirl: { value: 0 }, uSize: { value: 2.2 }, uColorA: { value: CYAN.clone() }, uColorB: { value: VIOLET.clone() }, uMix: { value: 0 } },
  }), []);
  const dpr = useThree((s) => s.viewport.dpr);
  const lastChapter = useRef(-1);
  const tmpB = useMemo(() => new THREE.Color(), []);
  useEffect(() => {
    lastChapter.current = -1; // geometry was rebuilt: re-copy targets on next frame
    return () => { geo.dispose(); mat.dispose(); };
  }, [geo, mat]);

  useFrame((_, dt) => {
    const { chapter, t } = progressStore;
    if (chapter !== lastChapter.current) {
      const [from, to] = CHAPTER_TARGETS[chapter];
      (geo.getAttribute('aFrom') as THREE.BufferAttribute).copyArray(targets[from]).needsUpdate = true;
      (geo.getAttribute('aTo') as THREE.BufferAttribute).copyArray(targets[to]).needsUpdate = true;
      lastChapter.current = chapter;
    }
    const u = mat.uniforms;
    u.uSize.value = 2.2 * dpr; // gl_PointSize is in device pixels
    u.uT.value = reduced ? 1 : THREE.MathUtils.damp(u.uT.value, t, 6, dt);
    u.uTime.value += dt;
    u.uSwirl.value = chapter === 2 ? 1 : chapter === 4 ? 0.5 : 0.15;
    u.uMix.value = THREE.MathUtils.damp(u.uMix.value, MIX_BY_CHAPTER[chapter], 4, dt);
    tmpB.copy(chapter === 6 ? GREEN : VIOLET);
    (u.uColorB.value as THREE.Color).lerp(tmpB, Math.min(1, dt * 3));
  });

  return <points geometry={geo} material={mat} frustumCulled={false} />;
};
