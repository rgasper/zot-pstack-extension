---
name: zot-adapter
description: Maps pstack's Cursor tool names, subagent fields, model slugs, and paths to zot equivalents. Load before running any pstack skill or spawning any subagent in zot. Use for pstack, poteto-mode, or any pstack:* skill.
---

# zot adapter for pstack

pstack skills were written for Cursor. This file tells you how to run them in zot. Read it once per session. Then map each Cursor name in a skill with the tables below. If a Cursor field has no row, drop it and say so in one line. Never pass a field or value zot's tools reject.

pstack skills are namespaced in zot: `pstack:poteto-mode`, `pstack:why`, and so on. Load them with `/skill:pstack:<name>`, or with the `skill` tool. A skill that names another skill ("the **principle-x** skill") means `pstack:principle-x`.

## Tools

| Cursor | zot |
| --- | --- |
| `Task` | `swarm_spawn` (see Subagents). Needs auto-swarm on in `/settings`. |
| `AskQuestion` | Ask in plain text. Give the options as a numbered list and wait for the reply. zot has no question tool. |
| Open a todolist, `TodoWrite` | Keep the list in your reply and update it at each phase. zot has no todo tool. |
| `Read` | `read` |
| `Write` | `write`, or `edit` for changes to existing files |
| `StrReplace` | `edit` |
| `Shell` | `bash` |
| `Grep` | `bash` with `rg` |
| `Glob` | `glob` |
| Run another pstack skill (`/name`, or "the **name** skill") | `skill` tool with `pstack:<name>`, or tell the user `/skill:pstack:<name>` |
| `SendToUser` | Skip. Only used by Cursor automations. |
| `Browser`, `MCP` tools | Skip unless the user has provided an equivalent CLI through `bash`. |

## Subagents

zot has no named subagent types. Every subagent is a `swarm_spawn` with a task string. Its `model` and `provider` come from the group config (see Models).

| Cursor | zot |
| --- | --- |
| `subagent_type: "poteto-agent"` | `swarm_spawn`. Put the full text of the `poteto-agent-prompt` skill at the top of the task, followed by the brief. |
| `subagent_type: "comment-sicko"` | `swarm_spawn`. Put the full text of the `comment-sicko-prompt` skill at the top of the task, followed by the scope. |
| `subagent_type: generalPurpose` | `swarm_spawn`, with the brief as the whole task. |
| `readonly: true` | Add "do not edit files or run mutating commands" to the task. zot does not enforce it. Check `git status` afterward. |
| `readonly: false` | No change. |
| `environment: "cloud"` | Skip. Cloud agents are not available. Run the step locally, or stop and tell the user. |
| `environment: "local"` | Omit. |
| `cloud_base_branch` | Tell the worker in the task to check out that branch first. |
| `run_in_background: true` | Omit. `swarm_spawn` already returns at once. |
| `resume` | Spawn a new agent, and paste the earlier result into its task. |
| `file_attachments` | List the paths in the task. |
| `model` set to a Cursor slug | Do not pass the slug. Pass `model` and `provider` from the config for the role's group (see Models). Omit both when the group has no value. |

Results from `swarm_spawn` do not come back as a return value. They arrive later as an `[auto-swarm update]` message in the chat, which recaps each agent's status and final response. Wait for that message before you use the results. Do not invent results while you wait.

## Models

Role models come from the group config that `/pstack-setup` writes. It is a zot command, and it replaces the Cursor `setup-pstack` skill and the `~/.cursor/rules/pstack-models.mdc` role file. Cursor model slugs (`claude-opus-5-5-xhigh`, `grok-4.7-xhigh-fast`, `gpt-*`) do not apply in zot.

Load the config once per session, before the first `swarm_spawn`:

