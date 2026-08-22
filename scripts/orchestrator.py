#!/usr/bin/env python3
"""Minimal hierarchical multi-agent orchestrator.

    leader (this script)
      └── worker-1..3  (opencode sessions = COORDINATORS, one worktree each)
            ├── planner
            ├── implementer
            ├── tester
            ├── reviewer
            ├── debugger   (only on failure / requested changes)
            └── documenter

Usage:
    python3 scripts/orchestrator.py            # run the demo end-to-end
    python3 scripts/orchestrator.py --clean    # remove worktrees/branches

Flow:
  Leader writes .agents/tasks/worker-N.md and creates one git worktree per
  worker (.agents/worktrees/worker-N on branch agents/worker-N), then launches
  3 parallel coordinator sessions. Each coordinator delegates through
  scripts/dispatch.py (see its docstring for the sub-agent protocol), which
  enforces the pipeline: plan -> implement -> test -> review -> document.
  When a worker finishes it writes .agents/results/ + .agents/status/.

Config via env: WORKER_MODEL, WORKER_TIMEOUT s, AGENT_TIMEOUT s.
"""
import os
import shutil
import subprocess
import sys
import time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import common


def git(*args, check=True):
    return common.git(*args, check=check)


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
    branch = f"agents/{name}"
    wt = common.worker_paths(name)["worktree"]
    if os.path.isdir(wt):
        git("worktree", "remove", "--force", wt, check=False)
    git("worktree", "prune")  # drop stale registrations from killed runs
    r = git("worktree", "add", "-B", branch, wt, check=False)
    if r.returncode != 0:  # branch stuck on a stale worktree: reset it
        git("branch", "-D", branch, check=False)
        git("worktree", "prune")
        git("worktree", "add", "-B", branch, wt)
    print(f"[leader] worktree ready: {wt} (branch {branch})")


def reset_worker_channels(name):
    """Fresh worker-level files and an empty sub-agent tree for this run."""
    wp = common.worker_paths(name)
    shutil.rmtree(wp["subtree"], ignore_errors=True)
    for key in ("result", "status"):
        common.write(wp[key], "")
    task = wp["task"]
    if not os.path.exists(task):
        sys.exit(f"[leader] missing task file: {task}")
    return task


def coordinator_prompt(name, task_path):
    wp = common.worker_paths(name)
    return (
        f"You are '{name}', a COORDINATOR agent. Your job is to complete the "
        f"task in {task_path} by delegating ALL real work to specialized "
        "sub-agents. Never implement the task yourself; you only orchestrate, "
        "read reports, and decide next steps.\n\n"
        "Dispatch sub-agents with this exact command pattern:\n"
        f"    python3 {common.DISPATCH} {name} <role> \"<self-contained instruction>\"\n\n"
        f"Available roles: {', '.join(common.ROLES)}\n\n"
        "REQUIRED PIPELINE ORDER:\n"
        "1. planner     - turn your task into a concrete step-by-step plan.\n"
        "2. implementer - execute the plan (pass along the relevant steps).\n"
        "3. tester      - verify the implementation with real commands.\n"
        "4. reviewer    - review quality; wait for its verdict.\n"
        "5. debugger    - ONLY if tester failed or reviewer asked for changes;\n"
        "                  afterwards re-run tester (and reviewer) until pass/approved.\n"
        "6. documenter  - write final docs/summary once approved.\n\n"
        "RULES:\n"
        "- Each dispatch BLOCKS until the sub-agent finishes, then prints its "
        "STATUS and report. Read both before proceeding.\n"
        "- If dispatch exits with code 2 about prerequisites, you skipped a "
        "step - go back and run the missing phase first.\n"
        "- Retry a failing sub-agent at most once with a sharper instruction.\n"
        "- You may read files in the worktree to write better instructions, "
        "but never create/modify project files yourself.\n\n"
        f"FINISH: write a short final report to {wp['result']} summarizing what "
        f"each sub-agent did and the outcome, then write exactly 'done' (or "
        f"'failed') to {wp['status']} as your very last action."
    )


def launch_worker(name, task_path):
    prompt = coordinator_prompt(name, task_path)
    log = open(common.worker_paths(name)["log"], "w")
    proc = subprocess.Popen(["opencode", "run", "-m", common.MODEL, prompt],
                            cwd=common.worker_paths(name)["worktree"],
                            stdout=log, stderr=subprocess.STDOUT)
    print(f"[leader] launched {name} (pid {proc.pid}, model {common.MODEL})")
    return proc


def wait_all(procs):
    deadline = time.time() + common.WORKER_TIMEOUT
    pending = dict(procs)
    while pending and time.time() < deadline:
        for name, p in list(pending.items()):
            if p.poll() is not None:
                status = "done" if p.returncode == 0 else "failed"
                spath = common.worker_paths(name)["status"]
                if not common.read(spath):
                    common.write(spath, status)
                print(f"[leader] {name} exited ({common.read(spath) or status})")
                del pending[name]
        if pending:
            time.sleep(3)
    for name, p in pending.items():
        p.kill()
        print(f"[leader] {name} timed out after {common.WORKER_TIMEOUT}s")


def collect_results():
    print("\n===== LEADER SUMMARY =====")
    for name in common.WORKERS:
        wp = common.worker_paths(name)
        print(f"\n--- {name}: status={common.read(wp['status']) or 'no-status'} ---")
        print(common.read(wp["result"]) or "(empty result)")
        subs = []
        subtree = wp["subtree"]
        if os.path.isdir(os.path.join(subtree, "status")):
            for role in common.ROLES:
                st = common.read(common.sub_paths(name, role)["status"])
                if st:
                    subs.append(f"{role}={st}")
        print(f"    sub-agents: {', '.join(subs) if subs else '(none dispatched)'}")


def clean():
    git("worktree", "remove", "--force", common.WORKTREES, check=False)
    shutil.rmtree(common.WORKTREES, ignore_errors=True)
    for name in common.WORKERS:
        git("branch", "-D", f"agents/{name}", check=False)
    print("[leader] cleaned worktrees and branches")


def main():
    if "--clean" in sys.argv:
        clean()
        return
    for d in (common.LOGS, common.ROLES_DIR):
        os.makedirs(d, exist_ok=True)
    ensure_initial_commit()
    procs = {}
    for name in common.WORKERS:
        setup_worktree(name)
        task_path = reset_worker_channels(name)
        procs[name] = launch_worker(name, task_path)
    wait_all(procs)
    collect_results()


if __name__ == "__main__":
    main()
