# docs/features

This file covers `docs/features/`: one directory per feature, and the
files in it.

## Voice

Two terms set the voice of each page below.

- **STE** is the ASD-STE100 writing rules: one idea per sentence, 20
  words maximum in an instruction, 25 in a description, active voice,
  one word for one meaning. Use the rules, not the approved word list;
  technical names such as Go, Ohm and CST are exempt.
- **EF support** is executive function support: lead with the answer or
  the next action, put a blocker in the first line, and give one next
  action, not a menu.

## The directory

**A feature is** `NNN-<name>/`. `NNN` is a number of three digits,
zero padded: `001-error-reporting`. A new feature takes the number
after the highest in this directory. The numbers follow the order of
each feature's first commit, so the listing is the timeline. A number
is never used again, even after its feature is merged into another or
removed. The name is kebab-case. Neither changes after the first
commit; a code comment cites the name. Every file in it is optional:

| file               | a page is                                                 |
|--------------------|-----------------------------------------------------------|
| `prd.md`           | the argument: what to build and why                       |
| `prd-<part>.md`    | a second argument in the same feature, with the same plan |
| `plan.md`          | the checklist for the PRD                                 |
| `arch.md`          | how the built feature works, and where                    |
| `notes-<topic>.md` | an analysis of this feature that decides nothing          |

**Two branches, one number.** Two branches can take the same number.
The branch that merges second renumbers its feature, before the merge,
to the next free number.

**The index.** [README.md](README.md) has one row per feature. A new
feature gets its row in the commit that creates the directory. A
commit that adds or removes a file in a feature updates its row.

**A code comment** cites a page by name, kind and section:
`error-reporting prd section 4`, `error-reporting arch section 2`.

## `prd.md`

**Reader.** An engineer and a product manager, deciding together.

**A page is** the argument for one thing to build. It opens with what
it specifies, in one paragraph, then **Decisions at a glance**: a table
of concern, decision and runners-up. Sections are numbered so the plan
can cite them. A status paragraph near the top says what is built. Open
decisions are listed, each with the `docs/kb/` page that analyses it.

**Blockers come first.** A finding that must change before the work can
start goes in `## 0. Blockers`, before Decisions at a glance: what is
wrong, what fixes it, the command that checked it, and the date. An item
leaves in the commit that lands it.

**Voice.** STE off; the argument needs room. EF support on.

## `plan.md`

**Reader.** An engineer or an agent that does the work, one phase at a
time.

**A page is** the checklist for the PRD beside it. One paragraph that
links `prd.md`, says "that document is the argument; this one is the
work", and states the tick rule. Then **Legend**, **Before starting**,
one `## Phase N — name [sections]` per phase, each ending with
**Done when** and **Documentation**, and **Not in any phase**. One
bullet names one change in one file, with the file first.

**Voice.** STE on. EF support on: the next unticked bullet is visible
from the top.

## `arch.md`

**A page is** how the built feature works, and where. The map comes
first. Sections are numbered, and are never renumbered, so a code
comment that cites one stays true. No links to source code; name the
package, file or function instead. A page that two features cite moves
to `docs/architecture/` under the feature's name, and `prd.md` links it
there.

## The other files

`notes-<topic>.md` decides nothing, and says so in the first paragraph.
A page about this repository's own design goes here, not in `docs/kb/`;
`kb/` takes research into external sources only, such as Ohm.js.

## Not here

A page for the user of `ohm-cli` goes to `ohm-cli/docs/`. A page for
the user of the `ohm` package goes to `ohm/README.md`. Research into an
external source goes to `docs/kb/`. A page about the whole system goes
to `docs/architecture/`.

## In the same commit

A bullet is ticked, and the PRD's status paragraph moves, in the commit
that lands the code. Work on a phase lands in chunks, and a commit can
come after any of them, so tick in the chunk that lands the code. Do
not save the ticks for a closing docs chunk. Tick a bullet that is
partly done as `[~]`, and say what is missing. Add a ticked bullet for
work that landed and that the plan did not name, such as a bug fixed on
the way. A decision that changes `arch.md` or a page in
`docs/architecture/` changes that page too.

## Who writes here

Anyone.
