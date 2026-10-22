# pstack for zot

A [zot](https://github.com/patriceckhart/zot) extension that gives zot the skills from poteto's [pstack](https://github.com/cursor/plugins/tree/main/pstack) (originally a Cursor plugin, MIT, by Lauren Tan).

This is a **skill-only bundle**. It has no `exec` and runs no process. zot reads `extension.json`, loads `skills/`, and namespaces every skill as `pstack:<name>`.

## Install

```bash
zot ext install /path/to/zot-pstack
```

For one session without installing:

```bash
zot --ext /path/to/zot-pstack
```

The extension runs with `go run ./setup`, so Go 1.22 or later must be on PATH.

Then start a session with auto-swarm on, which `/settings` controls. Parallel pstack steps use `swarm_spawn`, and it needs auto-swarm.

## Use

Start with the main entry point:

```text
/skill:pstack:poteto-mode this pr has a subtle bug where the scroll drifts every 750ms. repro first, then fix and verify.
```

Ask for help choosing a skill:

```text
/skill:pstack:poteto-help
```

Other skills are listed in `/skills`. Each one is invoked as `/skill:pstack:<name>`.

## How it differs from the Cursor plugin

Each pstack skill was written for Cursor's tool names and features. zot provides different ones, so the bundle includes `pstack:zot-adapter`. The adapter is a skill that maps each Cursor name to its zot equivalent, and it tells the model which steps to skip. The model reads it once per session.

| Area | Cursor | zot |
|---|---|---|
| Subagents | `Task`, with named types and per-role models | `swarm_spawn`, with the model for each group set by `/pstack-setup` |
| Subagent types | `poteto-agent`, `comment-sicko`, `generalPurpose` | Prompt text in `pstack:poteto-agent-prompt` and `pstack:comment-sicko-prompt`, put at the top of each task |
| Questions | `AskQuestion` | Plain text |
| Todo list | `TodoWrite` | A list kept in the reply |
| Model roles | `~/.cursor/rules/pstack-models.mdc` | `/pstack-setup` picks a provider and one model per group (workers, judges, reviewers), and writes `$ZOT_HOME/pstack/models.json` |
| Skill names | `poteto-mode` | `pstack:poteto-mode` |
| Transcripts | `~/.cursor/projects/<slug>/agent-transcripts/` | `$ZOT_HOME/sessions/<hash>/` |

Results from `swarm_spawn` arrive later as an `[auto-swarm update]` message, not as a return value. Skills that wait on subagent output need to wait for that message.

## What is left out, and why

These parts of pstack depend on Cursor features that zot does not have. They are left out of the first pass, so the bundle does not include them.

### `benny`: Slack automations

`automations/benny/` is a pair of Cursor Automations that work a Slack issue channel. One triages each new report and replies in-thread with a `[benny:bug]`, `[benny:performance]`, or `[benny:other]` marker. The other reproduces confirmed bugs through a control adapter and may open a draft PR.

Left out because zot has no automation runner, no Slack integration, and no tracker integration. Its setup also writes into `.cursor/`.

### Bugbot triage

Bugbot is Cursor's automated code reviewer. It posts comments on GitHub PRs. The `babysit` playbook and `references/bugbot-triage.md` decide, for each Bugbot comment, whether to fix it, dismiss it, or ask the user.

Left out because it depends on Cursor's Bugbot service. The triage rules are generic, but the step that reads Bugbot's review threads is Cursor-specific. A version using `gh` could come later.

### Cloud agents

Cursor cloud agents run on Cursor's servers. pstack uses them through `environment: "cloud"` and `cloud_base_branch`, mainly in `swarm`, `orchestrate`, and `shipping`. zot has no cloud agent. Its `swarm_spawn` runs locally, so the cloud-only steps are skipped or run locally.

Left out because zot cannot start or track a cloud agent.

### `watch-pr`

`skills/poteto-mode/scripts/watch-pr/` is a TypeScript PR watcher. It polls GitHub, classifies Bugbot threads, and decides when a PR is merge-ready. It needs `bun` and its own dependencies. The `babysit` and `shipping` playbooks use it.

Left out because it needs a Bun runtime and its own dependencies, and because the babysit and shipping playbooks cannot finish without it. Its `watch-pr` test and typecheck entries were removed from `scripts/package.json`. The remaining `bun test orch check-plan.test.ts` entry still needs bun.

### `setup-pstack` and `pstack-models.mdc`, replaced by `/pstack-setup`

`setup-pstack` detects the user's models and writes a per-role model rule to `~/.cursor/rules/`. In zot, `/pstack-setup` replaces it. It picks a provider, then one model per group (workers, judges, reviewers), and writes `models.json` to `$ZOT_HOME/pstack/`. The adapter reads that file and passes each role's model to `swarm_spawn`.

This is no longer left out. `pstack-models.mdc` is not read. If `models.json` is missing, every group uses the current model.

### `make-bot-ui`

This skill builds a web page that wakes a Grok Bot through a webhook. It relies on Cursor's `update_state` routines and a Cursor-side sender key.

Left out because it depends on Cursor's routine system.

## Changes to the Cursor copy

- `skills/poteto-mode/scripts/package.json`: removed the `watch-pr` test and typecheck scripts. The `orch` and `check-plan` scripts are copied unchanged.
- `skills/zot-adapter/`: new. Maps Cursor names to zot names and lists skipped features.
- `skills/poteto-agent-prompt/`: new. The `poteto-agent` agent text, with its Cursor-specific plugin search replaced by a `pstack:poteto-mode` load.
- `skills/comment-sicko-prompt/`: new. The `comment-sicko` agent text, unchanged except for its heading.

All other skills are copied unchanged. Some still contain Cursor names such as `Task`, `AskQuestion`, and `pstack-models.mdc`. The adapter tells the model how to map them, so those files need no edits in this pass.

## Layout

```
zot-pstack/
├── extension.json      # manifest: skills and the /pstack-setup Go command
├── LICENSE
├── README.md
├── docs/guide/         # the pstack guide, read by some skills
└── skills/             # 57 skills: 54 copied from pstack, plus zot-adapter and the two prompt skills
```

## Known limits

- **Model choice.** `/pstack-setup` sets one model per group: workers, judges, and reviewers. The adapter reads `models.json` and passes each model to `swarm_spawn`. Reviewers can be a list, which runs one subagent per entry. If the file is missing, every subagent uses the current model. Cross-model verification happens only when `reviewers` lists a model from another family.
- **Subagent results.** They come back later as an `[auto-swarm update]` message. Skills that assume an immediate return value will need the adapter's guidance.
- **Namespacing.** Skill references like `../<name>/SKILL.md` and "the **x** skill" are resolved by the model through the adapter's rule, not by zot. Some skills may still name a skill without the `pstack:` prefix, and the model has to map it.
- **Subagents do not inherit `--ext`.** Swarm children are started with `--cwd` and the parent's environment. They do not get the `--ext` flag. A session that loads pstack with `--ext` only will not give pstack skills to its subagents. Install the extension with `zot ext install` so every child can find it.
- **Hidden skills are still loadable.** `disable-model-invocation` removes a skill from the model's startup manifest. The `skill` tool still loads it by name, so the adapter can name these skills. I checked this in the source. It was not run as a test.
- **`orch` and `check-plan` need bun.** The `orchestrate` playbook calls `orch` directly, and it also depends on cloud agents. Expect that playbook to fail or need rewriting.
- **Partly verified.** The bundle loads with `zot --ext`, `/skill:pstack:bro` runs, and the model manifest lists `pstack:zot-adapter`. A full `poteto-mode` run, and a real `swarm_spawn` fan-out, have not been run.

## Source

Skills are copied from pstack v0.15.29 in `cursor-plugins/pstack/`, a reference copy. It will be deleted, so this extension does not depend on it.

## License

MIT. See `LICENSE`, which is the upstream pstack license.
