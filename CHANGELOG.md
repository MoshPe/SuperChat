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

## 2026-01-05

- Added dark theme with toggle and global styling adjustments (chat, sidebar, dashboard, auth screens).
- Introduced team settings modal (edit name/description/avatar, leave/delete), avatar upload/preview/removal with protected fetches.
- Surfaced team avatars across chat header, sidebar, dashboard; cleaned cache on updates/removals.
- Refined chat layout: contained scrolling with themed scrollbar, input docked to bottom.
- Fixed dashboard stats: larger message window, active teams based on online presence; improved team hover/hover states in dark mode.
