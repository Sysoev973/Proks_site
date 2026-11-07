// Поиск по документации: лениво грузит search-index.json своей языковой версии
// при первом фокусе, ищет подстроку с весами (заголовок > секция > текст),
// подсвечивает совпадение через <mark>. Без внешних зависимостей.
document.addEventListener("DOMContentLoaded", () => {
  const input = document.getElementById("site-search");
  if (!input) return;
  const list = document.getElementById("search-results");
  const noResults = input.dataset.noResults || "No results";

  let index = null;   // записи из search-index.json
  let loading = null; // промис загрузки, чтобы не запрашивать файл дважды
  let items = [];     // ссылки в текущей выдаче
  let active = -1;    // выбранная стрелками ссылка
  let timer = null;

  const load = () => {
    if (!loading) {
      loading = fetch(input.dataset.index)
        .then((r) => (r.ok ? r.json() : []))
        .then((data) => { index = data; })
        .catch(() => { index = []; });
    }
    return loading;
  };
  input.addEventListener("focus", load);

  const esc = (s) => s.replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));

  const mark = (s, q) => {
    const i = s.toLowerCase().indexOf(q);
    if (i < 0) return esc(s);
    return esc(s.slice(0, i)) + "<mark>" + esc(s.slice(i, i + q.length)) + "</mark>" + esc(s.slice(i + q.length));
  };

  // Кусок текста вокруг первого совпадения.
  const snippet = (text, q) => {
    const i = text.toLowerCase().indexOf(q);
    if (i < 0) return esc(text.slice(0, 120));
    const from = Math.max(0, i - 40);
    const to = Math.min(text.length, i + q.length + 80);
    return (from > 0 ? "…" : "") + mark(text.slice(from, to), q) + (to < text.length ? "…" : "");
  };

  const search = (q) => {
    const scored = [];
    for (const e of index) {
      let score = 0;
      if (e.title.toLowerCase().includes(q)) score = 3;
      else if (e.section.toLowerCase().includes(q)) score = 2;
      else if (e.text.toLowerCase().includes(q)) score = 1;
      if (score) scored.push([score, e]);
    }
    scored.sort((a, b) => b[0] - a[0]);
    return scored.slice(0, 8).map(([, e]) => e);
  };

  const close = () => {
    list.hidden = true;
    items = [];
    active = -1;
  };

  const render = () => {
    const q = input.value.trim().toLowerCase();
    if (q.length < 2 || index === null) { close(); return; }

    const found = search(q);
    list.innerHTML = "";
    if (!found.length) {
      const li = document.createElement("li");
      li.className = "search-empty";
      li.textContent = noResults;
      list.appendChild(li);
    } else {
      for (const e of found) {
        const li = document.createElement("li");
        const a = document.createElement("a");
        a.href = e.url;
        a.innerHTML =
          '<span class="search-title">' + mark(e.title, q) + "</span>" +
          (e.section ? '<span class="search-section">' + mark(e.section, q) + "</span>" : "") +
          '<span class="search-text">' + snippet(e.text, q) + "</span>";
        li.appendChild(a);
        list.appendChild(li);
      }
    }
    items = [...list.querySelectorAll("a")];
    active = -1;
    list.hidden = false;
  };

  input.addEventListener("input", () => {
    clearTimeout(timer);
    timer = setTimeout(async () => { await load(); render(); }, 180);
  });

  input.addEventListener("keydown", (ev) => {
    if (ev.key === "Escape") { close(); input.blur(); return; }
    if (list.hidden || !items.length) return;
    if (ev.key === "ArrowDown" || ev.key === "ArrowUp") {
      ev.preventDefault();
      active = (active + (ev.key === "ArrowDown" ? 1 : -1) + items.length) % items.length;
      items.forEach((a, i) => a.classList.toggle("active", i === active));
      items[active].scrollIntoView({ block: "nearest" });
    } else if (ev.key === "Enter" && active >= 0) {
      ev.preventDefault();
      items[active].click();
    }
  });

  document.addEventListener("click", (ev) => {
    if (!list.contains(ev.target) && ev.target !== input) close();
  });
});
