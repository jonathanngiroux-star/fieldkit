# gui_suite_full.py — full-surface walk of the fieldkit desktop GUI.
# Every check is a real DOM interaction through the bridge plus, where
# the claim is persistence, a server-side cross-check. 100% or the
# commit is blocked.
#
# Polling discipline: after any click that triggers a Go binding, poll
# for the NEXT state's specific marker (never mere presence).

def suite(gui):
    import json as _json

    ETH = "0x85ee7E71f762d772599cbF1EC20E651B30657521"
    BTC = "bc1qxe2zx5tv3hdreaej6s2x4p7han85uey828rrhg"
    DB = "/tmp/fieldkit-gui-test/datadir/fieldkit.db"

    # ---- 1. first run: wizard opens before the main view ----
    gui.expect(gui.count("#wizard-overlay:not(.hidden)") == 1, "first launch opens the wizard")
    gui.wait_read("#wizard-title", lambda t: "store" in t.lower() or "data" in t.lower(),
                  label="wizard step 1 (data)")

    # data step: click Next advances to import (no risky op)
    gui.click("#wizard-next")
    gui.wait_read("#wizard-title", lambda t: "import" in t.lower(), label="step 2 (import)")

    # import step: click runs the REAL import (survey 1), advances to author
    gui.click("#wizard-next")
    gui.wait_read("#wizard-result", lambda t: "imported" in t, timeout=60, label="import ran")
    gui.wait_read("#wizard-title", lambda t: "author" in t.lower() or "own survey" in t.lower(),
                  label="step 3 (author)")

    # author step: FILL FIRST, then click runs the create with our value
    gui.wait_count("#wiz-title", lambda c: c == 1, label="author field rendered")
    gui.fill("#wiz-title", "Wizard made this")
    gui.click("#wizard-next")
    gui.wait_read("#wizard-result", lambda t: "created survey" in t, timeout=60, label="author ran")
    gui.wait_read("#wizard-title", lambda t: "respond" in t.lower() or "answer" in t.lower(),
                  label="step 4 (respond)")

    # respond step: FILL FIRST, then click stores a REAL response
    gui.wait_count("#wiz-q1", lambda c: c == 1, label="respond fields rendered")
    gui.fill("#wiz-q1", "yes")
    gui.fill("#wiz-q3", "from the wizard")
    gui.click("#wizard-next")
    gui.wait_read("#wizard-result", lambda t: "stored response" in t, timeout=60, label="respond ran")
    gui.wait_read("#wizard-title", lambda t: "responses" in t.lower() or "csv" in t.lower(),
                  label="step 5 (export)")

    # export step: click shows the CSV line, advances to donate
    gui.click("#wizard-next")
    gui.wait_read("#wizard-title", lambda t: "donat" in t.lower(), label="step 6 (donate)")
    body = gui.read("#wizard-body", "text")["value"]
    gui.expect(ETH in body, "wizard donate step shows the ETH address")
    gui.expect(BTC in body, "wizard donate step shows the BTC address")
    gui.click("#wizard-next")
    gui.wait_count("#wizard-overlay:not(.hidden)", lambda c: c == 0, label="wizard closed")

    # ---- 2. main view lands non-empty ----
    gui.wait_count("#surveys-body tr", lambda c: c == 2, label="2 wizard surveys listed")

    import sqlite3
    db = sqlite3.connect(DB)
    n = db.execute("SELECT COUNT(*) FROM surveys").fetchone()[0]
    responses = db.execute("SELECT COUNT(*) FROM responses").fetchone()[0]
    db.close()
    gui.expect(n == 2, "sqlite cross-check: 2 surveys (got %d)" % n)
    gui.expect(responses == 1, "sqlite cross-check: 1 response (got %d)" % responses)

    # ---- 3. author tab: create a survey through the real form ----
    gui.click("#nav-author")
    gui.fill("#author-name", "GUI-created survey")
    sample = {
        "title": "GUI survey",
        "questions": [
            {"id": "q1", "type": "single_choice", "title": "Pick one", "required": True, "choices": ["a", "b"]},
            {"id": "q2", "type": "text", "title": "Why?", "hidden": True},
        ],
        "logic": [
            {"id": "r1", "when": {"question": "q1", "op": "eq", "value": "b"}, "then": {"action": "show", "question": "q2"}}
        ],
    }
    gui.fill("#author-json", _json.dumps(sample))
    gui.click("#author-create")
    gui.wait_read("#author-status", lambda t: "created survey" in t, timeout=60, label="author create ran")

    # ---- 4. respondent runner: branch hides q2 until q1=b ----
    gui.click("#nav-respond")
    gui.wait_count("#respond-survey option", lambda c: c == 3, label="3 surveys in runner")
    gui.cmd("select", selector="#respond-survey", value="3")
    gui.wait_count("#respond-form input[data-qid], #respond-form select[data-qid]", lambda c: c == 2, label="form rendered")
    gui.cmd("select", selector="#respond-form select", value="b")
    gui.click("#respond-preview")
    gui.wait_read("#respond-result", lambda t: "q2" in t, timeout=60, label="branch fired (q2 visible)")
    gui.click("#respond-submit")
    gui.wait_read("#respond-result", lambda t: "stored response" in t, timeout=60, label="submit stored")
    db = sqlite3.connect(DB)
    n2 = db.execute("SELECT COUNT(*) FROM responses").fetchone()[0]
    db.close()
    gui.expect(n2 == 2, "sqlite cross-check: 2 responses now (got %d)" % n2)

    # ---- 5. responses table ----
    gui.click("#nav-responses")
    gui.cmd("select", selector="#responses-survey", value="3")
    gui.click("#responses-reload")
    gui.wait_count("#responses-body tr", lambda c: c == 1, label="responses table row")

    # ---- 6. migrate tab: sample + dry-run ----
    gui.click("#nav-migrate")
    gui.click("#migrate-sample")
    gui.wait_read("#migrate-input", lambda v: v.endswith(".json"), what="value", label="sample path loaded")
    gui.click("#migrate-run")
    gui.wait_read("#migrate-result", lambda t: '"dry_run": true' in t, timeout=90, label="migrate report ready")
    rep = gui.read("#migrate-result", "text")["value"]
    gui.expect('"unmapped"' in rep, "report lists unmapped items honestly")
    gui.expect("matrix" in rep, "matrix shows as unmapped (plugin contract hook)")

    # ---- 7. plugin tab: error path + sample passes ----
    gui.click("#nav-plugin")
    gui.fill("#plugin-input", "/tmp/fieldkit-gui-test/not-a-plugin")
    gui.click("#plugin-validate")
    gui.wait_read("#plugin-result",
                  lambda t: "manifest" in t.lower() or "no such" in t.lower() or "stat" in t.lower(),
                  label="bad plugin error shown")
    gui.fill("#plugin-input", "/media/manitouabe8ich/project-backup/github_top_10/fieldkit/testdata/plugins/sample")
    gui.click("#plugin-validate")
    gui.wait_read("#plugin-result", lambda t: t.startswith("ok sample"), label="sample plugin passes")

    # ---- 8. donate tab: byte-for-byte ----
    gui.click("#nav-donate")
    eth = gui.read("#donate-eth", "text")["value"]
    btc = gui.read("#donate-btc", "text")["value"]
    gui.expect(eth == ETH, "donate tab ETH address byte-for-byte")
    gui.expect(btc == BTC, "donate tab BTC address byte-for-byte")
    footer = gui.read("#footer-donate-short", "text")["value"]
    gui.expect(ETH in footer and BTC in footer, "footer carries both addresses")

    # ---- 9. wizard reopen from Help; skip closes ----
    gui.click("#wizard-btn")
    gui.wait_count("#wizard-overlay:not(.hidden)", lambda c: c == 1, label="wizard reopened")
    gui.click("#wizard-skip")
    gui.wait_count("#wizard-overlay:not(.hidden)", lambda c: c == 0, label="skip closed the wizard")

    # ---- 10. keyboard path ----
    gui.key("w")
    gui.wait_count("#wizard-overlay:not(.hidden)", lambda c: c == 1, label="w key opened the wizard")
    gui.key("Escape")
    gui.wait_count("#wizard-overlay:not(.hidden)", lambda c: c == 0, label="Escape closed the wizard")

    print("ALL CHECKS PASSED")


suite_exit = 0
suite(gui)
