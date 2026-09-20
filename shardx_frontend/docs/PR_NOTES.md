# Frontend updates: OAuth linking + Upload flow alignment

This PR implements frontend behavior strictly according to the backend contract.

## Implemented

- OAuth drive linking popup flow:
  - Calls `GET /api/drive/link` and opens the returned `auth_url` in a popup.
  - Popup (`/oauth/finished` page) posts `{ type: 'oauth_finished', success: true, provider: 'google' }` back to the opener with `postMessage` and validated `targetOrigin`.
  - Parent (Profile page) validates `event.origin === FRONTEND_BASE_URL`, closes the popup, then refetches `GET /api/drive/accounts`.
  - Handles blocked popups (toast instructing to enable popups) and user closing without message (toast: canceled/interrupted).
  - Adds a "Refresh" button (re-fetches accounts on demand).

- Upload flow uses the documented endpoints:
  - `POST /api/files/upload/initiate` -> chunk upload -> `POST /api/files/upload/finalize` -> poll `GET /api/files/upload/status/<id>`.
  - On completion, shows a disabled "Download key file (TBD)" button. No server call is attempted because the endpoint is not implemented.

- Security and placeholders:
  - All protected calls include `Authorization: Bearer <JWT>`.
  - Origin validation enforced for `postMessage`.
  - Dummy features clearly labeled as `TBD` or `Coming Soon` and disabled.

## Tests

- Unit test (`src/lib/__tests__/oauth.test.ts`) verifying origin and payload shape validation for the popup `postMessage` handler.

## Manual test steps

1. Sign up or log in.
2. Go to Profile and click "Link Google Drive".
3. Grant consent in the popup.
4. After redirect to `/oauth/finished`, the popup posts back and auto-closes; Profile refreshes and shows the new account.
5. If a popup is blocked, a toast instructs to enable popups.
6. If the popup is closed without consent, a toast shows "Linking canceled or interrupted".
7. Upload a file on the Files page; observe upload progress and final processing status.
8. After completion, the "Download Key File" button remains disabled and labeled `TBD`.

## Missing backend endpoints (left as placeholders)

- Key download endpoint (e.g., `GET /api/files/upload/key/{session_id}`): frontend shows a disabled `TBD` button.
- Files list and file-level actions (`GET /api/files`, `GET /api/files/{id}`, `PATCH/DELETE`): frontend shows "Coming soon".

> Note: No backend code was modified in this PR. The `/oauth/finished` route is a frontend page that only posts a message to the opener; all OAuth code exchange and token storage remain server-side.
