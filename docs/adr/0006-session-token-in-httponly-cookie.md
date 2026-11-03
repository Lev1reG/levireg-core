# Session token in an HttpOnly cookie, not a Bearer header

ADR 0003 assumed the console would be TanStack Start, keeping the token in its own server-side cookie and calling the API only from server functions. The console turned out to be a client-only SPA (Vite + TanStack Router) served from `console.levireg.dev`, so the token must reach the browser. Its opaque Session token now travels only in a host-only `__Host-session` cookie set by `api.levireg.dev` (`HttpOnly; Secure; SameSite=Strict; Path=/; Max-Age` = Session lifetime). Login returns the User, never the token, and the `Authorization: Bearer` scheme is removed. We chose this over a Bearer token in `localStorage` (readable by any XSS) or in memory (lost on every refresh), and over a backend-for-frontend (a second service to run for one SPA). Opaque database Sessions from ADR 0003 are unchanged.

## Consequences

- Console and API are same-site, so `SameSite=Strict` cookies are sent on the console's requests. The API answers CORS with credentials for origins in `ALLOWED_ORIGINS` (production: `https://console.levireg.dev`).
- `SameSite` does not stop requests from sibling subdomains. Every non-GET request whose `Origin` header is present and is neither the API's own origin nor in `ALLOWED_ORIGINS` gets 403. Requests without `Origin` (curl, server-to-server) are allowed, because browsers always send it on POST.
- `personal-website-static` (Astro) fetches public content at build time from a server, so it needs neither CORS nor a Session.
- Local dev proxies `/api` through the Vite dev server, making it same-origin; browsers accept `Secure` and `__Host-` cookies on `localhost`.
- Non-browser clients sign in with a cookie jar (`curl -c jar -b jar`).

Revisit if a client that cannot hold cookies (mobile app, third-party integration) needs to sign in.
