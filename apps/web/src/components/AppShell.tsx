import { useRef, useState, type ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import gsap from 'gsap';
import { useGSAP } from '@gsap/react';
import { Files, User, BookOpen, LogOut, HelpCircle, Menu, X } from 'lucide-react';
import { useAuth } from '@/hooks/useAuth';
import { getAuthEmail } from '@/lib/api';
import { getInitialsFromName, cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';

const NAV = [
  { to: '/files', label: 'Files', icon: Files, id: 'nav-files' },
  { to: '/profile', label: 'Profile', icon: User, id: 'nav-profile' },
  { to: '/guide', label: 'Guide', icon: BookOpen, id: 'nav-guide' },
];

export const AppShell = ({ children, onHelp }: { children: ReactNode; onHelp?: () => void }) => {
  const { logout } = useAuth();
  const { pathname } = useLocation();
  const [open, setOpen] = useState(false);
  const email = getAuthEmail();
  const pill = useRef<HTMLSpanElement>(null);
  const list = useRef<HTMLElement>(null);

  // Sliding active indicator.
  useGSAP(() => {
    const active = list.current?.querySelector<HTMLElement>('[aria-current="page"]');
    if (!active || !pill.current) return;
    gsap.to(pill.current, { y: active.offsetTop, height: active.offsetHeight, duration: 0.4, ease: 'power3.out' });
  }, { dependencies: [pathname], scope: list });

  const nav = (
    <nav ref={list} className="relative flex flex-col gap-1" aria-label="Primary">
      <span ref={pill} aria-hidden className="absolute left-0 w-full rounded-lg bg-primary/10 ring-1 ring-primary/30" style={{ height: 0 }} />
      {NAV.map(({ to, label, icon: Icon, id }) => {
        const active = pathname === to || (to === '/files' && pathname.startsWith('/files/'));
        return (
          <Link key={to} id={id} to={to} aria-current={active ? 'page' : undefined} onClick={() => setOpen(false)}
            className={cn('relative z-10 flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm transition-colors', active ? 'text-primary' : 'text-muted-foreground hover:text-foreground')}>
            <Icon className="h-4 w-4" /> {label}
          </Link>
        );
      })}
    </nav>
  );

  const user = (
    <div className="mt-auto space-y-2">
      <Button id="tour-help" variant="ghost" size="sm" className="w-full justify-start gap-3 text-muted-foreground" onClick={onHelp}><HelpCircle className="h-4 w-4" /> Show me around</Button>
      <div className="flex items-center gap-3 rounded-lg border border-border/60 p-2">
        <Avatar className="h-8 w-8"><AvatarFallback className="bg-gradient-to-br from-primary to-accent text-xs text-background">{getInitialsFromName(email ?? 'Account')}</AvatarFallback></Avatar>
        <span className="min-w-0 flex-1 truncate text-xs text-muted-foreground">{email ?? 'Account'}</span>
        <Button variant="ghost" size="icon" aria-label="Log out" onClick={logout}><LogOut className="h-4 w-4" /></Button>
      </div>
    </div>
  );

  return (
    <div className="min-h-screen lg:grid lg:grid-cols-[240px_1fr]">
      <aside className="hidden lg:flex lg:flex-col lg:gap-8 lg:border-r lg:border-border/60 lg:p-5 lg:sticky lg:top-0 lg:h-screen">
        <Link to="/" className="flex items-center gap-2 px-2 font-semibold"><span className="h-3 w-3 rotate-45 bg-gradient-to-br from-primary to-accent shadow-glow-primary" /> ShardX</Link>
        {nav}
        {user}
      </aside>
      <header className="sticky top-0 z-40 flex items-center justify-between border-b border-border/60 glass px-4 py-3 lg:hidden">
        <Link to="/" className="flex items-center gap-2 font-semibold"><span className="h-3 w-3 rotate-45 bg-gradient-to-br from-primary to-accent" /> ShardX</Link>
        <Button variant="ghost" size="icon" aria-label={open ? 'Close menu' : 'Open menu'} aria-expanded={open} onClick={() => setOpen((v) => !v)}>{open ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}</Button>
      </header>
      {open && (
        <div className="fixed inset-0 z-30 flex flex-col gap-6 bg-background p-5 pt-20 lg:hidden">
          {nav}{user}
        </div>
      )}
      <main className="mx-auto w-full max-w-[1200px] px-5 py-8 lg:px-8 lg:py-10">{children}</main>
    </div>
  );
};
