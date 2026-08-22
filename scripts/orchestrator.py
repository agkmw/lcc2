#!/usr/bin/env python3
"""Minimal multi-agent orchestrator: 1 leader + 3 parallel OpenCode workers.

Usage:
    python3 scripts/orchestrator.py            # run the demo end-to-end
    python3 scripts/orchestrator.py --clean    # remove worker worktrees/branches

Flow:
  Leader writes .agents/tasks/worker-N.md, creates a git worktree per worker
  (.agents/worktrees/worker-N on branch agents/worker-N), launches 3 parallel
  `opencode run` sessions, waits for each to write:
      .agents/status/worker-N.md   ("done" / "failed")
      .agents/results/worker-N.md  (report)
  then prints the collected results.

Config via env: WORKER_MODEL (default opencode/x-preview-f-free),
WORKER_TIMEOUT seconds (default 900).
"""
import os
import shutil
import subprocess
import sys
import time

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
AGENTS = os.path.join(ROOT, ".agents")
WORKTREES = os.path.join(AGENTS, "worktrees")
MODEL = os.environ.get("WORKER_MODEL", "opencode/x-preview-f-free")
TIMEOUT = int(os.environ.get("WORKER_TIMEOUT", "900"))
WORKERS = ["worker-1", "worker-2", "worker-3"]


def git(*args, check=True):
    return subprocess.run(["git", *args], cwd=ROOT, capture_output=True,
                          text=True, check=check)


def ensure_initial_commit():
    """Worktrees need at least one commit; create one if HEAD is unborn."""
    r = git("rev-parse", "--verify", "HEAD", check=False)
    if r.returncode != 0:
        git("add", "-A")
        git("-c", "user.name=orchestrator",
            "-c", "user.email=orchestrator@local",
            "commit", "-m", "bootstrap commit for agent worktrees")
        print("[leader] created bootstrap commit")


def setup_worktree(name):
    wt = os.path.join(WORKTREES, name)
    if os.path.isdir(wt):
        git("worktree", "remove", "--force", wt, check=False)
    git("worktree", "add", "-B", f"agents/{name}", wt)
    print(f"[leader] worktree ready: {wt} (branch agents/{name})")
    return wt


def assign_task(name):
    task_path = os.path.join(AGENTS, "tasks", f"{name}.md")
    if not os.path.exists(task_path):
        sys.exit(f"missing task file: {task_path}")
    for d in ("results", "status"):
        open(os.path.join(AGENTS, d, f"{name}.md"), "w").close()
    return task_path


def launch_worker(name, wt, task_path):
    prompt = (
        f"You are '{name}', a worker agent. Work only inside {wt}.\n"
        f"1. Read your task file: {task_path}\n"
        "2. Complete the task exactly as written.\n"
        f"3. Write a short report of what you did (including any command "
        f"output) to: {AGENTS}/results/{name}.md\n"
        f"4. Finally write the single word done (or failed) to: "
        f"{AGENTS}/status/{name}.md\n"
        "Do not ask questions; just complete the task."
    )
    log = open(os.path.join(AGENTS, f"{name}.log"), "w")
    proc = subprocess.Popen(["opencode", "run", "-m", MODEL, prompt],
                            cwd=wt, stdout=log, stderr=subprocess.STDOUT)
    print(f"[leader] launched {name} (pid {proc.pid}, model {MODEL})")
    return proc


def wait_all(procs):
    deadline = time.time() + TIMEOUT
    pending = dict(procs)
    while pending and time.time() < deadline:
        for name, p in list(pending.items()):
            if p.poll() is not None:
                status = "done" if p.returncode == 0 else "failed"
                spath = os.path.join(AGENTS, "status", f"{name}.md")
                if p.returncode == 0 and not open(spath).read().strip():
                    open(spath, "w").write(status)
                print(f"[leader] {name} exited ({status})")
                del pending[name]
        if pending:
            time.sleep(3)
    for name, p in pending.items():
        p.kill()
        print(f"[leader] {name} timed out after {TIMEOUT}s")


def collect_results():
    print("\n===== LEADER SUMMARY =====")
    for name in WORKERS:
        status = open(os.path.join(AGENTS, "status", f"{name}.md")).read().strip()
        result = open(os.path.join(AGENTS, "results", f"{name}.md")).read().strip()
        print(f"\n--- {name}: status={status or 'no-status'} ---")
        print(result or "(empty result)")


def clean():
    git("worktree", "remove", "--force", WORKTREES, check=False)
    shutil.rmtree(WORKTREES, ignore_errors=True)
    for name in WORKERS:
        git("branch", "-D", f"agents/{name}", check=False)
    print("[leader] cleaned worktrees and branches")


def main():
    if "--clean" in sys.argv:
        clean()
        return
    os.makedirs(os.path.join(AGENTS, "tasks"), exist_ok=True)
    os.makedirs(os.path.join(AGENTS, "results"), exist_ok=True)
    os.makedirs(os.path.join(AGENTS, "status"), exist_ok=True)
    ensure_initial_commit()
    procs = {}
    for name in WORKERS:
        wt = setup_worktree(name)
        task_path = assign_task(name)
        procs[name] = launch_worker(name, wt, task_path)
    wait_all(procs)
    collect_results()


if __name__ == "__main__":
    main()
