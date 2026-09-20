import { lazy, Suspense, useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { useGSAP } from '@gsap/react';
import { ArrowDown } from 'lucide-react';
import { CHAPTERS, chapterProgress } from '@/lib/chapters';
import { useLenis } from '@/hooks/useLenis';
import { useReducedMotion } from '@/hooks/useReducedMotion';
import { useWebGL } from '@/hooks/useWebGL';
import { MagneticButton, Reveal, SplitText } from '@/components/ds';
import { Nav } from '@/components/landing/Nav';
import { ChapterRail } from '@/components/landing/ChapterRail';
import { Cursor } from '@/components/landing/Cursor';
import { Footer } from '@/components/landing/Footer';
import { CHAPTER_CONTENT } from '@/components/landing/content';
import { ChapterFallback } from '@/components/landing/Fallback';
import type { LandingSceneHandle } from '@/three/landing/LandingScene';

gsap.registerPlugin(ScrollTrigger, useGSAP);

const LandingScene = lazy(() => import('@/three/landing/LandingScene'));

const useIsMobile = () => {
  const [m, setM] = useState(() => window.innerWidth < 768);
  useEffect(() => {
    const on = () => setM(window.innerWidth < 768);
    window.addEventListener('resize', on);
    return () => window.removeEventListener('resize', on);
  }, []);
  return m;
};

const Landing = () => {
  const reduced = useReducedMotion();
  const webgl = useWebGL();
  const mobile = useIsMobile();
  const scene = useRef<LandingSceneHandle>(null);
  const main = useRef<HTMLElement>(null);
  const [current, setCurrent] = useState(0);
  useLenis(!reduced);

  // One master ScrollTrigger → scene progress + current chapter (state only on change).
  useGSAP(() => {
    if (!main.current) return;
    const st = ScrollTrigger.create({
      trigger: main.current, start: 'top top', end: 'bottom bottom', scrub: reduced ? false : 0.8,
      onUpdate: (self) => {
        scene.current?.setProgress(self.progress);
        const c = chapterProgress(self.progress).chapter;
        setCurrent((prev) => (prev === c ? prev : c));
      },
    });
    return () => st.kill();
  }, { dependencies: [reduced] });

  useEffect(() => {
    if (reduced) return;
    const on = (e: MouseEvent) => scene.current?.setMouse((e.clientX / window.innerWidth) * 2 - 1, -((e.clientY / window.innerHeight) * 2 - 1));
    window.addEventListener('mousemove', on);
    return () => window.removeEventListener('mousemove', on);
  }, [reduced]);

  return (
    <div className="relative">
      <Nav />
      <Cursor />
      {webgl ? (
        <Suspense fallback={null}>
          <LandingScene ref={scene} reducedMotion={reduced} mobile={mobile} className="fixed inset-0 z-0" />
        </Suspense>
      ) : (
        <div className="fixed inset-0 z-0 bg-[radial-gradient(ellipse_at_center,hsl(228_20%_8%),hsl(228_20%_4%))]" />
      )}
      <ChapterRail current={current} />

      <main ref={main} className="relative z-10">
        {CHAPTER_CONTENT.map((c, i) => (
          <section key={c.id} id={c.id} aria-labelledby={`${c.id}-h`} className={i === 0 ? 'flex min-h-screen items-center' : 'min-h-[120vh]'}>
            <div className="sticky top-[28vh] mx-auto w-full max-w-6xl px-6 lg:grid lg:grid-cols-2 lg:gap-12">
              <div className={i % 2 ? 'lg:col-start-2' : 'lg:col-start-1'}>
                <p className="eyebrow mb-4">{c.eyebrow}</p>
                <h2 id={`${c.id}-h`} className={`font-semibold leading-[1.02] tracking-tight ${i === 0 ? 'text-5xl md:text-7xl' : 'text-4xl md:text-6xl'}`}>
                  <SplitText text={c.headline} by="word" stagger={0.06} trigger={i !== 0} />
                </h2>
                <Reveal delay={0.2} immediate={i === 0} className="mt-6 max-w-lg text-base leading-relaxed text-muted-foreground md:text-lg"><p>{c.body}</p></Reveal>
                {c.pills.length > 0 && (
                  <Reveal delay={0.35} immediate={i === 0} className="mt-6 flex flex-wrap gap-2">
                    {c.pills.map((p) => <span key={p} className="rounded-full border border-border bg-card/60 px-3 py-1 font-mono text-xs text-foreground/80">{p}</span>)}
                  </Reveal>
                )}
                {!webgl && <div className="mt-8"><ChapterFallback id={c.id} /></div>}
                {i === 0 && (
                  <Reveal delay={0.5} immediate className="mt-10 flex flex-wrap items-center gap-3">
                    <MagneticButton size="lg" asChild><Link to="/signup">Create your vault</Link></MagneticButton>
                    <MagneticButton size="lg" variant="outline" asChild><a href="#upload">See how it works <ArrowDown className="ml-2 h-4 w-4" /></a></MagneticButton>
                  </Reveal>
                )}
                {i === CHAPTERS.length - 1 && (
                  <Reveal delay={0.4} className="mt-10 flex flex-wrap items-center gap-3">
                    <MagneticButton size="lg" asChild><Link to="/signup">Create your vault</Link></MagneticButton>
                    <MagneticButton size="lg" variant="outline" asChild><Link to="/guide">Read the guide</Link></MagneticButton>
                  </Reveal>
                )}
              </div>
            </div>
          </section>
        ))}
      </main>
      <Footer />
    </div>
  );
};
export default Landing;
