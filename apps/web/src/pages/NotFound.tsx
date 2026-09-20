import { Link } from 'react-router-dom';
import { MagneticButton } from '@/components/ds';
const NotFound = () => (
  <main className="grid min-h-screen place-items-center p-6 text-center">
    <div>
      <p className="eyebrow mb-4">404</p>
      <h1 className="text-4xl font-semibold tracking-tight md:text-6xl">This shard doesn't exist.</h1>
      <p className="mt-4 text-muted-foreground">The page you asked for was never written — or its key file is missing.</p>
      <div className="mt-8 flex justify-center gap-3">
        <MagneticButton asChild><Link to="/">Home</Link></MagneticButton>
        <MagneticButton variant="outline" asChild><Link to="/files">Your vault</Link></MagneticButton>
      </div>
    </div>
  </main>
);
export default NotFound;
