import { forwardRef, useImperativeHandle, useEffect, useRef, useState, type ReactNode } from 'react';
import { Canvas, useFrame, useThree } from '@react-three/fiber';
import { EffectComposer, Bloom, Vignette } from '@react-three/postprocessing';
import * as THREE from 'three';
import { chapterProgress } from '@/lib/chapters';
import { progressStore } from './store';
import { NoiseField } from './NoiseField';
import { Accents } from './Accents';

export type LandingSceneHandle = { setProgress(p: number): void; setMouse(x: number, y: number): void };

const CameraRig = ({ reduced, mobile }: { reduced: boolean; mobile: boolean }) => {
  const { camera } = useThree();
  useFrame((_, dt) => {
    const { chapter, mouse } = progressStore;
    const z = chapter === 2 ? 9 : chapter === 4 || chapter === 7 ? 8 + (mobile ? 0 : 1.0) : 7;
    const targetX = reduced ? 0 : mouse[0] * 0.6;
    const targetY = (reduced ? 0 : mouse[1] * 0.4) + (chapter === 0 || chapter === 8 ? 1.2 : 0);
    camera.position.x = THREE.MathUtils.damp(camera.position.x, targetX, 3, dt);
    camera.position.y = THREE.MathUtils.damp(camera.position.y, targetY, 3, dt);
    camera.position.z = THREE.MathUtils.damp(camera.position.z, z, 2, dt);
    camera.lookAt(0, 0, 0);
  });
  return null;
};

/** Shifts the noise field + accents opposite the copy column so desktop chapters don't overlap the text. */
const SceneShift = ({ mobile, reducedMotion, children }: { mobile: boolean; reducedMotion: boolean; children: ReactNode }) => {
  const shift = useRef<THREE.Group>(null);
  useFrame((_, dt) => {
    const g = shift.current;
    if (!g) return;
    const target = mobile ? 0 : progressStore.chapter % 2 === 0 ? 2.3 : -2.3;
    g.position.x = reducedMotion ? target : THREE.MathUtils.damp(g.position.x, target, 2.5, dt);
  });
  return <group ref={shift}>{children}</group>;
};

/** Pauses rendering when the tab is hidden. */
const usePauseWhenHidden = () => {
  const [visible, setVisible] = useState(!document.hidden);
  useEffect(() => {
    const on = () => setVisible(!document.hidden);
    document.addEventListener('visibilitychange', on);
    return () => document.removeEventListener('visibilitychange', on);
  }, []);
  return visible;
};

const LandingScene = forwardRef<LandingSceneHandle, { reducedMotion: boolean; mobile: boolean; className?: string }>(
  ({ reducedMotion, mobile, className }, ref) => {
    useImperativeHandle(ref, () => ({
      setProgress(p) {
        const { chapter, t } = chapterProgress(p);
        progressStore.p = p; progressStore.chapter = chapter; progressStore.t = t;
      },
      setMouse(x, y) { progressStore.mouse = [x, y]; },
    }), []);
    const visible = usePauseWhenHidden();
    const count = mobile ? 6000 : 20000;
    return (
      <div className={className} aria-hidden>
        <Canvas dpr={[1, 2]} frameloop={visible ? 'always' : 'never'} camera={{ position: [0, 0, 7], fov: 45 }} gl={{ antialias: true, powerPreference: 'high-performance', alpha: true }}>
          <color attach="background" args={['#07080d']} />
          <ambientLight intensity={0.4} />
          <pointLight position={[4, 4, 6]} intensity={30} color="#33e0ff" />
          <pointLight position={[-4, -2, 4]} intensity={20} color="#b56bff" />
          <CameraRig reduced={reducedMotion} mobile={mobile} />
          <SceneShift mobile={mobile} reducedMotion={reducedMotion}>
            <NoiseField count={count} reduced={reducedMotion} />
            <Accents reduced={reducedMotion} />
          </SceneShift>
          {!mobile && (
            <EffectComposer>
              <Bloom intensity={0.9} luminanceThreshold={0.2} luminanceSmoothing={0.6} mipmapBlur />
              <Vignette eskil={false} offset={0.2} darkness={0.8} />
            </EffectComposer>
          )}
        </Canvas>
      </div>
    );
  },
);
LandingScene.displayName = 'LandingScene';
export default LandingScene;
