import { Link } from 'react-router-dom';
export const Footer = () => (
  <footer className="relative z-10 border-t border-border/60 bg-background/80 py-10 backdrop-blur">
    <div className="mx-auto flex max-w-6xl flex-col items-center justify-between gap-4 px-6 text-sm text-muted-foreground md:flex-row">
      <span>© {new Date().getFullYear()} ShardX · MIT</span>
      <div className="flex gap-6">
        <Link to="/guide" className="hover:text-foreground">Guide</Link>
        <a href="https://github.com/DEEPESH-845/shardX" className="hover:text-foreground" target="_blank" rel="noreferrer">GitHub</a>
        <Link to="/login" className="hover:text-foreground">Sign in</Link>
      </div>
    </div>
  </footer>
);
