import { useEffect } from 'react';
import { Link } from 'react-router-dom';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
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
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-background via-background to-secondary p-4">
      <Card className="w-full max-w-md shadow-[var(--shadow-elevated)]">
        <CardHeader className="text-center space-y-4">
          <div className="flex justify-center">
            <div className="p-4 rounded-full bg-success/10">
              <CheckCircle className="w-12 h-12 text-success" />
            </div>
          </div>
          <CardTitle className="text-2xl">Authorization Complete!</CardTitle>
          <CardDescription>
            Your Google Drive has been successfully connected to shardX
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="p-4 bg-muted/50 rounded-lg text-sm text-muted-foreground">
            You can now close this window and return to your profile to see your connected drive account.
          </div>

          <Button asChild className="w-full">
            <Link to="/profile">
              <ArrowLeft className="w-4 h-4 mr-2" />
              Back to Profile
            </Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  );
};

export default OAuthFinished;
