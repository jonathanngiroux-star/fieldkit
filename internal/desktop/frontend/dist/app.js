// app.js: fieldkit desktop frontend (vanilla JS, no bundler). All logic
// lives in the Go bindings; this file renders and calls.
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
  const $ = (sel) => document.querySelector(sel);
  const $$ = (sel) => Array.from(document.querySelectorAll(sel));

  // ---------- navigation ----------
  function activateTab(name) {
    $$(".nav-btn[data-tab]").forEach((b) => b.classList.toggle("active", b.dataset.tab === name));
    $$(".tab").forEach((s) => s.classList.toggle("active", s.id === "tab-" + name));
    if (name === "surveys") refreshSurveys();
    if (name === "respond") loadRespondSurveys();
    if (name === "responses") loadResponsesSurveys();
  }
  $$(".nav-btn[data-tab]").forEach((b) => b.addEventListener("click", () => activateTab(b.dataset.tab)));
  $("#wizard-btn").addEventListener("click", () => window.fieldkitWizard.open());

  function status(sel, msg, isError) {
    const el = $(sel);
    el.textContent = msg;
    el.classList.toggle("wizard-error", !!isError);
    el.classList.toggle("wizard-ok", !isError && !!msg);
  }

  // ---------- surveys ----------
  async function refreshSurveys() {
    const body = $("#surveys-body");
    const loading = $("#surveys-loading");
    const empty = $("#surveys-empty");
    const table = $("#surveys-table");
    loading.classList.remove("hidden");
    empty.classList.add("hidden");
    table.classList.add("hidden");
    try {
      const rows = await fkBindings().Surveys();
      loading.classList.add("hidden");
      if (!rows || rows.length === 0) {
        empty.classList.remove("hidden");
        return;
      }
      table.classList.remove("hidden");
      body.textContent = "";
      rows.forEach((r) => {
        const tr = document.createElement("tr");
        const tdId = document.createElement("td"); tdId.textContent = String(r.id);
        const tdName = document.createElement("td"); tdName.textContent = r.name;
        const tdCount = document.createElement("td"); tdCount.textContent = String(r.count);
        const tdCreated = document.createElement("td"); tdCreated.textContent = r.created;
        const tdDel = document.createElement("td");
        const btn = document.createElement("button"); btn.textContent = "Delete";
        btn.addEventListener("click", async () => {
          if (!confirm("Delete survey " + r.name + " and its responses?")) return;
          try {
            await fkBindings().DeleteSurvey(r.id);
            refreshSurveys();
          } catch (e) { alert(String(e)); }
        });
        tdDel.appendChild(btn);
        tr.append(tdId, tdName, tdCount, tdCreated, tdDel);
        body.appendChild(tr);
      });
    } catch (e) {
      loading.classList.add("hidden");
      status("#surveys-loading", String(e), true);
    }
  }

  // ---------- author ----------
  const sampleSurvey = {
    title: "Branch demo",
    questions: [
      { id: "q1", type: "single_choice", title: "Did the onboarding flow work?", required: true, choices: ["yes", "no"] },
      { id: "q2", type: "text", title: "What failed?", hidden: true },
      { id: "q3", type: "text", title: "Anything else?" }
    ],
    logic: [
      { id: "br1", when: { question: "q1", op: "eq", value: "no" }, then: { action: "show", question: "q2" } }
    ]
  };
  $("#author-load-sample").addEventListener("click", () => {
    $("#author-json").value = JSON.stringify(sampleSurvey, null, 2);
    status("#author-status", "sample loaded — edit and create", false);
  });
  $("#author-create").addEventListener("click", async () => {
    status("#author-status", "creating…", false);
    try {
      const out = await fkBindings().CreateSurvey($("#author-json").value, $("#author-name").value);
      status("#author-status", "created survey " + out.id, false);
      refreshSurveys();
    } catch (e) {
      status("#author-status", String(e), true);
    }
  });

  // ---------- respondent ----------
  async function loadRespondSurveys() {
    const sel = $("#respond-survey");
    sel.textContent = "";
    try {
      const rows = await fkBindings().Surveys();
      (rows || []).forEach((r) => {
        const o = document.createElement("option");
        o.value = String(r.id);
        o.textContent = r.name + " (" + r.count + " responses)";
        sel.appendChild(o);
      });
      if ((rows || []).length > 0) renderRespondForm();
    } catch (e) { /* empty list is fine */ }
  }
  $("#respond-reload").addEventListener("click", loadRespondSurveys);
  $("#respond-survey").addEventListener("change", renderRespondForm);

  async function renderRespondForm() {
    const id = Number($("#respond-survey").value);
    const form = $("#respond-form");
    form.textContent = "";
    if (!id) return;
    let def;
    try {
      def = JSON.parse(await fkBindings().SurveyDefinition(id));
    } catch (e) {
      form.textContent = String(e);
      return;
    }
    (def.questions || []).forEach((q) => {
      const wrap = document.createElement("div");
      wrap.className = "row";
      wrap.dataset.qid = q.id;
      const label = document.createElement("label");
      label.textContent = q.title + (q.required ? " *" : "") + (q.hidden ? " (hidden)" : "");
      wrap.appendChild(label);
      let input;
      if (q.type === "single_choice" || q.type === "multi_choice") {
        input = document.createElement("select");
        if (q.type === "multi_choice") input.multiple = true;
        (q.choices || []).forEach((c) => {
          const o = document.createElement("option");
          o.value = c; o.textContent = c;
          input.appendChild(o);
        });
      } else {
        input = document.createElement("input");
        input.type = "text";
        input.dataset.qid = q.id;
        input.placeholder = q.type;
      }
      input.dataset.qid = q.id;
      input.dataset.qtype = q.type;
      wrap.appendChild(input);
      form.appendChild(wrap);
    });
  }

  function collectAnswers() {
    const answers = {};
    $$("#respond-form [data-qid]").forEach((el) => {
      const qid = el.dataset.qid;
      if (el.tagName === "SELECT") {
        if (el.multiple) {
          const picked = Array.from(el.selectedOptions).map((o) => o.value);
          if (picked.length) answers[qid] = picked;
        } else if (el.value) {
          answers[qid] = el.value;
        }
      } else if (el.value !== "") {
        const t = el.dataset.qtype;
        if (t === "rating" || t === "nps") {
          const n = Number(el.value);
          if (!isNaN(n)) answers[qid] = n;
        } else {
          answers[qid] = el.value;
        }
      }
    });
    return answers;
  }

  function refreshRespondResult(res) {
    const el = $("#respond-result");
    const lines = [];
    lines.push("visible: " + (res.visible_questions || []).join(", "));
    if (res.next) lines.push("jumps to: " + res.next);
    (res.errors || []).forEach((e) => lines.push("error " + e.question + ": " + e.message));
    el.textContent = lines.join("\n") || "ok";
  }

  $("#respond-preview").addEventListener("click", async () => {
    const id = Number($("#respond-survey").value);
    if (!id) return;
    try {
      const res = await fkBindings().Preview(id, JSON.stringify(collectAnswers()));
      refreshRespondResult(res);
    } catch (e) { $("#respond-result").textContent = String(e); }
  });
  $("#respond-submit").addEventListener("click", async () => {
    const id = Number($("#respond-survey").value);
    if (!id) return;
    try {
      const out = await fkBindings().SubmitResponse(id, JSON.stringify(collectAnswers()));
      if (out.result && out.result.errors && out.result.errors.length) {
        refreshRespondResult(out.result);
      } else {
        $("#respond-result").textContent = "stored response " + out.response_id;
      }
    } catch (e) { $("#respond-result").textContent = String(e); }
  });

  // ---------- responses ----------
  async function loadResponsesSurveys() {
    const sel = $("#responses-survey");
    sel.textContent = "";
    try {
      const rows = await fkBindings().Surveys();
      (rows || []).forEach((r) => {
        const o = document.createElement("option");
        o.value = String(r.id);
        o.textContent = r.name;
        sel.appendChild(o);
      });
      if ((rows || []).length > 0) renderResponses();
    } catch (e) { /* fine */ }
  }
  $("#responses-reload").addEventListener("click", renderResponses);
  $("#responses-survey").addEventListener("change", renderResponses);

  async function renderResponses() {
    const id = Number($("#responses-survey").value);
    const head = $("#responses-head");
    const body = $("#responses-body");
    const empty = $("#responses-empty");
    const table = $("#responses-table");
    head.textContent = ""; body.textContent = "";
    if (!id) return;
    let def, rows;
    try {
      def = JSON.parse(await fkBindings().SurveyDefinition(id));
      rows = await fkBindings().Responses(id);
    } catch (e) { return; }
    if (!rows || rows.length === 0) {
      empty.classList.remove("hidden");
      table.classList.add("hidden");
      return;
    }
    empty.classList.add("hidden");
    table.classList.remove("hidden");
    const qids = (def.questions || []).map((q) => q.id);
    const trh = document.createElement("tr");
    ["response_id", "submitted"].concat(qids).forEach((h) => {
      const th = document.createElement("th"); th.textContent = h; trh.appendChild(th);
    });
    head.appendChild(trh);
    rows.forEach((r) => {
      const tr = document.createElement("tr");
      const tdId = document.createElement("td"); tdId.textContent = String(r.response_id);
      const tdWhen = document.createElement("td"); tdWhen.textContent = r.submitted;
      tr.append(tdId, tdWhen);
      qids.forEach((qid) => {
        const td = document.createElement("td");
        const v = (r.answers || {})[qid];
        td.textContent = v === undefined ? "" : (typeof v === "object" ? JSON.stringify(v) : String(v));
        tr.appendChild(td);
      });
      body.appendChild(tr);
    });
  }

  $("#responses-csv").addEventListener("click", async () => {
    const id = Number($("#responses-survey").value);
    if (!id) return;
    try {
      const csv = await fkBindings().ExportCSV(id);
      const path = await fkBindings().SaveCSVPath("survey-" + id + "-responses.csv");
      if (!path) return;
      await fkBindings().SaveFile(path, csv);
      alert("saved " + path);
    } catch (e) { alert(String(e)); }
  });

  // ---------- migrate ----------
  $("#migrate-pick").addEventListener("click", async () => {
    try {
      const p = await fkBindings().PickJSONPath("Choose a Typeform export JSON");
      if (p) $("#migrate-input").value = p;
    } catch (e) { /* cancelled */ }
  });
  $("#migrate-sample").addEventListener("click", async () => {
    try {
      const p = await fkBindings().SampleFixturePath();
      $("#migrate-input").value = p;
      status("#migrate-status", "sample export loaded", false);
    } catch (e) { status("#migrate-status", String(e), true); }
  });
  $("#migrate-run").addEventListener("click", async () => {
    const input = $("#migrate-input").value;
    if (!input) { status("#migrate-status", "choose an export first", true); return; }
    status("#migrate-status", "running…", false);
    try {
      const rep = await fkBindings().MigrateDryRun("typeform", input);
      $("#migrate-result").textContent = rep;
      status("#migrate-status", "dry-run complete — nothing imported", false);
    } catch (e) {
      $("#migrate-result").textContent = String(e);
      status("#migrate-status", "dry-run failed", true);
    }
  });

  // ---------- plugin ----------
  $("#plugin-pick").addEventListener("click", async () => {
    try {
      const p = await fkBindings().PickJSONPath("Choose a plugin manifest or directory");
      if (p) $("#plugin-input").value = p;
    } catch (e) { /* cancelled */ }
  });
  $("#plugin-validate").addEventListener("click", async () => {
    const target = $("#plugin-input").value;
    if (!target) { $("#plugin-result").textContent = "enter a plugin directory or manifest path"; return; }
    try {
      const out = await fkBindings().PluginValidate(target);
      $("#plugin-result").textContent = out.message;
    } catch (e) {
      $("#plugin-result").textContent = String(e);
    }
  });

  // ---------- donate ----------
  const ETH = "0x85ee7E71f762d772599cbF1EC20E651B30657521";
  const BTC = "bc1qxe2zx5tv3hdreaej6s2x4p7han85uey828rrhg";
  $("#donate-eth").textContent = ETH;
  $("#donate-btc").textContent = BTC;
  $("#footer-donate-short").textContent = "Donate: ETH/USDC " + ETH + " · BTC " + BTC;
  async function copyText(text) {
    try {
      await fkBindings().CopyAddress(text);
      return true;
    } catch (e) {
      try {
        await navigator.clipboard.writeText(text);
        return true;
      } catch (e2) { return false; }
    }
  }
  $("#copy-eth").addEventListener("click", async () => {
    status("#plugin-result", "", false);
    (await copyText(ETH)) ? alert("Ethereum address copied") : alert(ETH);
  });
  $("#copy-btc").addEventListener("click", async () => {
    (await copyText(BTC)) ? alert("Bitcoin address copied") : alert(BTC);
  });

  // ---------- boot ----------
  // refresh hook for the wizard (which may create surveys/responses)
  window.fieldkitRefresh = function () {
    const active = document.querySelector(".tab.active");
    if (!active) return;
    if (active.id === "tab-surveys") refreshSurveys();
    if (active.id === "tab-respond") loadRespondSurveys();
    if (active.id === "tab-responses") loadResponsesSurveys();
  };
  (async function boot() {
    // wait for the Wails bindings to be injected (async; up to 10s)
    for (let i = 0; i < 100 && !fkBindings(); i++) {
      await new Promise((r) => setTimeout(r, 100));
    }
    try {
      const v = await fkBindings().Version();
      $("#version-line").textContent = "v" + v;
      $("#footer-version").textContent = "v" + v;
    } catch (e) { /* version is cosmetic */ }
    refreshSurveys();
    // first run: open the wizard before anything else
    try {
      const seen = await fkBindings().WizardSeen();
      if (!seen) window.fieldkitWizard.open();
    } catch (e) { /* wizard is not fatal */ }
  })();
})();
