import { useState, useEffect } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { api, getAuthToken } from '@/lib/api';
import { toast } from 'sonner';
import { AuthLayout } from '@/components/AuthLayout';
import { MagneticButton } from '@/components/ds';
import { Eye, EyeOff, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';

const Signup = () => {
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [errors, setErrors] = useState<{ email?: string; password?: string; confirm?: string }>({});

  useEffect(() => {
    if (getAuthToken()) {
      navigate('/files');
    }
  }, [navigate]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrors({});

    if (!email || !password || !confirmPassword) {
      setErrors({ email: email ? undefined : 'Email is required', password: password ? undefined : 'Password is required', confirm: confirmPassword ? undefined : 'Please confirm your password' });
      toast.error('Please fill in all fields');
      return;
    }

    if (password.length < 6) {
      setErrors({ password: 'Password must be at least 6 characters' });
      toast.error('Password must be at least 6 characters');
      return;
    }

    if (password !== confirmPassword) {
      setErrors({ confirm: 'Passwords do not match' });
      toast.error('Passwords do not match');
      return;
    }

    setIsLoading(true);

    try {
      await api.signup(email, password);
      toast.success('Account created! Please sign in.');
      navigate('/login');
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Signup failed');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <AuthLayout
      title="Create your vault"
      subtitle="Zero-knowledge storage starts with an account."
      footer={<>Already have an account? <Link to="/login" className="text-primary hover:underline">Sign in</Link></>}
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
            <Input id="password" type={showPassword ? 'text' : 'password'} autoComplete="new-password" placeholder="••••••••" value={password} onChange={(e) => setPassword(e.target.value)} disabled={isLoading} required aria-invalid={!!errors.password} aria-describedby={errors.password ? 'password-error' : undefined} className={cn('pr-10', errors.password && 'border-destructive focus-visible:ring-destructive')} />
            <button type="button" aria-label={showPassword ? 'Hide password' : 'Show password'} onClick={() => setShowPassword((v) => !v)} className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-muted-foreground hover:text-foreground">
              {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
          </div>
          {errors.password && <p id="password-error" className="text-xs text-destructive">{errors.password}</p>}
        </div>
        <div className="space-y-2">
          <Label htmlFor="confirmPassword">Confirm password</Label>
          <Input id="confirmPassword" type={showPassword ? 'text' : 'password'} autoComplete="new-password" placeholder="••••••••" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} disabled={isLoading} required aria-invalid={!!errors.confirm} aria-describedby={errors.confirm ? 'confirm-error' : undefined} className={cn(errors.confirm && 'border-destructive focus-visible:ring-destructive')} />
          {errors.confirm && <p id="confirm-error" className="text-xs text-destructive">{errors.confirm}</p>}
        </div>
        <MagneticButton type="submit" className="w-full" disabled={isLoading}>
          {isLoading ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
          {isLoading ? 'Creating…' : 'Create account'}
        </MagneticButton>
      </form>
    </AuthLayout>
  );
};

export default Signup;
