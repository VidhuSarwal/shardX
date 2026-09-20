import { lazy, Suspense, useState, type ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { GlowCard, SplitText } from '@/components/ds';
import { useWebGL } from '@/hooks/useWebGL';

const AuthScene = lazy(() => import('@/three/auth/AuthScene'));

const CLAIMS = [
  'The server never sees the seed.',
  'A compromised bucket yields nothing readable.',
  'Every shard is verified. Every step is logged.',
];

export const AuthLayout = ({ title, subtitle, children, footer }: { title: string; subtitle: string; children: ReactNode; footer: ReactNode }) => {
  const webgl = useWebGL();
  const [claim] = useState(() => CLAIMS[Math.floor(Date.now() / 10000) % CLAIMS.length]);
  return (
    <div className="min-h-screen lg:grid lg:grid-cols-2">
      <aside className="relative h-[30vh] lg:h-auto overflow-hidden bg-gradient-to-br from-background via-[hsl(228_20%_6%)] to-[hsl(275_40%_8%)]">
        {webgl && (
          <Suspense fallback={null}>
            <AuthScene className="absolute inset-0" />
          </Suspense>
        )}
        <div className="relative z-10 flex h-full flex-col justify-between p-8 lg:p-12">
          <Link to="/" className="flex items-center gap-2 text-lg font-semibold">
            <span className="h-3 w-3 rotate-45 bg-gradient-to-br from-primary to-accent" />
            ShardX
          </Link>
          <p className="hidden lg:block max-w-md text-2xl font-medium leading-snug text-foreground/90">
            <SplitText text={claim} by="word" trigger={false} />
          </p>
        </div>
      </aside>
      <main className="flex items-center justify-center p-6 lg:p-12">
        <GlowCard className="w-full max-w-md p-8">
          <h1 className="text-2xl font-semibold">{title}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{subtitle}</p>
          <div className="mt-8">{children}</div>
          <div className="mt-6 text-center text-sm text-muted-foreground">{footer}</div>
        </GlowCard>
      </main>
    </div>
  );
};
