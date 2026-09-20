import { Button } from '@/components/ui/button';
import { Moon, Sun } from 'lucide-react';
import { useTheme } from 'next-themes';

export function ThemeToggle() {
  const { theme, setTheme, systemTheme } = useTheme();
  const effectiveTheme = theme === 'system' ? systemTheme : theme;
  const toggle = () => setTheme(effectiveTheme === 'dark' ? 'light' : 'dark');

  return (
    <Button
      variant="ghost"
      size="sm"
      className="p-2"
      aria-label="Toggle theme"
      onClick={toggle}
      title={`Switch to ${effectiveTheme === 'dark' ? 'light' : 'dark'} mode`}
    >
      <Sun className="h-4 w-4 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
      <Moon className="absolute h-4 w-4 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
      <span className="sr-only">Toggle theme</span>
    </Button>
  );
}
