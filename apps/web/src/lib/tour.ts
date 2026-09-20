export type TourStep = { selector: string; title: string; body: string; placement: 'top' | 'bottom' | 'left' | 'right' };
export type TourState = { open: boolean; index: number };
export type TourAction = { type: 'start' } | { type: 'next'; available: boolean[] } | { type: 'prev'; available: boolean[] } | { type: 'skip' } | { type: 'done' };

export const TOUR_DONE_KEY = 'shardx_tour_done';

export const FILES_TOUR: TourStep[] = [
  { selector: '#dropzone', title: 'Drop a file here', body: 'Anything up to 100 GB. It streams in 5 MB chunks and you can pause, resume or cancel.', placement: 'bottom' },
  { selector: '#strategy', title: 'Pick a distribution strategy', body: 'Balanced is a safe default. Preview the plan to see where each shard will land before you finalize.', placement: 'top' },
  { selector: '#nav-files', title: 'Health & shard map', body: 'Once processing completes, open a file to see its health score, a 3D map of every shard and the audit timeline.', placement: 'right' },
  { selector: '#downloadKey, #tour-help', title: 'Download your key file', body: 'When processing completes, download your .2xpfm.key from the upload card. It holds the seed and chunk map — the server never has it.', placement: 'top' },
];

export const tourReducer = (s: TourState, a: TourAction): TourState => {
  switch (a.type) {
    case 'start': return { open: true, index: 0 };
    case 'skip': case 'done': return { ...s, open: false };
    case 'next': {
      let i = s.index + 1;
      while (i < a.available.length && !a.available[i]) i++;
      return i >= a.available.length ? { ...s, open: false } : { open: true, index: i };
    }
    case 'prev': {
      let i = s.index - 1;
      while (i >= 0 && !a.available[i]) i--;
      return i < 0 ? s : { open: true, index: i };
    }
  }
};
