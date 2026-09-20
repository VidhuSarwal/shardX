import { useMemo } from 'react';

export const hasWebGL = (doc: Document = document): boolean => {
  try {
    const c = doc.createElement('canvas');
    return !!(c.getContext('webgl2') || c.getContext('webgl'));
  } catch {
    return false;
  }
};

/** True when the browser can create a WebGL context. Evaluated once per mount. */
export const useWebGL = (): boolean => useMemo(() => typeof document !== 'undefined' && hasWebGL(), []);
