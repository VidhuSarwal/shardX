import { useState, useEffect } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { api, setAuthToken, getAuthToken } from '@/lib/api';
import { toast } from 'sonner';
import { AuthLayout } from '@/components/AuthLayout';
import { MagneticButton } from '@/components/ds';
import { Eye, EyeOff, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';

const Login = () => {
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [errors, setErrors] = useState<{ email?: string; password?: string }>({});

  useEffect(() => {
    if (getAuthToken()) {
      navigate('/files');
    }
  }, [navigate]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrors({});

    if (!email || !password) {
      setErrors({ email: email ? undefined : 'Email is required', password: password ? undefined : 'Password is required' });
      toast.error('Please fill in all fields');
      return;
    }

    if (password.length < 6) {
      setErrors({ password: 'Password must be at least 6 characters' });
      toast.error('Password must be at least 6 characters');
      return;
    }

    setIsLoading(true);

    try {
      const { token } = await api.login(email, password);
      setAuthToken(token, email);
      toast.success('Logged in successfully');
      navigate('/files');
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Login failed');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <AuthLayout
      title="Welcome back"
      subtitle="Sign in to open your vault."
      footer={<>No account? <Link to="/signup" className="text-primary hover:underline">Create one</Link></>}
    >
      <form onSubmit={handleSubmit} className="space-y-5" noValidate>
        <div className="space-y-2">
          <Label htmlFor="email">Email</Label>
          <Input id="email" type="email" autoComplete="email" placeholder="you@example.com" value={email} onChange={(e) => setEmail(e.target.value)} disabled={isLoading} required aria-invalid={!!errors.email} aria-describedby={errors.email ? 'email-error' : undefined} className={cn(errors.email && 'border-destructive focus-visible:ring-destructive')} />
          {errors.email && <p id="email-error" className="text-xs text-destructive">{errors.email}</p>}
        </div>
        <div className="space-y-2">
          <Label htmlFor="password">Password</Label>
          <div className="relative">
            <Input id="password" type={showPassword ? 'text' : 'password'} autoComplete="current-password" placeholder="••••••••" value={password} onChange={(e) => setPassword(e.target.value)} disabled={isLoading} required aria-invalid={!!errors.password} aria-describedby={errors.password ? 'password-error' : undefined} className={cn('pr-10', errors.password && 'border-destructive focus-visible:ring-destructive')} />
            <button type="button" aria-label={showPassword ? 'Hide password' : 'Show password'} onClick={() => setShowPassword((v) => !v)} className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-muted-foreground hover:text-foreground">
              {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
          </div>
          {errors.password && <p id="password-error" className="text-xs text-destructive">{errors.password}</p>}
        </div>
        <MagneticButton type="submit" className="w-full" disabled={isLoading}>
          {isLoading ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
          {isLoading ? 'Signing in…' : 'Sign in'}
        </MagneticButton>
      </form>
    </AuthLayout>
  );
};

export default Login;
