import { ReactNode } from 'react';
import { Navigate } from 'react-router-dom';
import { getAuthToken } from '@/lib/api';

export const ProtectedRoute = ({ children }: { children: ReactNode }) =>
  getAuthToken() ? <>{children}</> : <Navigate to="/login" replace />;
