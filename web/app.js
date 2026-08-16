"use strict";

// --- tiny helpers -----------------------------------------------------------

const $ = (sel) => document.querySelector(sel);

function el(tag, cls, text) {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  if (text !== undefined) e.textContent = text;
  return e;
}

function fmtBytes(n) {
  if (n === 0) return "0 B";
  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  const i = Math.min(units.length - 1, Math.floor(Math.log2(n) / 10));
  return (n / 2 ** (10 * i)).toFixed(i === 0 ? 0 : 1) + " " + units[i];
}

function fmtTime(iso) {
  const d = new Date(iso);
  return d.toLocaleString(undefined, { dateStyle: "short", timeStyle: "short" });
}

function toast(msg, isError = true) {
  const t = $("#toast");
  t.textContent = msg;
  t.style.background = isError ? "#b3261e" : "#2e7d32";
  t.hidden = false;
  clearTimeout(toast._timer);
  toast._timer = setTimeout(() => (t.hidden = true), 5000);
}

async function api(method, url, body) {
  const opts = { method, headers: {} };
  if (body !== undefined) {
    opts.headers["Content-Type"] = "application/json";
    opts.body = JSON.stringify(body);
  }
  const resp = await fetch(url, opts);
  const text = await resp.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch { data = null; }
  if (!resp.ok) {
    throw new Error((data && (data.error || data.message)) || `${resp.status} ${resp.statusText}`);
  }
  return data;
}

// --- theme: system default + persisted manual override ----------------------
// Same method as globnotes' theme engine: stored id ("system"|"light"|"dark"),
// "system" resolves via prefers-color-scheme, OS changes re-apply live while
// in system mode. (The anti-FOUC inline script in index.html mirrors this.)

const THEME_KEY = "localsend-nas-theme";
const THEME_ORDER = ["system", "light", "dark"];
const THEME_ICONS = { light: "☀️", dark: "🌙" };

// "system" mode shows the actual device class: phone on coarse pointers
// (touch), desktop otherwise.
function systemIcon() {
  return window.matchMedia("(pointer: coarse)").matches ? "📱" : "🖥";
}

function themeIcon(id) {
  return id === "system" ? systemIcon() : THEME_ICONS[id];
}

function currentThemeId() {
  return localStorage.getItem(THEME_KEY) || "system";
}

function themeIsDark(id) {
  return id === "dark" || (id === "system" && window.matchMedia("(prefers-color-scheme: dark)").matches);
}

function applyTheme(id) {
  const dark = themeIsDark(id);
  const root = document.documentElement;
  root.classList.toggle("dark", dark);            // html.dark drives CSS vars
  root.style.colorScheme = dark ? "dark" : "light"; // native widgets/scrollbars
  const meta = document.querySelector('meta[name="theme-color"]');
  if (meta) meta.content = dark ? "#161a22" : "#f6f7f9";
  const btn = $("#themeToggle");
  if (btn) {
    btn.textContent = themeIcon(id);
    btn.title = `Theme: ${id} (click to change)`;
  }
}

function initTheme() {
  applyTheme(currentThemeId());
  window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", () => {
    if (currentThemeId() === "system") applyTheme("system");
  });
  $("#themeToggle").addEventListener("click", () => {
    const next = THEME_ORDER[(THEME_ORDER.indexOf(currentThemeId()) + 1) % THEME_ORDER.length];
    localStorage.setItem(THEME_KEY, next);
    applyTheme(next);
  });
}

// --- state ------------------------------------------------------------------

const state = {
  shares: [],
  share: "",
  path: "",           // current dir, relative POSIX
  basket: new Map(),  // key "share:rel" -> {share, rel, name, size, isDir}
  devices: [],
  jobs: new Map(),    // jobId -> job
  events: new Map(),  // jobId -> EventSource
};

// --- tabs -------------------------------------------------------------------

$("#tabs").addEventListener("click", (e) => {
  const btn = e.target.closest("button[data-tab]");
  if (!btn) return;
  document.querySelectorAll("#tabs button").forEach((b) => b.classList.toggle("active", b === btn));
  document.querySelectorAll("main > section").forEach((s) => (s.hidden = s.id !== "tab-" + btn.dataset.tab));
});

// --- files / basket ---------------------------------------------------------

async function loadShares() {
  state.shares = await api("GET", "/api/shares");
  const sel = $("#shareSelect");
  sel.replaceChildren(...state.shares.map((s) => {
    const o = el("option", "", s.name);
    o.value = s.name;
    return o;
  }));
  if (state.shares.length && !state.share) state.share = state.shares[0].name;
  sel.value = state.share;
  await navigate("");
}

