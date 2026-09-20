# Historic Agent Skill

The repository includes a shareable agent skill at:

```text
.agents/skills/historic/SKILL.md
```

It contains the workspace rules, command contracts, sync-meta reconciliation behavior, TUI conventions, validation patterns, and coordination workflow needed for an agent to work safely with Historic.

## Install for a local agent

Copy the skill into the agent's skills directory:

```sh
mkdir -p ~/.agents/skills/historic
cp .agents/skills/historic/SKILL.md ~/.agents/skills/historic/SKILL.md
```

For the VS Code/Copilot environment used by this project, the destination may be:

```text
~/.agents/skills/historic/SKILL.md
```

After copying, start a new agent session if the host loads skills only at session start.

## Keep the skill current

The repository copy is the distributable version. Update it when command behavior or safety rules change, especially when changing:

- the canonical `.historic/` workspace;
- `sync-meta` targeted, batch, or reconciliation behavior;
- `historic find` or `historic search` contracts;
- JSON output envelopes;
- lifecycle/archive/import/restore safety;
- SPEC and Work Order coordination rules.

The global copy is an installed copy and should not be treated as the repository source of truth.
