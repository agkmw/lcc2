"""Shared constants and helpers for the multi-agent system."""
import os
import subprocess

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
AGENTS = os.path.join(ROOT, ".agents")
WORKTREES = os.path.join(AGENTS, "worktrees")
LOGS = os.path.join(AGENTS, "logs")
ROLES_DIR = os.path.join(AGENTS, "roles")
DISPATCH = os.path.join(ROOT, "scripts", "dispatch.py")

MODEL = os.environ.get("WORKER_MODEL", "opencode/x-preview-f-free")
WORKER_TIMEOUT = int(os.environ.get("WORKER_TIMEOUT", "1800"))
AGENT_TIMEOUT = int(os.environ.get("AGENT_TIMEOUT", "600"))

WORKERS = ["worker-1", "worker-2", "worker-3"]
ROLES = ["planner", "implementer", "tester", "reviewer", "debugger",
         "documenter"]

# Valid terminal statuses a sub-agent may write.
STATUSES = {"done", "failed", "approved", "changes-requested"}

# Minimum status requirements before a role may be dispatched.
# Missing entry = role has no prerequisite.
PREREQS = {
    "planner": {},
    "implementer": {"planner": {"done"}},
    "tester": {"implementer": {"done"}},
    "reviewer": {"tester": {"done"}},
    "debugger": {"implementer": {"done"}},
    "documenter": {"tester": {"done"},
                   "reviewer": {"approved", "done", "changes-requested"}},
}

SUCCESS_STATUSES = {"done", "approved"}


def worker_paths(name):
    """Top-level communication files of a worker (unchanged legacy layout)."""
    return {
        "task": os.path.join(AGENTS, "tasks", f"{name}.md"),
        "result": os.path.join(AGENTS, "results", f"{name}.md"),
        "status": os.path.join(AGENTS, "status", f"{name}.md"),
        "log": os.path.join(LOGS, f"{name}.log"),
        "worktree": os.path.join(WORKTREES, name),
        "subtree": os.path.join(AGENTS, "workers", name),
    }


def sub_paths(worker, role):
    """Communication files of a sub-agent (worker -> sub-agent channel)."""
    base = os.path.join(AGENTS, "workers", worker)
    return {
        "subtask": os.path.join(base, "subtasks", f"{role}.md"),
        "result": os.path.join(base, "results", f"{role}.md"),
        "status": os.path.join(base, "status", f"{role}.md"),
        "log": os.path.join(LOGS, f"{worker}-{role}.log"),
    }


def read(path):
    try:
        with open(path) as f:
            return f.read().strip()
    except FileNotFoundError:
        return ""


def write(path, text):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w") as f:
        f.write(text)


def git(*args, check=True):
    return subprocess.run(["git", *args], cwd=ROOT, capture_output=True,
                          text=True, check=check)
