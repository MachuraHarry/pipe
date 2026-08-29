// Service Worker: "worldcraft" — PWA-Offline-Fallback + sauberes Update-Handling.
// CACHE_VERSION wird per Build-Script (tools/bump_cache.py) ersetzt.
const CACHE_VERSION = "4f27185b65";
const CACHE_NAME = "worldcraft-v" + CACHE_VERSION;

const STATIC_ASSETS = [
  "/",
  "/index.html",
  "/manifest.json",
  "/css/minimap.css",
  "/css/style.css",
  "/js/api.js",
  "/js/app.js",
  "/js/audio.js",
  "/js/i18n.js",
  "/js/minimap.js",
  "/js/ui.js",
  "/img/icon-192.png",
  "/img/icon-512.png",
  "/audio/dialog.wav",
  "/audio/hit.wav",
  "/audio/pickup.wav",
  "/audio/step.wav",
  "/audio/victory.wav",
];

const ASSET_URLS = new Set(STATIC_ASSETS.map(a => a.replace(/^\//, "")));

const isAsset = (pathname) => {
  return pathname === "/" || pathname === "/index.html" || ASSET_URLS.has(pathname.replace(/^\//, ""));
};

self.addEventListener("install", (e) => {
  self.skipWaiting();
  e.waitUntil(
    caches.open(CACHE_NAME).then((c) => c.addAll(STATIC_ASSETS)).catch(() => {})
  );
});

self.addEventListener("activate", (e) => {
  e.waitUntil(
    Promise.all([
      self.clients.claim(),
      caches.keys().then((keys) =>
        Promise.all(keys.filter((k) => k !== CACHE_NAME).map((k) => caches.delete(k)))
      )
    ])
  );
});

self.addEventListener("fetch", (e) => {
  const url = new URL(e.request.url);
  // Nur GET cachen.
  if (e.request.method !== "GET") return;

  // API: Netzwerk-only (mit Offline-Ersatz für saubere Fehlermeldung).
  if (url.pathname.startsWith("/api/")) {
    e.respondWith(
      fetch(e.request).catch(() =>
        new Response(
          JSON.stringify({ error: { code: "offline", message: "Offline" } }),
          { headers: { "Content-Type": "application/json" }, status: 503 }
        )
      )
    );
    return;
  }

  if (!isAsset(url.pathname)) return; // alles andere (z. B. dynamisch) lassen wir durch

  const isNavigation = () => {
    // index.html ist die SPA-Shell: Netzwerk zuerst, Cache-Fallback.
    return url.pathname === "/" || url.pathname === "/index.html";
  };

  if (isNavigation()) {
    e.respondWith(
      fetch(e.request)
        .then((r) => {
          const copy = r.clone();
          if (r.ok) caches.open(CACHE_NAME).then((c) => c.put("/index.html", copy));
          return r;
        })
        .catch(() => caches.match("/index.html"))
    );
    return;
  }

  // Statische Assets: Cache-first mit Background-Update (stale-while-revalidate).
  e.respondWith(
    caches.match(e.request, { ignoreSearch: true }).then((cached) => {
      const networkUpdate = fetch(e.request)
        .then((r) => {
          if (r.ok) {
            caches.open(CACHE_NAME).then((c) => c.put(e.request, r.clone()));
          }
          return r;
        })
        .catch(() => {});
      // Sofortige Antwort aus Cache (oder Netzwerk, falls frisch); Update läuft.
      return cached || networkUpdate;
    })
  );
});