$("#shareSelect").addEventListener("change", async (e) => {
  state.share = e.target.value;
  await navigate("");
});

function joinRel(base, name) { return base ? base + "/" + name : name; }

async function navigate(path) {
  state.path = path;
  const entries = await api("GET", `/api/list?share=${encodeURIComponent(state.share)}&path=${encodeURIComponent(path)}`);
  renderBreadcrumb();
  renderList(entries || []);
}

function renderBreadcrumb() {
  const bc = $("#breadcrumb");
  bc.replaceChildren();
  const root = el("a", "crumb", "/");
  root.addEventListener("click", () => navigate(""));
  bc.append(root);
  let acc = "";
  for (const part of state.path.split("/").filter(Boolean)) {
    acc = joinRel(acc, part);
    const link = el("a", "crumb", part);
    const target = acc;
    link.addEventListener("click", () => navigate(target));
    bc.append(el("span", "sep", " / "), link);
  }
}

function renderList(entries) {
  const tbody = $("#fileList tbody");
  tbody.replaceChildren();
  if (state.path) {
    const tr = el("tr", "dirrow");
    const name = el("td", "name", "↩ ..");
    name.colSpan = 4;
    name.addEventListener("click", () => navigate(state.path.split("/").slice(0, -1).join("/")));
    tr.append(el("td", "check"), name);
    tbody.append(tr);
  }
  for (const e of entries) {
    const tr = el("tr", e.isDir ? "dirrow" : "");
    const key = `${state.share}:${e.rel}`;
    const cb = el("input");
    cb.type = "checkbox";
    cb.checked = state.basket.has(key);
    cb.addEventListener("change", () => toggleBasket(key, e, cb.checked));
    const checkTd = el("td", "check");
    checkTd.append(cb);
    const nameTd = el("td", "name", (e.isDir ? "📁 " : "📄 ") + e.name);
    if (e.isDir) {
      nameTd.style.cursor = "pointer";
      nameTd.addEventListener("click", () => navigate(e.rel));
    }
    tr.append(checkTd, nameTd,
      el("td", "num", e.isDir ? "—" : fmtBytes(e.size)),
      el("td", "", fmtTime(e.modTime)));
    tbody.append(tr);
  }
}

function toggleBasket(key, entry, on) {
  if (on) {
    state.basket.set(key, { share: state.share, rel: entry.rel, name: entry.name, size: entry.size, isDir: entry.isDir });
  } else {
    state.basket.delete(key);
  }
  renderBasketBar();
}

function renderBasketBar() {
  const items = [...state.basket.values()];
  const total = items.reduce((a, i) => a + (i.isDir ? 0 : i.size), 0);
  $("#basketInfo").textContent = items.length
    ? `Basket: ${items.length} item${items.length > 1 ? "s" : ""} (${fmtBytes(total)}${items.some((i) => i.isDir) ? "+" : ""})`
    : "Basket empty";
  $("#chooseDevice").disabled = items.length === 0;
  $("#clearBasket").disabled = items.length === 0;
}

$("#clearBasket").addEventListener("click", () => {
  state.basket.clear();
  renderBasketBar();
  navigate(state.path); // refresh checkboxes
});

$("#chooseDevice").addEventListener("click", () => {
  document.querySelector('#tabs button[data-tab="devices"]').click();
});

// --- devices ----------------------------------------------------------------

const TYPE_ICONS = { mobile: "📱", desktop: "💻", web: "🌐", headless: "🖥", server: "🗄" };

async function loadDevices() {
  state.devices = await api("GET", "/api/devices");
  renderDevices();
}

function renderDevices() {
  const ul = $("#deviceList");
  ul.replaceChildren();
  if (!state.devices.length) {
    ul.append(el("li", "empty", "No devices yet — waiting for announcements, or add one by IP above."));
    return;
  }
  const basketEmpty = state.basket.size === 0;
  for (const d of state.devices) {
    const li = el("li", "card");
    const head = el("div", "card-head");
    head.append(
      el("span", "icon", TYPE_ICONS[d.deviceType] || "📟"),
      el("strong", "", d.alias),
      el("span", "muted", `${d.ip}:${d.port}`),
    );
    if (d.manual) head.append(el("span", "badge", "manual"));
    li.append(head);
    const meta = el("div", "muted small",
      `${d.deviceModel || "?"} · seen ${fmtTime(d.lastSeen)} · ${d.fingerprint.slice(0, 12)}…`);
    li.append(meta);
    const actions = el("div", "actions");
    const send = el("button", "", "Send basket");
    send.disabled = basketEmpty;
    send.addEventListener("click", () => sendBasket(d));
    actions.append(send);
    if (d.manual) {
      const del = el("button", "secondary", "Forget");
      del.addEventListener("click", async () => {
        try {
          await api("DELETE", `/api/devices/${d.fingerprint}`);
          await loadDevices();
        } catch (err) { toast(err.message); }
      });
      actions.append(del);
    }
    li.append(actions);
    ul.append(li);
  }
}