1. Resolve the path. If `ZOT_HOME` is set, use `$ZOT_HOME/pstack/models.json`. Otherwise use the platform default:
   - macOS: `~/Library/Application Support/zot/pstack/models.json`
   - Linux: `~/.local/state/zot/pstack/models.json`
   - Windows: `%LOCALAPPDATA%\zot\pstack\models.json`
2. Read it with `read`. Its shape is:

```json
{
  "provider": "amazon-bedrock",
  "workers": "anthropic.claude-haiku-5-5",
  "judges": "anthropic.claude-opus-5-5",
  "reviewers": ["anthropic.claude-opus-5-5", "global.xai.grok-4.7"]
}
```

3. If the file is missing, every group inherits the parent model. Say so once in your reply, then omit `model` and `provider` on every spawn.

| Group | Roles | Spawns |
| --- | --- | --- |
| `workers` (single model) | feature, refactoring, bug-fix, perf-issue, hillclimb, swarm workers, how explorer, why investigators, reflect tooling | One `swarm_spawn` with `workers` as `model`. |
| `judges` (single model) | judgment and prose, hardest tasks, how explainer, why synthesizer, reflect judgment, reflect divergent, reflect synthesizer, arena cross-judge pool | One `swarm_spawn` with `judges` as `model`. |
| `reviewers` (list) | arena runners, architect runners, interrogate reviewers | One `swarm_spawn` per entry in the list. |

- Always pass `model` and `provider` together, from the same config. If `provider` is missing or empty, treat every group as having no value: omit both fields on every spawn, even when `workers`, `judges`, or `reviewers` are set. For `workers` and `judges`, a missing key means omit both.
- For `reviewers`, spawn one `swarm_spawn` per entry, each with that entry's `model` and `provider`. Do not spawn them as one call. An empty or missing list means run one subagent on the inherited model, with both omitted.
- If a step asks for a different model family for verification, use a `reviewers` entry from another family. If no entry qualifies, run the verification on the same family.

## Paths

| Cursor | zot |
| --- | --- |
| `.cursor/skills/`, `~/.cursor/skills/` | `.zot/skills/`, `~/.zot/skills/`, or `$ZOT_HOME/skills/` |
| `~/.cursor/plugins/` | The zot extension directory: `$ZOT_HOME/extensions/pstack/`. Use `zot ext list` to find it. |
| `.cursor/rules/*.mdc`, always-applied rules | `AGENTS.md` in the project, or `$ZOT_HOME/AGENTS.md` globally. |
| `~/.cursor/projects/<slug>/agent-transcripts/` | `$ZOT_HOME/sessions/<hash>/`. Sessions are `.jsonl` files named by timestamp. The hash is the first 8 bytes of sha256 of the cwd, in hex. Read only this cwd's folder. Do not glob the whole sessions directory. |
| `https://github.com/cursor/plugins/blob/main/pstack/<path>` | The local copy at `docs/guide/` in the extension directory, or the local `README.md`. Never fetch it. |
| `docs/guide/...`, `../<name>/SKILL.md` | Same paths relative to the extension's `skills/` directory. Skill references resolve inside this bundle. |

## Cursor-only features

These are not in this bundle and cannot run in zot. If a step needs one, write one line, `Skipped in zot: <step>. <reason>.`, and go on. Do not stop or retry.

- **Cursor Automations**, including the `benny` Slack automations. Not included.
- **Bugbot** comment triage. Not included. The GitHub review threads are ordinary PR comments, and `gh` can read them if the user asks.
- **Cloud agents**, including `environment: "cloud"` and `cloud_base_branch`.
- **`watch-pr`** (TypeScript PR watcher). Not included.
- **`make-bot-ui`**, which relies on Cursor's `update_state` routines. Not included.

Use `gh` for pull requests: `gh pr create`, `gh pr view`, `gh pr checks`. That is not a skip, so write no line for it.

## Principle skills

The `principle-*` skills are leaf skills. Apply one by loading `pstack:principle-<name>` when its trigger matches. Do not load them all at once.
