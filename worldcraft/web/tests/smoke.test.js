// tests/smoke.test.js — Frontend-Smoke-Tests (Node, ohne Browser).
// Lädt die Vanilla-JS-Dateien in einen Node-Kontext (mit Minimal-Shims für
// localStorage/window) und testet die reinen Logikteile:
//   - i18n t() / setLang-Verhalten
//   - UI.esc / _escAttr (XSS-Schutz)
//   - Kommando-Router (erweitere / neues abenteuer / Freitext)
//   - api.pollJob (done/error/running-Verhalten)
//
// Aufruf:  node --test tests/   (im worldcraft/web-Verzeichnis)

const { test, before, describe } = require("node:test");
const assert = require("node:assert");
const path = require("node:path");
const fs = require("node:fs");

const JS = path.join(__dirname, "..", "js");
const read = (f) => fs.readFileSync(path.join(JS, f), "utf8");

function makeLocalStorage() {
  const store = new Map();
  return {
    getItem: (k) => (store.has(k) ? store.get(k) : null),
    setItem: (k, v) => store.set(k, String(v)),
    removeItem: (k) => store.delete(k),
    clear: () => store.clear(),
  };
}

// Minimal-DOM/global-Shims: reicht für das Laden der Dateien (kein Browser).
global.window = globalThis;
global.localStorage = makeLocalStorage();
global.navigator = { serviceWorker: { register: async () => ({}) } };
global.document = {
  documentElement: { lang: "de" },
  getElementById: () => null,
  querySelectorAll: () => [],
  addEventListener: () => {},
  createElement: () => ({ style: {}, appendChild() {}, addEventListener() {} }),
  body: { appendChild() {} },
};

// Reihenfolge wie in index.html — so bleiben die globalen Abhängigkeiten intakt.
// vm.runInThisContext hält Deklarationen aus dem globalen Kontext (kein Leak
// in diesen Test-Modul-Scope, kein doppeltes Deklarieren).
const vm = require("node:vm");
for (const f of ["i18n.js", "api.js", "ui.js", "minimap.js", "audio.js", "app.js"]) {
  vm.runInThisContext(read(f), { filename: f });
}

const { UI, t } = globalThis;
const app = globalThis._wcApp;
// class-Deklarationen hängen nicht am globalen Objekt; über app.api verfügbar.
globalThis.WorldcraftAPI = app.api.constructor ? app.api.constructor : globalThis.WorldcraftAPI;
const WorldcraftAPI = app.api.constructor;

describe("i18n", () => {
  test("t() liefert deutschen Text für de", () => {
    window._wcLang = "de";
    assert.strictEqual(t("thinking"), "Denke nach…");
  });

  test("t() liefert englischen Text für en", () => {
    window._wcLang = "en";
    assert.strictEqual(t("thinking"), "Thinking…");
  });

  test("t() fällt auf de zurück, wenn englischer Key fehlt", () => {
    window._wcLang = "en";
    assert.strictEqual(t("kein_solcher_key", "FB"), "FB");
  });

  test("t() liefert Fallback bei unbekanntem Key", () => {
    window._wcLang = "en";
    assert.strictEqual(t("nicht_vorhanden", "FB"), "FB");
  });
});

describe("UI.esc / XSS-Schutz", () => {
  test("esc() neutralisiert HTML", () => {
    const evil = "</span><img src=x onerror=alert(1)>";
    const out = UI.esc(evil);
    assert.ok(!out.includes("<img"));
    assert.ok(!out.includes("</span>"));
    assert.ok(out.includes("&lt;img"));
  });

  test("esc() kodiert Anführungszeichen", () => {
    assert.strictEqual(UI.esc('a"b\'c'), "a&quot;b&#39;c");
  });

  test("showLoading escapt den Befehl (keine rohe innerHTML-Injektion)", () => {
    const evilCmd = '</span><img src=x onerror=window.__xss=1>';
    let html = "";
    const orig = global.document.createElement;
    global.document.createElement = (tag) => {
      if (tag === "div") {
        const el = { className: "", id: "", appendChild() {}, remove() {} };
        Object.defineProperty(el, "innerHTML", {
          get: () => html,
          set: (v) => { html = v; },
        });
        return el;
      }
      return orig(tag);
    };
    global.document.getElementById = () => null;
    global.document.body.appendChild = () => {};
    try {
      UI.showLoading(evilCmd);
    } finally {
      global.document.createElement = orig;
    }
    assert.ok(!html.includes("<img"));
    assert.ok(html.includes("&lt;img"));
  });
});

describe("Kommando-Router", () => {
  test("erweitere erkannt (erstes Wort)", () => {
    assert.ok(app._isExpandCmd("erweitere die Stadt"));
    assert.ok(app._isExpandCmd("erweitern den Turm"));
    assert.ok(!app._isExpandCmd("bitte erweitere die Stadt"));
    assert.ok(!app._isExpandCmd("erweitereWelt"));
  });

  test("expand_wunsch_of entfernt nur das erste Wort", () => {
    assert.strictEqual(app._expandWunschOf("erweitere die Stadt um einen Hafen"), "die Stadt um einen Hafen");
    assert.strictEqual(app._expandWunschOf("erweitern"), "");
  });

  test("neues abenteuer nur als ganzes Kommando", () => {
    assert.ok(app._isNewAdventureCmd("neues abenteuer"));
    assert.ok(app._isNewAdventureCmd("Neues Abenteuer!"));
    assert.ok(!app._isNewAdventureCmd("Ich träume von einem neues abenteuer"));
    assert.ok(!app._isNewAdventureCmd("neues abenteuer aufräumen"));
  });
});

describe("api.pollJob", () => {
  test("done + error → reject", async () => {
    const api = new WorldcraftAPI();
    api.sessionId = "s1";
    api.secret = "x";
    api._fetch = async () => ({ status: "done", error: "kaputt" });
    await assert.rejects(api.pollJob("j1"), /kaputt/);
  });

  test("done + result → resolve mit result", async () => {
    const api = new WorldcraftAPI();
    api.sessionId = "s1";
    api.secret = "x";
    api._fetch = async () => ({ status: "done", result: { narration: "hi" } });
    const r = await api.pollJob("j1");
    assert.strictEqual(r.narration, "hi");
  });

  test("running → done (nächster Poll)", async () => {
    const api = new WorldcraftAPI();
    api.sessionId = "s1";
    api.secret = "x";
    let calls = 0;
    api._fetch = async () => {
      calls++;
      return calls === 1 ? { status: "running" } : { status: "done", result: { narration: "fertig" } };
    };
    // Intervall für schnellen Test überschreiben
    api._pollInterval = 5;
    const r = await api.pollJob("j1");
    assert.strictEqual(r.narration, "fertig");
    assert.ok(calls >= 2);
  });

  test("pollJob bricht auf Abort-Signal ab", async () => {
    const api = new WorldcraftAPI();
    api.sessionId = "s1";
    api.secret = "x";
    api._pollInterval = 1000;
    api._fetch = async () => ({ status: "running" });
    const ac = new AbortController();
    const p = api.pollJob("j1", null, ac.signal);
    await new Promise((r) => setTimeout(r, 30));
    ac.abort();
    await assert.rejects(p, (e) => e && e.name === "AbortError");
  });
});