const app = document.getElementById("app");

function api(path) {
  return fetch(path).then(async (r) => {
    const body = await r.json().catch(() => null);
    if (!r.ok) {
      throw new Error((body && body.error) || r.statusText);
    }
    return body;
  });
}

function el(tag, attrs, children) {
  const e = document.createElement(tag);
  for (const k in attrs || {}) {
    if (k === "text") e.textContent = attrs[k];
    else if (k === "html") e.innerHTML = attrs[k];
    else e.setAttribute(k, attrs[k]);
  }
  (children || []).forEach((c) => e.appendChild(c));
  return e;
}

function escapeHTML(s) {
  return s.replace(/[&<>"']/g, (c) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
  }[c]));
}

function highlightSnippet(s) {
  return escapeHTML(s).replace(/\[(.*?)\]/g, "<mark>$1</mark>");
}

function formatBytes(n) {
  if (!n) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  let v = n, i = 0;
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++; }
  return v.toFixed(1) + " " + units[i];
}

function showError(err) {
  app.innerHTML = "";
  app.appendChild(el("p", { class: "error", text: "error: " + err.message }));
}

function rowCard(row, showSnippet) {
  const star = row.is_starred ? "★ " : "";
  const card = el("div", { class: "row-card" });
  card.appendChild(el("a", { href: "#/get/" + row.id, class: "row-title", text: star + (row.title || "(untitled)") }));
  card.appendChild(el("div", {
    class: "row-meta",
    text: `${row.local_date}  ·  ${row.duration_min.toFixed(0)}m  ·  ${row.category}`,
  }));
  if (row.speakers && row.speakers.length) {
    card.appendChild(el("div", { class: "row-speakers", text: "speakers: " + row.speakers.join(", ") }));
  }
  if (showSnippet && row.snippet) {
    card.appendChild(el("div", { class: "row-snippet", html: highlightSnippet(row.snippet) }));
  }
  return card;
}

function renderResultsList(rows, showSnippet, emptyMsg) {
  app.innerHTML = "";
  if (!rows || rows.length === 0) {
    app.appendChild(el("p", { class: "empty", text: emptyMsg || "no results" }));
    return;
  }
  const list = el("div", { class: "row-list" });
  rows.forEach((r) => list.appendChild(rowCard(r, showSnippet)));
  app.appendChild(list);
}

function viewRecent() {
  app.innerHTML = "";
  api("/api/recent?n=25")
    .then((data) => renderResultsList(data.results, false, "no recordings yet"))
    .catch(showError);
}

function viewSearch(query) {
  app.innerHTML = "";
  const form = el("form", { class: "search-form" });
  const input = el("input", { type: "text", name: "q", placeholder: "search transcripts…" });
  input.value = query || "";
  form.appendChild(input);
  form.appendChild(el("button", { type: "submit", text: "Search" }));
  form.addEventListener("submit", (e) => {
    e.preventDefault();
    location.hash = "#/search/" + encodeURIComponent(input.value.trim());
  });
  app.appendChild(form);

  const results = el("div", { class: "results" });
  app.appendChild(results);
  if (!query) return;

  api("/api/search?q=" + encodeURIComponent(query) + "&limit=30")
    .then((data) => {
      results.innerHTML = "";
      if (!data.results || data.results.length === 0) {
        results.appendChild(el("p", { class: "empty", text: "no matches for \"" + query + "\"" }));
        return;
      }
      data.results.forEach((r) => results.appendChild(rowCard(r, true)));
    })
    .catch((err) => {
      results.innerHTML = "";
      results.appendChild(el("p", { class: "error", text: "error: " + err.message }));
    });
}

function viewGet(id) {
  app.innerHTML = "";
  api("/api/get/" + encodeURIComponent(id) + "?full=true")
    .then((r) => {
      app.appendChild(el("a", { href: "#/recent", class: "back-link", text: "← back" }));
      app.appendChild(el("h2", { text: r.title || "(untitled)" }));
      app.appendChild(el("div", {
        class: "row-meta",
        text: `${r.local_date}  ·  ${r.start_utc} – ${r.end_utc}  ·  ${r.duration_min.toFixed(0)}m  ·  ${r.category}`,
      }));
      if (r.speakers && r.speakers.length) {
        app.appendChild(el("div", { class: "row-speakers", text: "speakers: " + r.speakers.join(", ") }));
      }
      app.appendChild(el("pre", { class: "transcript", text: r.transcript_md || "(no transcript)" }));
    })
    .catch(showError);
}

function statTile(label, value) {
  return el("div", { class: "stat-tile" }, [
    el("div", { class: "stat-value", text: String(value) }),
    el("div", { class: "stat-label", text: label }),
  ]);
}

function viewStats() {
  app.innerHTML = "";
  api("/api/stats")
    .then((st) => {
      app.appendChild(el("h2", { text: "Catalog stats" }));

      const grid = el("div", { class: "stats-grid" });
      grid.appendChild(statTile("Total conversations", st.total));
      grid.appendChild(statTile("Date range", (st.first_date || "–") + " → " + (st.last_date || "–")));
      grid.appendChild(statTile("Total hours", st.total_hours.toFixed(1)));
      grid.appendChild(statTile("Database size", formatBytes(st.db_bytes)));
      grid.appendChild(statTile("Last ingest", st.last_ingest || "never"));
      app.appendChild(grid);

      app.appendChild(el("h3", { text: "By category" }));
      const catList = el("ul", { class: "plain-list" });
      Object.keys(st.by_category || {}).sort().forEach((cat) => {
        catList.appendChild(el("li", { text: `${cat}: ${st.by_category[cat]}` }));
      });
      app.appendChild(catList);

      app.appendChild(el("h3", { text: "Per month" }));
      const table = el("table", { class: "month-table" });
      (st.per_month || []).forEach((mc) => {
        table.appendChild(el("tr", {}, [
          el("td", { text: mc.month }),
          el("td", { text: String(mc.count) }),
        ]));
      });
      app.appendChild(table);

      if (st.empty_days && st.empty_days.length) {
        app.appendChild(el("h3", { text: `Empty days (${st.empty_days.length})` }));
        app.appendChild(el("p", { class: "empty-days", text: st.empty_days.join(", ") }));
      }
    })
    .catch(showError);
}

function route() {
  const hash = location.hash.replace(/^#\/?/, "");
  const parts = hash.split("/").filter(Boolean);
  if (parts[0] === "search") {
    viewSearch(parts[1] ? decodeURIComponent(parts[1]) : "");
  } else if (parts[0] === "get" && parts[1]) {
    viewGet(decodeURIComponent(parts[1]));
  } else if (parts[0] === "stats") {
    viewStats();
  } else {
    viewRecent();
  }
}

window.addEventListener("hashchange", route);
window.addEventListener("DOMContentLoaded", route);
