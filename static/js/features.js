/* Визуальные фичи: счётчики в шапке и живая демо-симуляция кэша.
   Без зависимостей. Тексты приходят из шаблонов через data-атрибуты (словарь content/i18n). */
(() => {
  'use strict';
  const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches;
  const $ = (s, r = document) => r.querySelector(s);
  const $$ = (s, r = document) => Array.from(r.querySelectorAll(s));
  const el = (tag, cls, text) => {
    const e = document.createElement(tag);
    if (cls) e.className = cls;
    if (text != null) e.textContent = text;
    return e;
  };
  const svgEl = (tag, attrs = {}) => {
    const e = document.createElementNS('http://www.w3.org/2000/svg', tag);
    for (const k in attrs) e.setAttribute(k, attrs[k]);
    return e;
  };
  const once = (node, fn) => {
    if (!('IntersectionObserver' in window)) return fn();
    const o = new IntersectionObserver(es => {
      if (es.some(e => e.isIntersecting)) { o.disconnect(); fn(); }
    }, { threshold: 0.3 });
    o.observe(node);
  };
  const watch = (node, set) => {
    if (!('IntersectionObserver' in window)) return set(true);
    new IntersectionObserver(es => set(es[es.length - 1].isIntersecting), { threshold: 0.2 }).observe(node);
  };
  const lcg = seed => () => (seed = (Math.imul(seed, 1664525) + 1013904223) >>> 0) / 4294967296;

  /* ---------- Счётчики: числа в шапке набираются при появлении на экране ---------- */
  function counters() {
    if (reduce) return;
    $$('.stat-line b').forEach(b => {
      const m = b.textContent.match(/^(\d+(?:[.,]\d+)?)([\s\S]*)$/);
      if (!m) return;
      const sep = m[1].includes(',') ? ',' : '.';
      const dec = (m[1].split(/[.,]/)[1] || '').length;
      const to = parseFloat(m[1].replace(',', '.'));
      const show = v => (b.textContent = v.toFixed(dec).replace('.', sep) + m[2]);
      b.style.minWidth = b.offsetWidth + 'px';
      b.style.display = 'inline-block';
      show(0);
      once(b, () => {
        const t0 = performance.now(), D = 900;
        const tick = now => {
          const k = Math.min(1, (now - t0) / D);
          show(to * (1 - Math.pow(1 - k, 3)));
          if (k < 1) requestAnimationFrame(tick);
        };
        requestAnimationFrame(tick);
      });
    });
  }

  // <cache> Политики вытеснения для демо (упрощённые версии тех, что в Proks).
  class LRU {
    constructor(cap) { this.cap = cap; this.m = new Map(); }
    has(k) { if (!this.m.has(k)) return false; this.m.delete(k); this.m.set(k, 1); return true; }
    get(k) { return this.has(k); }
    add(k) {
      this.m.set(k, 1);
      if (this.m.size > this.cap) { const v = this.m.keys().next().value; this.m.delete(v); return v; }
    }
    victim() { return this.m.keys().next().value; }
  }
  class Sketch { // count-min sketch с периодическим делением счётчиков пополам
    constructor(width, limit) {
      this.w = width; this.limit = limit; this.n = 0;
      this.rows = [0x9e3779b1, 0x85ebca6b, 0xc2b2ae35, 0x27d4eb2f].map(s => ({ s, a: new Uint8Array(width) }));
    }
    idx(r, k) {
      let h = Math.imul(k + 1, r.s); h ^= h >>> 15; h = Math.imul(h, 0x2c1b3c6d); h ^= h >>> 12;
      return (h >>> 0) % this.w;
    }
    inc(k) {
      for (const r of this.rows) { const i = this.idx(r, k); if (r.a[i] < 255) r.a[i]++; }
      if (++this.n >= this.limit) { for (const r of this.rows) for (let i = 0; i < r.a.length; i++) r.a[i] >>= 1; this.n = 0; }
    }
    est(k) { let m = 255; for (const r of this.rows) m = Math.min(m, r.a[this.idx(r, k)]); return m; }
  }
  class TinyLFU { // окно LRU + основная область + допуск по частоте
    constructor(cap) {
      const w = Math.max(1, Math.round(cap * 0.05));
      this.win = new LRU(w); this.main = new LRU(Math.max(1, cap - w));
      this.f = new Sketch(Math.max(512, cap * 10), cap * 10);
    }
    get(k) { this.f.inc(k); return this.win.has(k) || this.main.has(k); }
    add(k) {
      const ev = this.win.add(k);
      if (ev === undefined) return;
      if (this.main.m.size < this.main.cap) { this.main.add(ev); return; }
      if (this.f.est(ev) > this.f.est(this.main.victim())) this.main.add(ev);
    }
  }
  function zipf(n, s, rnd) {
    const c = new Float64Array(n); let sum = 0;
    for (let i = 0; i < n; i++) { sum += 1 / Math.pow(i + 1, s); c[i] = sum; }
    return () => {
      const u = rnd() * sum; let lo = 0, hi = n - 1;
      while (lo < hi) { const m = (lo + hi) >> 1; if (c[m] < u) lo = m + 1; else hi = m; }
      return lo;
    };
  }
  // </cache>

  /* ---------- Демо кэша: LRU и TinyLFU на одном потоке запросов ---------- */
  function sim(root) {
    const L = root.dataset, N = 2000, WIN = 200, STRIP = 48;
    let cap = 100, running = !reduce, visible = true, scanLeft = 0, scanKey = N;
    let rnd, next, pol;

    const btnPlay = el('button', 'btn btn-ghost btn-sm'), btnScan = el('button', 'btn btn-accent btn-sm', L.scan),
      btnReset = el('button', 'btn btn-ghost btn-sm', L.reset);
    for (const b of [btnPlay, btnScan, btnReset]) b.type = 'button';
    const range = el('input'); range.type = 'range'; range.min = 50; range.max = 300; range.step = 10; range.value = cap;
    range.setAttribute('aria-label', L.size);
    const sizeOut = el('output', 'sim-size', String(cap));
    const ctl = el('div', 'sim-controls');
    const lab = el('label', 'sim-range'); lab.append(el('span', null, L.size), range, sizeOut);
    ctl.append(btnPlay, btnScan, btnReset, lab);

    const panels = [['lru', L.lru], ['tiny', L.tiny]].map(([id, title]) => {
      const box = el('div', 'sim-panel'), ratio = el('span', 'sim-ratio', '0%'), bar = el('i');
      const head = el('div', 'sim-head'); head.append(el('h3', 'sim-title', title), ratio);
      const track = el('div', 'sim-bar'); track.append(bar);
      const strip = el('div', 'sim-strip'); strip.setAttribute('aria-hidden', 'true');
      const cells = Array.from({ length: STRIP }, () => strip.appendChild(el('span', 'sim-cell')));
      box.append(head, track, strip);
      return { id, box, ratio, bar, cells };
    });
    const grid = el('div', 'sim-grid'); grid.append(...panels.map(p => p.box));
    const legend = el('p', 'sim-legend small text-body-secondary');
    legend.append(el('span', 'sim-key sim-key-hit'), L.hit, ' ', el('span', 'sim-key sim-key-miss'), L.miss);
    root.replaceChildren(ctl, grid, legend);

    function reset() {
      rnd = lcg(7); next = zipf(N, 1, rnd); scanLeft = 0; scanKey = N;
      pol = [new LRU(cap), new TinyLFU(cap)].map(c => ({ c, ring: new Uint8Array(WIN), pos: 0, n: 0, hits: 0, recent: [] }));
      paint();
    }
    function request() {
      const key = scanLeft > 0 ? (scanLeft--, scanKey++) : next();
      for (const p of pol) {
        const hit = p.c.get(key);
        if (!hit) p.c.add(key);
        if (p.n === WIN) p.hits -= p.ring[p.pos]; else p.n++;
        p.ring[p.pos] = hit ? 1 : 0; p.hits += p.ring[p.pos]; p.pos = (p.pos + 1) % WIN;
        p.recent.push(hit ? 1 : 0); if (p.recent.length > STRIP) p.recent.shift();
      }
    }
    function paint() {
      pol.forEach((p, i) => {
        const r = p.n ? p.hits / p.n : 0, v = panels[i];
        v.ratio.textContent = (r * 100).toFixed(1) + '%';
        v.bar.style.width = (r * 100) + '%';
        v.cells.forEach((c, j) => {
          const s = p.recent[j - (STRIP - p.recent.length)];
          c.className = 'sim-cell' + (s === 1 ? ' hit' : s === 0 ? ' miss' : '');
        });
      });
      btnPlay.textContent = running ? L.pause : L.play;
    }
    btnPlay.onclick = () => { running = !running; paint(); };
    btnScan.onclick = () => { scanLeft += 200; };
    btnReset.onclick = reset;
    range.oninput = () => { cap = +range.value; sizeOut.textContent = cap; reset(); };
    watch(root, v => (visible = v));
    setInterval(() => {
      if (!running || !visible) return;
      for (let i = scanLeft > 0 ? 10 : 2; i > 0; i--) request();
      paint();
    }, 100);
    reset();
  }

  /* ---------- Графики бенчмарков: перекрестие, подсказка, легенда, прорисовка линий ---------- */
  function charts() {
    $$('[data-series]').forEach(b => b.addEventListener('click', () => {
      const on = b.getAttribute('aria-pressed') !== 'true';
      b.setAttribute('aria-pressed', String(on));
      $$('.chart .' + b.dataset.series).forEach(n => n.classList.toggle('is-hidden', !on));
    }));
    $$('svg.chart').forEach(svg => {
      const dots = $$('.chart-dot', svg);
      dots.forEach(d => { const t = $('title', d); if (t) { d.dataset.tip = t.textContent; t.remove(); } });
      const wrap = el('div', 'chart-wrap'); svg.replaceWith(wrap); wrap.append(svg);
      const tip = el('div', 'chart-tip'); tip.hidden = true; wrap.append(tip);
      const vb = svg.viewBox.baseVal;
      const cross = svgEl('line', { class: 'chart-cross', y1: 12, y2: vb.height - 28 }); cross.style.display = 'none'; svg.append(cross);
      const xs = [...new Set(dots.map(d => +d.getAttribute('cx')))].sort((a, b) => a - b);
      const hide = () => { tip.hidden = true; cross.style.display = 'none'; dots.forEach(d => d.classList.remove('is-active')); };
      svg.addEventListener('pointermove', e => {
        if (!xs.length) return;
        const pt = svg.createSVGPoint(); pt.x = e.clientX; pt.y = e.clientY;
        const p = pt.matrixTransform(svg.getScreenCTM().inverse());
        const cx = xs.reduce((a, b) => (Math.abs(b - p.x) < Math.abs(a - p.x) ? b : a));
        const at = dots.filter(d => +d.getAttribute('cx') === cx && !d.classList.contains('is-hidden'));
        dots.forEach(d => d.classList.toggle('is-active', at.includes(d)));
        if (!at.length) return hide();
        cross.setAttribute('x1', cx); cross.setAttribute('x2', cx); cross.style.display = '';
        tip.replaceChildren(...at.map(d => el('div', null, d.dataset.tip)));
        tip.hidden = false;
        const left = cx * svg.getBoundingClientRect().width / vb.width, half = tip.offsetWidth / 2;
        tip.style.left = Math.min(Math.max(left, half), wrap.clientWidth - half) + 'px';
      });
      svg.addEventListener('pointerleave', hide);
      if (!reduce) once(svg, () => $$('.chart-line', svg).forEach(l => {
        try {
          const len = l.getTotalLength();
          l.style.strokeDasharray = len; l.style.strokeDashoffset = len;
          l.getBoundingClientRect(); // применить начальное состояние до анимации
          l.style.transition = 'stroke-dashoffset .9s ease'; l.style.strokeDashoffset = 0;
          setTimeout(() => { l.style.strokeDasharray = ''; l.style.transition = ''; }, 1000);
        } catch (_) { /* getTotalLength недоступен: линия остаётся как есть */ }
      }));
    });
  }

  const init = () => {
    counters();
    const s = $('[data-sim]'); if (s) sim(s);
    charts();
  };
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init); else init();
})();
