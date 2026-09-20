export const FRONTEND_BASE_URL: string =
  (import.meta as any).env?.VITE_BASE_URL || window.location.origin;
