# Universal Agent Entry Point & Context Router (`dev_com_agent.md`)

> **Single Source of Agent Entry & Context Routing**
> Follow this progressive context disclosure protocol to understand repository ground truths, architectural invariants, and operational workflows.

---

## 1. Master Context Router

1. **Step 1: Read the Master Index**
   - Read [`dev_md_guides/agent.md`](dev_md_guides/agent.md) first to discover all active guides and their summaries.

2. **Step 2: Read Targeted Guides Based on Your Current Task**
   - **Branch / Worktree State**: [`dev_md_guides/branch.md`](dev_md_guides/branch.md)
   - **Feature Tracking**: [`dev_md_guides/features.md`](dev_md_guides/features.md)
   - **Code Architecture**: [`dev_md_guides/structure.md`](dev_md_guides/structure.md)
   - **Decisions & Invariants**: [`dev_md_guides/memory.md`](dev_md_guides/memory.md)
   - **Failure Modes & Fixes**: [`dev_md_guides/gotchas.md`](dev_md_guides/gotchas.md)
   - **Execution Workflows**: [`dev_md_guides/flow.md`](dev_md_guides/flow.md)
   - **Operational Playbooks**: [`dev_md_guides/playbook.md`](dev_md_guides/playbook.md) (reproducible framework runbooks & execution recipes)
   - **Validated Solutions**: [`dev_md_guides/solution.md`](dev_md_guides/solution.md) (mutable catalog of verified problem resolutions & latest empirical truths)
   - **Environment Endpoints**: [`dev_md_guides/directory.md`](dev_md_guides/directory.md)
   - **Secrets & Credentials**: [`dev_md_guides/credentials.md.sample`](dev_md_guides/credentials.md.sample) / local [`dev_md_guides/credentials.md`](dev_md_guides/credentials.md)
   - **Operational Audit Log**: [`dev_md_guides/changelog.md`](dev_md_guides/changelog.md)
   - **Dynamic Topic Guides**: Consult [`dev_md_guides/agent.md`](dev_md_guides/agent.md) for dynamic topic guides (e.g. `commands.md`, `mcp.md`).

---

## 2. Inviolable Core Ground Rules

1. **Zero Hardcoded Directories & Endpoints**: Reference all paths/servers via `dev_md_guides/directory.md`.
2. **Zero Credential Exposure**: Never commit credentials.md or live secrets to git. Real secrets remain strictly local and gitignored. Issue explicit warning before any secret commits.
3. **Local Sensitive Overrides Protocol (`local.<filename>.md` / `*.local.md`)**: Any machine-specific or sensitive notes (e.g. `local.agent.md`, `dev_md_guides/local.agent.md`) are strictly gitignored by default and must never be pushed to version control.
4. **Proactive Context Compaction Triggers**: Automatically trigger compaction at task completion boundaries.
5. **Strict Grounding & Anti-Hallucination Audit Protocol**: When auditing or exploring code, report ONLY what ACTUALLY exists on disk today. Never assume framework defaults. If an element cannot be found, explicitly state "NOT FOUND" instead of guessing. For every claim, cite the exact file path and function/route name (`path/to/file.ext#symbol`). Perform audits in read-only mode.

