/**
 * Service Worker — кеширование для работы на медленных соединениях (3G, 4G).
 * Стратегия: Cache-First для статики, Network-First для API.
 * Версия кеша обновляется при каждом деплое (CACHE_VERSION меняется автоматически).
 */

const CACHE_VERSION = 'v1';
const STATIC_CACHE  = 'aqi-static-'  + CACHE_VERSION;
const DATA_CACHE    = 'aqi-data-'    + CACHE_VERSION;

// Статические файлы которые кешируем при установке (критически важны для offline)
const PRECACHE_URLS = [
  '/',
  '/manifest.webmanifest',
];

// ── Установка: предзагрузка критических ресурсов ──────────────────────────────
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(STATIC_CACHE).then((cache) =>
      cache.addAll(PRECACHE_URLS).catch(() => {})
    )
  );
  // Немедленная активация без ожидания закрытия старых вкладок
  self.skipWaiting();
});

// ── Активация: удаление устаревших кешей ─────────────────────────────────────
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(
        keys
          .filter((k) => k !== STATIC_CACHE && k !== DATA_CACHE)
          .map((k) => caches.delete(k))
      )
    )
  );
  self.clients.claim();
});

// ── Fetch: стратегия в зависимости от типа запроса ───────────────────────────
self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);

  // Не кешируем запросы к другим доменам
  if (url.origin !== self.location.origin) return;

  // API-запросы: Network-First (свежие данные важнее, но при ошибке — кеш)
  if (url.pathname.startsWith('/api/')) {
    event.respondWith(networkFirst(request, DATA_CACHE));
    return;
  }

  // Статика (JS, CSS, изображения): Cache-First (быстро на 3G)
  if (
    request.destination === 'script' ||
    request.destination === 'style'  ||
    request.destination === 'image'  ||
    request.destination === 'font'
  ) {
    event.respondWith(cacheFirst(request, STATIC_CACHE));
    return;
  }

  // HTML-навигация: Network-First, fallback на root
  if (request.mode === 'navigate') {
    event.respondWith(
      fetch(request).catch(() =>
        caches.match('/').then((r) => r ?? new Response('Нет сети', { status: 503 }))
      )
    );
    return;
  }

  // Всё остальное: попробовать сеть, иначе кеш
  event.respondWith(networkFirst(request, STATIC_CACHE));
});

// ── Стратегия Cache-First (статика, шрифты) ───────────────────────────────────
async function cacheFirst(request, cacheName) {
  const cached = await caches.match(request);
  if (cached) return cached;
  try {
    const response = await fetch(request);
    if (response.ok) {
      const cache = await caches.open(cacheName);
      cache.put(request, response.clone());
    }
    return response;
  } catch {
    return new Response('', { status: 503 });
  }
}

// ── Стратегия Network-First (API, динамические данные) ────────────────────────
async function networkFirst(request, cacheName) {
  try {
    const response = await fetch(request);
    if (response.ok && request.method === 'GET') {
      const cache = await caches.open(cacheName);
      cache.put(request, response.clone());
    }
    return response;
  } catch {
    const cached = await caches.match(request);
    if (cached) return cached;
    return new Response(JSON.stringify({ error: 'Нет соединения с сервером' }), {
      status: 503,
      headers: { 'Content-Type': 'application/json' },
    });
  }
}
