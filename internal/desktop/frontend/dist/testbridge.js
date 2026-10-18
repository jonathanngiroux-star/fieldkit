// testbridge.js: in-app test bridge interpreter (fixed switch, no
// eval). Ops: click, clickText, fill, select, check, key, read, count,
// texts, ping. Results go back through the TestResult binding as
// `TEST-RESULT {json}` lines the external driver reads. Armed only
// when the Go side has FIELDKIT_GUI_TESTBRIDGE=1 (it emits test:cmd).
(function () {
// bindings(): Wails namespaces generated bindings by the bound struct's
// package name. fieldkit binds internal/desktop.DesktopApp, so the
// namespace is window.go.desktop (older layouts used main). Resolve
// dynamically — never hardcode.
function fkBindings() {
  if (!window.go) return null;
  for (const ns of Object.values(window.go)) {
    if (ns && ns.DesktopApp) return ns.DesktopApp;
  }
  return null;
}
  // The Wails bindings (fkBindings()) are injected
  // asynchronously — guard inside send() and the registration poll,
  // never early-return the whole interpreter.
  const send = (obj) => {
    try {
      fkBindings().TestResult(JSON.stringify(obj));
    } catch (e) { /* bridge off in production */ }
  };

  function vis(el) {
    if (!el) return false;
    const style = window.getComputedStyle(el);
    if (style.display === "none" || style.visibility === "hidden") return false;
    const r = el.getBoundingClientRect();
    return r.width > 0 && r.height > 0;
  }

  function findByText(selector, text, scope) {
    const root = scope || document;
    const els = root.querySelectorAll(selector);
    for (const el of els) {
      if (el.textContent.trim().includes(text)) return el;
    }
    return null;
  }

  function fire(el, type, opts) {
    const ev = new Event(type, Object.assign({ bubbles: true, cancelable: true }, opts || {}));
    el.dispatchEvent(ev);
  }

  function fireKey(el, key) {
    const opts = { key: key, bubbles: true, cancelable: true };
    el.dispatchEvent(new KeyboardEvent("keydown", opts));
    el.dispatchEvent(new KeyboardEvent("keyup", opts));
  }

  function run(cmd) {
    const id = cmd.id;
    const done = (result) => send(Object.assign({ id: id }, result));
    try {
      switch (cmd.op) {
        case "ping":
          return done({ pong: true });
        case "click": {
          const el = document.querySelector(cmd.selector);
          if (!el) return done({ error: "no element " + cmd.selector });
          el.click();
          return done({ clicked: true });
        }
        case "clickText": {
          const el = findByText(cmd.selector || "button", cmd.text, cmd.scope ? document.querySelector(cmd.scope) : document);
          if (!el) return done({ error: "no element with text " + cmd.text });
          el.click();
          return done({ clicked: true });
        }
        case "fill": {
          const el = document.querySelector(cmd.selector);
          if (!el) return done({ error: "no element " + cmd.selector });
          el.focus();
          el.value = cmd.value;
          fire(el, "input");
          fire(el, "change");
          return done({ filled: true });
        }
        case "select": {
          const el = document.querySelector(cmd.selector);
          if (!el) return done({ error: "no element " + cmd.selector });
          if (cmd.value !== undefined) el.value = cmd.value;
          else if (cmd.label !== undefined) {
            for (const o of el.options) {
              if (o.textContent.trim() === cmd.label) { el.value = o.value; break; }
            }
          }
          fire(el, "change");
          return done({ selected: el.value });
        }
        case "key": {
          const el = document.activeElement || document.body;
          fireKey(el, cmd.key);
          return done({ key: cmd.key });
        }
        case "read": {
          const el = document.querySelector(cmd.selector);
          if (!el) return done({ error: "no element " + cmd.selector });
          const what = cmd.what || "text";
          if (what === "value") return done({ value: el.value });
          if (what === "checked") return done({ value: el.checked });
          if (what === "display") return done({ value: window.getComputedStyle(el).display });
          if (what === "disabled") return done({ value: el.disabled });
          if (what === "visible") return done({ value: vis(el) });
          return done({ value: el.textContent });
        }
        case "count": {
          return done({ value: document.querySelectorAll(cmd.selector).length });
        }
        case "texts": {
          const els = document.querySelectorAll(cmd.selector);
          return done({ value: Array.from(els).map((e) => e.textContent.trim()) });
        }
        case "attr": {
          const el = document.querySelector(cmd.selector);
          if (!el) return done({ error: "no element " + cmd.selector });
          return done({ value: el.getAttribute(cmd.name) });
        }
        default:
          return done({ error: "unknown op " + cmd.op });
      }
    } catch (e) {
      return done({ error: String(e) });
    }
  }

  // Register the test:cmd listener as soon as the Wails runtime shows
  // up — it is injected asynchronously and may not exist at
  // DOMContentLoaded; polling avoids the race where the bridge misses
  // commands entirely.
  let attempts = 0;
  function tryRegister() {
    if (window.runtime && window.runtime.EventsOn) {
      window.runtime.EventsOn("test:cmd", (line) => {
        let cmd;
        try { cmd = JSON.parse(line); } catch (e) { return; }
        run(cmd);
      });
      send({ id: "bridge-ready", ready: true });
      return;
    }
    if (attempts++ < 200) {
      setTimeout(tryRegister, 100);
    }
  }
  tryRegister();
})();
