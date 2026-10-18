# gui_suite_system.py — step-7 cross-surface checks: the GUI's migrate
# report must agree with the CLI's on the same fixture; a GUI submit
# must land in the same SQLite the CLI reads; addresses identical.

def suite(gui):
    import json as _json
    import sqlite3
    import subprocess

    ETH = "0x85ee7E71f762d772599cbF1EC20E651B30657521"
    BTC = "bc1qxe2zx5tv3hdreaej6s2x4p7han85uey828rrhg"
    DB = "/tmp/fieldkit-gui-test/datadir/fieldkit.db"
    FIXTURE = "/media/manitouabe8ich/project-backup/github_top_10/fieldkit/testdata/fixtures/typeform/sample-export.json"

    # ---- 1. migrate: GUI report == CLI report (same fixture) ----
    gui.click("#wizard-skip")
    gui.wait_count("#wizard-overlay:not(.hidden)", lambda c: c == 0, label="wizard skipped")
    gui.click("#nav-migrate")
    gui.fill("#migrate-input", FIXTURE)
    gui.click("#migrate-run")
    gui.wait_read("#migrate-result", lambda t: '"dry_run": true' in t, timeout=90, label="gui report ready")
    gui_rep = gui.read("#migrate-result", "text")["value"]

    cli = subprocess.run(
        ["/tmp/fieldkit", "migrate", "--from=typeform", "--dry-run", "--format=json", "--input", FIXTURE],
        capture_output=True, text=True)
    cli_rep = _json.loads(cli.stdout)

    # both parse; counts agree
    gui_json = _json.loads(gui_rep)
    gui.expect(gui_json["source"] == cli_rep["source"] == "typeform", "GUI and CLI agree on source")
    gui.expect(len(gui_json["survey"]["questions"]) == len(cli_rep["survey"]["questions"]) == 4,
               "GUI and CLI agree: 4 mapped questions")
    gui.expect(len(gui_json["unmapped"]) == len(cli_rep["unmapped"]) == 3,
               "GUI and CLI agree: 3 unmapped items")
    gui.expect(len(gui_json["endings"]) == len(cli_rep["endings"]) == 1,
               "GUI and CLI agree: 1 ending")

    # ---- 2. same survey submits from GUI and lands where the CLI reads it ----
    # author a minimal survey through the GUI
    gui.click("#nav-author")
    gui.fill("#author-name", "Cross-surface survey")
    minimal = {
        "title": "Cross-surface survey",
        "questions": [{"id": "q1", "type": "text", "title": "Say something", "required": True}],
    }
    gui.fill("#author-json", _json.dumps(minimal))
    gui.click("#author-create")
    gui.wait_read("#author-status", lambda t: "created survey" in t, timeout=60, label="survey created in GUI")

    # submit through the GUI's respondent runner
    gui.click("#nav-respond")
    gui.cmd("select", selector="#respond-survey", value="1")
    gui.wait_count("#respond-form input[data-qid]", lambda c: c == 1, label="form rendered")
    gui.cmd("fill", selector="#respond-form input", value="from the GUI")
    gui.click("#respond-submit")
    gui.wait_read("#respond-result", lambda t: "stored response" in t, timeout=60, label="GUI submit stored")

    # the CLI reads the same SQLite: export csv must contain the GUI's answer
    csv = subprocess.run(
        ["/tmp/fieldkit", "export", "csv", "--survey", "1", "--data-dir", "/tmp/fieldkit-gui-test/datadir"],
        capture_output=True, text=True)
    gui.expect("from the GUI" in csv.stdout, "CLI export sees the GUI-submitted response")

    # ---- 3. HTTP submit on the same data dir (third surface: the server) ----
    # start the server via CLI
    import socket, time as _time
    proc = subprocess.Popen(
        ["/tmp/fieldkit", "serve", "--data-dir", "/tmp/fieldkit-gui-test/datadir", "--addr", "127.0.0.1:18099"],
        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    try:
        _time.sleep(1.0)
        import urllib.request
        req = urllib.request.Request(
            "http://127.0.0.1:18099/surveys/1/submit",
            data=b'{"q1":"from the server"}',
            headers={"Content-Type": "application/json"})
        with urllib.request.urlopen(req) as resp:
            gui.expect(resp.status == 201, "HTTP submit accepted")
    finally:
        proc.terminate()
        proc.wait()

    csv = subprocess.run(
        ["/tmp/fieldkit", "export", "csv", "--survey", "1", "--data-dir", "/tmp/fieldkit-gui-test/datadir"],
        capture_output=True, text=True)
    gui.expect("from the GUI" in csv.stdout and "from the server" in csv.stdout,
               "one export shows both surfaces' responses")

    # ---- 4. addresses identical across surfaces ----
    # GUI
    gui.click("#nav-donate")
    gui.expect(gui.read("#donate-eth", "text")["value"] == ETH, "GUI donate ETH byte-for-byte")
    gui.expect(gui.read("#donate-btc", "text")["value"] == BTC, "GUI donate BTC byte-for-byte")
    # CLI
    out = subprocess.run(["/tmp/fieldkit", "donate"], capture_output=True, text=True).stdout
    gui.expect(ETH in out and BTC in out, "CLI donate has both addresses")
    # README + GOVERNANCE + docs/donate
    readme = open("/media/manitouabe8ich/project-backup/github_top_10/fieldkit/README.md").read()
    gov = open("/media/manitouabe8ich/project-backup/github_top_10/fieldkit/GOVERNANCE.md").read()
    don = open("/media/manitouabe8ich/project-backup/github_top_10/fieldkit/docs/donate.md").read()
    for name, doc in [("README", readme), ("GOVERNANCE", gov), ("docs/donate", don)]:
        gui.expect(ETH in doc and BTC in doc, name + " lists both addresses")
    # TUI donate view is tested in the TUI suite; the model fn is the same
    # source of truth — spot-check the binary prints it (non-TTY path pins
    # the hint; the donate text itself is covered by TestTUIDonateView).

    print("SYSTEM SUITE PASSED")


suite_exit = 0
suite(gui)
