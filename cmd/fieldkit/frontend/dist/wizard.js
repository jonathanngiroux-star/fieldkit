// wizard.js: the first-run wizard. Steps are specific to fieldkit, in
// order; every step runs a REAL operation through the Go bindings (the
// import step creates a real survey; the respond step stores a real
// response). The last step shows both donation addresses. Reopened any
// time via "Help: open wizard" or the w key.
(function () {
  const $ = (sel) => document.querySelector(sel);
  const $$ = (sel) => Array.from(document.querySelectorAll(sel));
  let steps = [];
  let idx = 0;
  let lastResult = "";

  async function load() {
    steps = await window.go.main.DesktopApp.WizardSteps();
  }

  function render() {
    const s = steps[idx];
    $("#wizard-title").textContent = s.title;
    // donate addresses must render as plain text nodes (no innerHTML)
    const body = $("#wizard-body");
    body.textContent = "";
    s.body.split("\n").forEach((line, i) => {
      if (i > 0) body.appendChild(document.createElement("br"));
      body.appendChild(document.createTextNode(line));
    });
    const fields = $("#wizard-fields");
    fields.textContent = "";
    (s.fields || []).forEach((f) => {
      const label = document.createElement("label");
      label.textContent = f.label + (f.required ? " *" : "");
      label.htmlFor = "wiz-" + f.id;
      const input = document.createElement("input");
      input.type = "text";
      input.id = "wiz-" + f.id;
      input.dataset.field = f.id;
      input.value = f.value || "";
      fields.appendChild(label);
      fields.appendChild(input);
    });
    $("#wizard-result").textContent = lastResult;
    $("#wizard-next").textContent = s.nextLabel || "Next";
    $("#wizard-back").disabled = idx === 0;
    // keyboard focus to the first field for a no-mouse path
    const first = fields.querySelector("input");
    if (first) first.focus();
    $("#wizard-progress").textContent = "step " + (idx + 1) + " / " + steps.length;
  }

  function collect() {
    const values = {};
    $$("#wizard-fields input").forEach((i) => { values[i.dataset.field] = i.value; });
    return values;
  }

  async function next() {
    const s = steps[idx];
    try {
      lastResult = await window.go.main.DesktopApp.WizardRunStep(s.id, collect());
    } catch (e) {
      lastResult = String(e);
      $("#wizard-result").textContent = lastResult;
      return; // stay on the step; the error is shown in place
    }
    if (idx < steps.length - 1) {
      idx++;
      render();
    } else {
      // finish: close and land on the main view (which now has content)
      close();
    }
  }

  function back() {
    if (idx > 0) { idx--; lastResult = ""; render(); }
  }

  async function skip() {
    // Skip is allowed: the tour is optional. Mark seen so it does not
    // nag, and land on the main view.
    try { await window.go.main.DesktopApp.WizardDone(); } catch (e) {}
    close();
  }

  function open() {
    $("#wizard-overlay").classList.remove("hidden");
    load().then(render);
  }
  function close() {
    $("#wizard-overlay").classList.add("hidden");
    // the wizard may have created surveys/responses — refresh whatever
    // the active tab shows so the main view is never stale after the
    // wizard closes. Exposed by app.js as window.fieldkitRefresh.
    try { window.fieldkitRefresh && window.fieldkitRefresh(); } catch (e) { /* boot-order safe */ }
  }

  $("#wizard-next").addEventListener("click", next);
  $("#wizard-back").addEventListener("click", back);
  $("#wizard-skip").addEventListener("click", skip);

  // keyboard path: Enter = next, Backspace = back, w = open wizard
  document.addEventListener("keydown", (ev) => {
    const openNow = !$("#wizard-overlay").classList.contains("hidden");
    if (openNow) {
      if (ev.key === "Enter") { ev.preventDefault(); next(); }
      if (ev.key === "Backspace" && document.activeElement.tagName !== "INPUT") { ev.preventDefault(); back(); }
      if (ev.key === "Escape") { ev.preventDefault(); skip(); }
    } else if (ev.key === "w" && document.activeElement.tagName !== "INPUT" && document.activeElement.tagName !== "TEXTAREA") {
      open();
    }
  });

  window.fieldkitWizard = { open, close };
})();