$("#addDevice").addEventListener("click", async () => {
  const input = $("#addrInput");
  try {
    await api("POST", "/api/devices", { address: input.value.trim() });
    input.value = "";
    await loadDevices();
  } catch (err) { toast(err.message); }
});
$("#addrInput").addEventListener("keydown", (e) => { if (e.key === "Enter") $("#addDevice").click(); });
$("#refreshDevices").addEventListener("click", () => loadDevices().catch((e) => toast(e.message)));

async function sendBasket(d) {
  const items = [...state.basket.values()].map(({ share, rel }) => ({ share, rel }));
  try {
    const { jobId } = await api("POST", "/api/send", { target: d.fingerprint, items });
    state.basket.clear();
    renderBasketBar();
    toast(`Sending to ${d.alias}…`, false);
    document.querySelector('#tabs button[data-tab="transfers"]').click();
    await refreshTransfers();
    watchJob(jobId);
  } catch (err) { toast(err.message); }
}

// --- transfers --------------------------------------------------------------

const ACTIVE = new Set(["queued", "preparing", "awaiting-accept", "sending"]);

async function refreshTransfers() {
  const jobs = await api("GET", "/api/transfers");
  state.jobs = new Map((jobs || []).map((j) => [j.id, j]));
  renderTransfers();
  for (const j of state.jobs.values()) if (ACTIVE.has(j.state)) watchJob(j.id);
}

function watchJob(id) {
  if (state.events.has(id)) return;
  const es = new EventSource(`/api/transfers/${id}/events`);
  es.onmessage = (ev) => {
    const job = JSON.parse(ev.data);
    state.jobs.set(id, job);
    renderTransfers();
    if (!ACTIVE.has(job.state)) { es.close(); state.events.delete(id); }
  };
  es.onerror = () => { es.close(); state.events.delete(id); };
  state.events.set(id, es);
}

function renderTransfers() {
  const ul = $("#transferList");
  ul.replaceChildren();
  const jobs = [...state.jobs.values()];
  const activeCount = jobs.filter((j) => ACTIVE.has(j.state)).length;
  $("#transferBadge").textContent = activeCount ? ` (${activeCount})` : "";
  if (!jobs.length) {
    ul.append(el("li", "empty", "No transfers yet."));
    return;
  }
  for (const j of jobs) {
    const li = el("li", "card");
    const head = el("div", "card-head");
    head.append(el("strong", "", `→ ${j.alias}`), el("span", `state state-${j.state}`, j.state));
    li.append(head);

    const totalBar = el("div", "bar");
    const totalFill = el("div", "fill");
    totalFill.style.width = j.total ? `${Math.min(100, (100 * j.sent) / j.total).toFixed(1)}%` : "0%";
    totalBar.append(totalFill);
    li.append(totalBar, el("div", "muted small", `${fmtBytes(j.sent)} / ${fmtBytes(j.total)} · ${j.files.length} file(s)`));

    for (const f of j.files) {
      const row = el("div", "file-row");
      row.append(el("span", "name", f.name), el("span", "muted small", `${fmtBytes(f.sent)}/${fmtBytes(f.size)}`));
      li.append(row);
    }
    if (j.error) li.append(el("div", "error small", j.error));
    if (ACTIVE.has(j.state)) {
      const cancel = el("button", "secondary", "Cancel");
      cancel.addEventListener("click", async () => {
        try { await api("POST", `/api/transfers/${j.id}/cancel`); } catch (err) { toast(err.message); }
      });
      const actionsDiv = el("div", "actions");
      actionsDiv.append(cancel);
      li.append(actionsDiv);
    }
    ul.append(li);
  }
}

// --- boot -------------------------------------------------------------------

(async function boot() {
  initTheme();
  renderBasketBar();
  try {
    await loadShares();
    await Promise.all([loadDevices(), refreshTransfers()]);
  } catch (err) { toast(err.message); }
  setInterval(() => { if (!$("#tab-devices").hidden) loadDevices().catch(() => {}); }, 5000);
  setInterval(() => { if (!$("#tab-transfers").hidden) refreshTransfers().catch(() => {}); }, 3000);
})();
