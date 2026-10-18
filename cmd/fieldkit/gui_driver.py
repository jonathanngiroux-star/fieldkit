#!/usr/bin/env python3
"""gui_driver.py — external driver for the fieldkit in-app test bridge.

Writes commands (unique id per command) to the command file the Go
poller reads, and polls the result log with a seek offset so only fresh
lines are matched. Both defenses matter: offset prevents re-reading old
results, ids prevent crossing results from abandoned commands.

wait_* helpers poll for step-specific markers — "poll until non-empty"
passes on the previous step's content; never poll for mere presence.
"""
import json
import os
import sys
import time
import uuid

CMD_FILE = os.environ.get("FIELDKIT_GUI_CMD_FILE", "/tmp/fieldkit-gui-cmd.jsonl")
RESULT_FILE = os.environ.get("FIELDKIT_GUI_RESULT_FILE", "/tmp/fieldkit-gui-result.log")


class Gui:
    def __init__(self, timeout=20.0):
        self.timeout = timeout
        self.seq = 0
        try:
            self.offset = os.path.getsize(RESULT_FILE)
        except OSError:
            self.offset = 0

    def _next_id(self):
        self.seq += 1
        return "%d-%s" % (self.seq, uuid.uuid4().hex[:8])

    def cmd(self, op, **kw):
        cid = self._next_id()
        payload = dict(kw)
        payload["id"] = cid
        payload["op"] = op
        line = json.dumps(payload) + "\n"
        with open(CMD_FILE, "a") as f:
            f.write(line)
        deadline = time.time() + self.timeout
        while time.time() < deadline:
            try:
                size = os.path.getsize(RESULT_FILE)
            except OSError:
                time.sleep(0.05)
                continue
            if size > self.offset:
                with open(RESULT_FILE) as f:
                    f.seek(self.offset)
                    fresh = f.read()
                self.offset += len(fresh)
                for ln in fresh.splitlines():
                    if ln.startswith("TEST-RESULT "):
                        try:
                            obj = json.loads(ln[len("TEST-RESULT "):])
                        except json.JSONDecodeError:
                            continue
                        if obj.get("id") == cid:
                            if "error" in obj:
                                raise AssertionError("cmd %s failed: %s" % (op, obj["error"]))
                            return obj
            time.sleep(0.05)
        raise AssertionError("timeout waiting for %s (id=%s)" % (op, cid))

    # ---- polling helpers (step-specific markers, never mere presence) ----
    def wait_read(self, selector, predicate, what="text", timeout=30.0, label=None):
        """Poll a read op until the value satisfies predicate(marker-specific)."""
        deadline = time.time() + timeout
        last = None
        while time.time() < deadline:
            last = self.read(selector, what)["value"]
            if predicate(last):
                return last
            time.sleep(0.15)
        raise AssertionError("timeout waiting for %s to satisfy %s (last=%r)%s" % (
            selector, getattr(predicate, "__name__", "predicate"), last,
            " [%s]" % label if label else ""))

    def wait_count(self, selector, predicate, timeout=30.0, label=None):
        deadline = time.time() + timeout
        last = -1
        while time.time() < deadline:
            last = self.count(selector)
            if predicate(last):
                return last
            time.sleep(0.15)
        raise AssertionError("timeout waiting for count(%s) to satisfy predicate (last=%d)%s" % (
            selector, last, " [%s]" % label if label else ""))

    # ---- convenience wrappers ----
    def ping(self):
        return self.cmd("ping")

    def click(self, selector):
        return self.cmd("click", selector=selector)

    def click_text(self, text, selector="button", scope=None):
        kw = dict(text=text, selector=selector)
        if scope:
            kw["scope"] = scope
        return self.cmd("clickText", **kw)

    def fill(self, selector, value):
        return self.cmd("fill", selector=selector, value=value)

    def read(self, selector, what="text"):
        return self.cmd("read", selector=selector, what=what)

    def count(self, selector):
        return self.cmd("count", selector=selector)["value"]

    def texts(self, selector):
        return self.cmd("texts", selector=selector)["value"]

    def key(self, key):
        return self.cmd("key", key=key)

    def expect(self, cond, label):
        if not cond:
            raise AssertionError("check failed: " + label)
        print("PASS " + label)


def run_suite(path):
    gui = Gui()
    deadline = time.time() + 30
    while time.time() < deadline:
        try:
            gui.ping()
            break
        except AssertionError:
            time.sleep(0.5)
    else:
        print("FAIL ping (app not ready)")
        return 1
    ns = {"gui": gui, "print": print, "suite_exit": 0}
    with open(path) as f:
        code = f.read()
    try:
        exec(compile(code, path, "exec"), ns)
    except AssertionError as e:
        print("FAIL " + str(e))
        return 1
    return ns.get("suite_exit", 0)


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("usage: gui_driver.py <suite.py>")
        sys.exit(2)
    sys.exit(run_suite(sys.argv[1]))
