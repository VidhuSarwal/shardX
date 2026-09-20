import { Link } from 'react-router-dom';
import { useAuth } from '@/hooks/useAuth';
import { Button } from '@/components/ui/button';
import { MagneticButton } from '@/components/ds';

export const Nav = () => {
  const { isAuthenticated } = useAuth();
  return (
    <header className="fixed inset-x-0 top-0 z-50">
      <div className="mx-auto mt-4 flex max-w-6xl items-center justify-between rounded-full glass px-5 py-2.5">
        <Link to="/" className="flex items-center gap-2 font-semibold">
          <span className="h-3 w-3 rotate-45 bg-gradient-to-br from-primary to-accent shadow-glow-primary" />
          ShardX
        </Link>
        <nav className="flex items-center gap-1">
          <Button variant="ghost" size="sm" asChild><Link to="/guide">Guide</Link></Button>
          <Button variant="ghost" size="sm" asChild><a href="https://github.com/DEEPESH-845/shardX" target="_blank" rel="noreferrer">GitHub</a></Button>
          {isAuthenticated
            ? <MagneticButton size="sm" asChild><Link to="/files">Open vault</Link></MagneticButton>
            : <MagneticButton size="sm" asChild><Link to="/login">Sign in</Link></MagneticButton>}
        </nav>
      </div>
    </header>
  );
};
