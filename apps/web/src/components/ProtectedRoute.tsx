import { ReactNode, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { getAuthToken } from '@/lib/api';

export const ProtectedRoute = ({ children }: { children: ReactNode }) => {
  const navigate = useNavigate();

  useEffect(() => {
    if (!getAuthToken()) {
      navigate('/login');
    }
  }, [navigate]);

  return getAuthToken() ? <>{children}</> : null;
};
