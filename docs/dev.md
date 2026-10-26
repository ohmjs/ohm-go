# Developing ohm-go

This page is for someone who changes this repository. It covers the
workspace, the dev shell, the tests, and the release steps that
`Taskfile.yml` automates. A user of `ohm-cli` reads `ohm-cli/README.md`
and `ohm-cli/docs/`; a user of the `ohm` package reads `ohm/README.md`.

## The workspace

ohm-go holds three Go modules. `go.work` at the root joins them, so a
change to `ohm` is seen by `ohm-cli` at once, with no tag.

| module     | path                                 | published as         |
|------------|--------------------------------------|----------------------|
| `ohm`      | `github.com/ohmjs/ohm-go/ohm`        | tag `ohm/v<version>` |
| `ohm-cli`  | `github.com/ohmjs/ohm-go/ohm-cli`    | tag `ohm-cli/v<version>` |
| `examples` | `github.com/ohmjs/ohmgo-examples`    | not published        |

Outside this workspace, `ohm-cli/go.mod` and `examples/go.mod` name the
`ohm` and `ohm-cli` versions they build against. A consumer, and a nix
build, see only those versions. That is why a release has two parts:
tag the module, then bump the `go.mod` files that require it.

## The dev shell

`flake.nix` gives a shell with `go`, `adl` and `task`. `.envrc` loads it
through direnv; without direnv, run `nix develop`. Docker is not in the
shell. The txtar fixtures that compile a grammar need it, so start it
before the tests.

## The tasks

Run `task --list` for the summary. Each task runs from the repository
root; `task` finds `Taskfile.yml` from any subdirectory.

### `task test`

Runs `go test ./...` in `ohm-cli`, then every txtar fixture with
`--strict`. Strict means a fixture that did not run, such as one whose
docker compile failed, fails the task. Run it before a commit. The
fixtures are described in `ohm-cli/docs/txtar-tests.md`.

### `task tag:list`

Shows the latest tag of `ohm` and of `ohm-cli`, the commit it points
at, and whether origin has it. Run it first, to see where a release
stands.

### `task tag:ohm VERSION=1.1.0`, `task tag:ohm-cli VERSION=1.1.0`

Makes the annotated tag `ohm/v1.1.0`, or `ohm-cli/v1.1.0`, at HEAD,
with the message `release`. `VERSION` is the number only, as
`major.minor.patch`, without the `v`. The task stops when:

- the tag exists;
- `VERSION` is missing or not three numbers;
- the module directory has uncommitted changes. Commit them first, so
  the tag names what you tested.

It does not push. It prints the push command, and you run it:

```sh
git push origin ohm/v1.1.0
```

Tag at the commit you mean to publish. The tag is at HEAD, so check
out that commit, or make sure HEAD is it, before you run the task.

### `task gomod:update-ohm [VERSION=1.1.0]`

Moves the `ohm` require of `ohm-cli/go.mod` and `examples/go.mod` to
`ohm/v<VERSION>`. Without `VERSION` it takes the latest local `ohm/v*`
tag. For each module it runs `go get` and `go mod tidy`, with
`GOWORK=off` so the one module is edited, and `GOPROXY=direct` so the
fetch does not wait for the module proxy. Then it runs `go work sync`
and prints the changed files. Commit them.

The tag must be on origin. `go get` fetches it from there, so the task
stops when it is not, and prints the push command. Push, then run the
task again.

### `task gomod:update-ohm-cli [VERSION=1.1.0]`

The same for the `ohm-cli` require of `examples/go.mod`.

## A release, start to end

Release `ohm` first. `ohm-cli` requires it, so the `ohm-cli` tag must
point at a commit whose `go.mod` names the new `ohm` version.

```sh
task test                           # with docker running
task tag:ohm VERSION=1.1.0
git push origin ohm/v1.1.0
task gomod:update-ohm               # ohm-cli and examples require ohm v1.1.0
git commit -am 'ohm-cli, examples: require ohm v1.1.0'
task tag:ohm-cli VERSION=1.1.0
git push origin ohm-cli/v1.1.0
task gomod:update-ohm-cli           # examples requires ohm-cli v1.1.0
git commit -am 'examples: require ohm-cli v1.1.0'
git push
```

Skip the `ohm-cli` half when only `ohm` changed, and the `ohm` half
when only `ohm-cli` changed.

A consumer that pins ohm-go through nix, such as sheafdb, vendors
`ohm-cli` with `GOWORK=off`. It builds against the `ohm` version in
`ohm-cli/go.mod`, not the checkout's `ohm/`. So a change to `ohm` that
`ohm-cli` uses is not buildable there until the `ohm` tag is pushed and
`ohm-cli/go.mod` names it. Do the `ohm` half before such a consumer
bumps its pin.

## When something stops

| message | cause | do |
|---|---|---|
| `VERSION must be major.minor.patch` | `VERSION` missing or short | `task tag:ohm VERSION=1.1.0` |
| `VERSION is the number only, without the v` | `VERSION=v1.1.0` | drop the `v` |
| `ohm/v1.1.0 exists, at abc1234` | tagged already | `task tag:list`, then push it or pick the next version |
| `ohm/ has uncommitted changes` | the module is dirty | commit, then tag |
| `ohm/v1.1.0 is not on origin` | tag made, not pushed | `git push origin ohm/v1.1.0`, then run the task again |
| `go get` cannot find the version | the push is seconds old | run the task again |
