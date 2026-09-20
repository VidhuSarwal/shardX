import { useEffect, useState } from 'react';
import { Nav } from '@/components/landing/Nav';
import { Footer } from '@/components/landing/Footer';
import { Reveal } from '@/components/ds';
import { GUIDE_SECTIONS } from '@/components/guide/sections';
import { cn } from '@/lib/utils';

const Guide = () => {
  const [active, setActive] = useState(GUIDE_SECTIONS[0].id);
  useEffect(() => {
    const io = new IntersectionObserver((entries) => {
      const top = entries.filter((e) => e.isIntersecting).sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)[0];
      if (top) setActive(top.target.id);
    }, { rootMargin: '-30% 0px -60% 0px' });
    GUIDE_SECTIONS.forEach((s) => { const el = document.getElementById(s.id); if (el) io.observe(el); });
    return () => io.disconnect();
  }, []);
  return (
    <div>
      <Nav />
      <div className="mx-auto max-w-6xl px-6 pb-24 pt-32 lg:grid lg:grid-cols-[220px_1fr] lg:gap-16">
        <aside className="mb-10 lg:sticky lg:top-32 lg:mb-0 lg:self-start">
          <p className="eyebrow mb-4">User guide</p>
          <ol className="space-y-2 text-sm">
            {GUIDE_SECTIONS.map((s, i) => (
              <li key={s.id}><a href={`#${s.id}`} className={cn('flex gap-3 transition-colors hover:text-foreground', active === s.id ? 'text-primary' : 'text-muted-foreground')}><span className="font-mono text-xs">{String(i + 1).padStart(2, '0')}</span>{s.title}</a></li>
            ))}
          </ol>
        </aside>
        <article className="prose-invert max-w-none space-y-20">
          {GUIDE_SECTIONS.map((s, i) => (
            <Reveal key={s.id} as="div" className="scroll-mt-32">
              <section id={s.id} aria-labelledby={`${s.id}-h`}>
                <p className="eyebrow mb-3">{String(i + 1).padStart(2, '0')}</p>
                <h2 id={`${s.id}-h`} className="mb-6 text-3xl font-semibold tracking-tight">{s.title}</h2>
                <div className="space-y-4 text-muted-foreground [&_code]:rounded [&_code]:bg-muted [&_code]:px-1.5 [&_code]:py-0.5 [&_code]:font-mono [&_code]:text-xs [&_code]:text-foreground [&_strong]:text-foreground [&_table]:w-full [&_table]:text-sm [&_td]:border-t [&_td]:border-border [&_td]:py-2 [&_td]:pr-4 [&_th]:pb-2 [&_th]:text-left [&_th]:text-foreground [&_ul]:list-disc [&_ul]:pl-5">{s.body}</div>
              </section>
            </Reveal>
          ))}
        </article>
      </div>
      <Footer />
    </div>
  );
};
export default Guide;
