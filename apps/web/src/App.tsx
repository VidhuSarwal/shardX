import { lazy, Suspense } from 'react';
import { Toaster as Sonner } from '@/components/ui/sonner';
import { TooltipProvider } from '@/components/ui/tooltip';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { AuthProvider } from '@/hooks/useAuth';
import { Loader, Grain, PageTransition } from '@/components/ds';
import Login from './pages/Login';
import Signup from './pages/Signup';
import Files from './pages/Files';
import FileDetail from './pages/FileDetail';
import Profile from './pages/Profile';
import OAuthFinished from './pages/OAuthFinished';
import NotFound from './pages/NotFound';

const Landing = lazy(() => import('./pages/Landing'));
const Guide = lazy(() => import('./pages/Guide'));

const queryClient = new QueryClient();

const App = () => (
  <QueryClientProvider client={queryClient}>
    <TooltipProvider>
      <AuthProvider>
        <Sonner theme="dark" />
        <Grain />
        <BrowserRouter>
          <Suspense fallback={<Loader />}>
            <PageTransition>
              <Routes>
                <Route path="/" element={<Landing />} />
                <Route path="/guide" element={<Guide />} />
                <Route path="/login" element={<Login />} />
                <Route path="/signup" element={<Signup />} />
                <Route path="/files" element={<Files />} />
                <Route path="/files/:sessionId" element={<FileDetail />} />
                <Route path="/profile" element={<Profile />} />
                <Route path="/oauth/finished" element={<OAuthFinished />} />
                <Route path="*" element={<NotFound />} />
              </Routes>
            </PageTransition>
          </Suspense>
        </BrowserRouter>
      </AuthProvider>
    </TooltipProvider>
  </QueryClientProvider>
);

export default App;
