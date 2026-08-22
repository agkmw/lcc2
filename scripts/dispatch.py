#!/usr/bin/env python3
"""Dispatch a specialized sub-agent on behalf of a worker coordinator.

Usage (from any directory):
    python3 <repo>/scripts/dispatch.py <worker> <role> "<instruction>"

Flow:
  - validates role and enforces pipeline prerequisites (e.g. tester may only
    run after implementer is done)
  - writes .agents/workers/<worker>/subtasks/<role>.md
  - launches an isolated `opencode run` session in the worker's worktree,
    seeded with the role charter (.agents/roles/<role>.md)
  - waits, then prints STATUS and the sub-agent's report

Exit code: 0 on success (done/approved), 1 on sub-agent failure/timeout,
2 on usage/prerequisite errors.
"""
import os
import subprocess
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import common


def die(msg):
    print(f"[dispatch] ERROR: {msg}", file=sys.stderr)
    sys.exit(2)


def main():
    if len(sys.argv) < 4:
        die("usage: dispatch.py <worker> <role> <instruction>")
    worker, role = sys.argv[1], sys.argv[2]
    instruction = " ".join(sys.argv[3:]).strip()
    if worker not in common.WORKERS:
        die(f"unknown worker '{worker}'")
    if role not in common.ROLES:
        die(f"unknown role '{role}' (valid: {', '.join(common.ROLES)})")

    wp = common.worker_paths(worker)
    if not os.path.isdir(wp["worktree"]):
        die(f"worktree missing: {wp['worktree']}")

    # Dependency guard: refuse to run out of order.
    for dep_role, allowed in common.PREREQS[role].items():
        st = common.read(common.sub_paths(worker, dep_role)["status"])
        if st not in allowed:
            die(f"prerequisite not met: {dep_role} status is "
                f"'{st or 'none'}' but must be one of {sorted(allowed)}")

    charter_file = os.path.join(common.ROLES_DIR, f"{role}.md")
    charter = common.read(charter_file)
    if not charter:
        die(f"missing or empty role charter: {charter_file}")

    sp = common.sub_paths(worker, role)
    common.write(sp["subtask"],
                 f"# Assignment for {worker}/{role}\n\n{instruction}\n")
    common.write(sp["result"], "")
    common.write(sp["status"], "")

    prompt = (
        f"You are the '{role}' sub-agent of worker '{worker}'.\n"
        f"Work only inside {wp['worktree']}\n\n"
        f"ROLE CHARTER:\n{charter}\n\n"
        f"YOUR ASSIGNMENT (saved at {sp['subtask']}, read it if needed):\n"
        f"{instruction}\n\n"
        "RULES:\n"
        "- Stay within your charter's responsibility; other roles handle the rest.\n"
        "- Do not touch other agents' files under .agents/workers/ except your own.\n"
        "- Write your report/findings to: " + sp["result"] + "\n"
        "- FINAL STEP: write exactly one status word to " + sp["status"] +
        " (done | failed | approved | changes-requested), per your charter.\n"
        "- Do not ask questions; act autonomously."
    )

    print(f"[dispatch] launching {worker}/{role} ...", flush=True)
    env = {k: v for k, v in os.environ.items() if not k.startswith("OPENCODE_")}
    try:
        with open(sp["log"], "w") as log:
            proc = subprocess.run(
                ["opencode", "run", "-m", common.MODEL, prompt],
                cwd=wp["worktree"], stdout=log, stderr=subprocess.STDOUT,
                env=env, timeout=common.AGENT_TIMEOUT)
        rc = proc.returncode
        timed_out = False
    except subprocess.TimeoutExpired:
        rc, timed_out = -1, True

    status = common.read(sp["status"])
    if timed_out:
        status = "failed"
        common.write(sp["result"],
                     f"(sub-agent timed out after {common.AGENT_TIMEOUT}s; "
                     f"see log: {sp['log']})\n")
    elif not status:
        status = "done" if rc == 0 else "failed"
    common.write(sp["status"], status)

    print(f"\n===== STATUS [{worker}/{role}]: {status} =====")
    report = common.read(sp["result"])
    print(report if report else "(no report written)")
    if timed_out:
        print(f"(full session log: {sp['log']})")

    sys.exit(0 if status in common.SUCCESS_STATUSES else 1)


if __name__ == "__main__":
    main()
