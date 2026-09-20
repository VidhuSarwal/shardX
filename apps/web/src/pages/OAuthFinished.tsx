import { useEffect } from 'react';
import { Link } from 'react-router-dom';
import { GlowCard, MagneticButton } from '@/components/ds';
import { CheckCircle, ArrowLeft } from 'lucide-react';
import { FRONTEND_BASE_URL } from '@/lib/config';

const OAuthFinished = () => {
  useEffect(() => {
    try {
      if (window.opener) {
        window.opener.postMessage(
          { type: 'oauth_finished', success: true, provider: 'google' },
          FRONTEND_BASE_URL,
        );
      }
      // Give the opener a moment to handle, then close if we were opened as a popup
      const timer = setTimeout(() => {
        if (window.opener && !window.opener.closed) {
          window.close();
        }
      }, 300);
      return () => clearTimeout(timer);
    } catch {
      // no-op; UI below provides manual navigation
    }
  }, []);

  return (
    <div className="grid min-h-screen place-items-center p-4">
      <GlowCard className="w-full max-w-md text-center">
        <div className="flex justify-center">
          <div className="grid h-16 w-16 place-items-center rounded-full bg-success/10">
            <CheckCircle className="h-8 w-8 text-success" />
          </div>
        </div>
        <h1 className="mt-4 text-2xl font-semibold">Authorization complete</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Your Google Drive has been successfully connected to ShardX
        </p>

        <div className="mt-6 rounded-lg bg-muted/50 p-4 text-sm text-muted-foreground">
          You can now close this window and return to your profile to see your connected drive account.
        </div>

        <MagneticButton asChild className="mt-6 w-full">
          <Link to="/profile">
            <ArrowLeft className="h-4 w-4" />
            Back to profile
          </Link>
        </MagneticButton>
      </GlowCard>
    </div>
  );
};

export default OAuthFinished;
