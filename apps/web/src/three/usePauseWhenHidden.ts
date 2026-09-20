import { useEffect, useState } from 'react';

/** True while the tab is visible; use to set `frameloop="never"` on hidden canvases. */
export const usePauseWhenHidden = () => {
  const [visible, setVisible] = useState(!document.hidden);
  useEffect(() => {
    const on = () => setVisible(!document.hidden);
    document.addEventListener('visibilitychange', on);
    return () => document.removeEventListener('visibilitychange', on);
  }, []);
  return visible;
};
