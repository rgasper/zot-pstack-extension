# pstack for zot

Skills from poteto's [pstack](https://github.com/cursor/plugins/tree/main/pstack), packaged for [zot](https://github.com/patriceckhart/zot). pstack is a Cursor plugin, MIT Licensed, by Lauren Tan.

## Setup

Run `/pstack-setup` in zot first. It lets you pick a model provider, then which models to use for each role group: workers, judges, and reviewers. It saves the choices to `$ZOT_HOME/pstack/models.json`. To change providers or models later, run `/pstack-setup` again.

Install the extension:

```bash
zot ext install /path/to/zot-pstack
```

Or load it for one session:

```bash
zot --ext /path/to/zot-pstack
```

Please note that if loaded using `--ext`, pstack will not function correctly for swarm agents, only the top-level agent.

Requirements:

- zot on PATH, for `/pstack-setup`'s model list. The extension ships prebuilt binaries for darwin and linux, amd64 and arm64, so Go is not needed to install or run it.
- Auto-swarm must be **on**, set in `/settings`. Parallel steps use `swarm_spawn`.

## Use

Start a task in poteto mode:

```text
/poteto-mode this pr has a subtle bug where the scroll drifts every 750ms. repro first, then fix and verify.
```

`/poteto-mode` picks the playbook that fits your request and says which one. Run `/poteto-mode` with no request to enter the mode and wait for your next message.

Get help choosing a skill:

```text
/skill:pstack:poteto-help
```

Other skills are listed in `/skills`. Run one as `/skill:pstack:<name>`.

## Not included

We're not using these Cursor-specific features:

- Benny (Slack automations)
- Bugbot triage
- Cloud agents
- `watch-pr`
- `make-bot-ui`
- `setup-pstack` (replaced by `/pstack-setup`)

## How it differs from Cursor

`pstack:zot-adapter` maps Cursor's names to zot's. The model reads it once per session.

| Area | Cursor | zot |
|---|---|---|
| Subagents | `Task`, with named types and per-role models | `swarm_spawn`, with the model for each group set by `/pstack-setup` |
| Subagent types | `poteto-agent`, `comment-sicko`, `generalPurpose` | Prompt text in `pstack:poteto-agent-prompt` and `pstack:comment-sicko-prompt` |
| Questions | `AskQuestion` | Plain text |
| Todo list | `TodoWrite` | A list in the reply |
| Model roles | `~/.cursor/rules/pstack-models.mdc` | `$ZOT_HOME/pstack/models.json` |
| Skill names | `poteto-mode` | `pstack:poteto-mode` |

Subagent results arrive later as an `[auto-swarm update]` message, not as a return value.

## Known limits

- With no `models.json`, every subagent uses the current model.
- Subagents do not inherit `--ext`. Install with `zot ext install` so they can find the skills.
- `orchestrate` needs bun and cloud agents, so it will not run here.
- Some skills name other skills without the `pstack:` prefix. The model maps these using the adapter.

## Layout

```
zot-pstack/
├── extension.json      # manifest: skills, and the launcher that starts /pstack-setup
├── Makefile            # make build | test | grab-original | clean
├── bin/                # prebuilt pstack-setup binaries, one per platform
├── setup/              # Go source for /pstack-setup and /poteto-mode
├── skills/             # pstack skills, plus zot-adapter
└── docs/guide/         # the pstack guide, read by some skills
```

## License

MIT. See `LICENSE`.
