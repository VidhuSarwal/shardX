export const Loader = ({ label = 'Loading' }: { label?: string }) => (
  <div role="status" aria-live="polite" className="fixed inset-0 z-[100] grid place-items-center bg-background">
    <div className="flex flex-col items-center gap-4">
      <div className="h-10 w-10 rotate-45 rounded-sm bg-gradient-to-br from-primary to-accent animate-pulse shadow-glow-primary" />
      <span className="eyebrow">{label}</span>
    </div>
  </div>
);
