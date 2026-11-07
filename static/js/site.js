// Кнопка «копировать» у блоков кода. Тексты кнопки — из data-атрибутов <body>,
// их проставляет шаблон из словаря, поэтому интерфейс остаётся двуязычным.
document.addEventListener("DOMContentLoaded", () => {
  const label = document.body.dataset.copy || "Copy";
  const done = document.body.dataset.copied || "Copied";

  document.querySelectorAll("pre > code").forEach((code) => {
    const pre = code.parentElement;
    const wrap = document.createElement("div");
    wrap.className = "code-block";
    pre.parentNode.insertBefore(wrap, pre);
    wrap.appendChild(pre);

    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "code-copy mono-caps";
    btn.textContent = label;
    btn.addEventListener("click", async () => {
      try {
        await navigator.clipboard.writeText(code.innerText);
        btn.textContent = done;
        btn.classList.add("copied");
        setTimeout(() => {
          btn.textContent = label;
          btn.classList.remove("copied");
        }, 1500);
      } catch {
        // Буфер недоступен (http без localhost, старый браузер) — просто не копируем.
      }
    });
    wrap.appendChild(btn);
  });
});
