# Changelog

## 2025-12-31

- Added display name support across chat messages (REST + WS) and UI.
- Secured uploads with owner/team metadata, header-based access, and token-free URLs; frontend fetches protected images.
- Implemented message deletion (author-only) with UI actions and deletion broadcast.
- Enforced membership: removed users are notified, redirected, and prevented from reconnecting; added styled removal modal.
- Hardened WebSocket reconnect logic and membership checks; faster reconnect backoff.
- Added global server connectivity banner via StatusContext and interceptor; themed alert when backend is unreachable.
- Optimized sidebar polling (change detection, 2s interval) and reduced flicker.
- Added paste preview and sticky input tweaks; updated UI labels to use display names.
- Added favicon logo for browser tab.
