# TUI : `heimdall` or `tui`

Running `heimdall` without any command (or `heimdall tui`, `heimdall ui`) opens a full screen interface to navigate through your git repositories, see their state and run actions on one or several of them.

```bash
heimdall -w ~/work
```

The list shows the repositories found in the [work directory](flags.md#work-directory----work-dir-or--w), grouped by folder, the ones needing attention first in each folder. Next to it are the details of the current repository : what state it is in and what you can do about it, the output of the last commands run on it, its changed files, its incoming and not pushed commits, its branches and its last commits.

## Reading the list

The state of each repository is written in plain words on its right, in a color telling how urgent it is.

| Color | State |
|---|---|
| Red | Diverged from origin, or the repository can't be read |
| Orange | Behind origin |
| Yellow | Local changes |
| Green | Commits not pushed |
| Gray | Up to date |

!!!info "No network at startup"
    To be displayed quickly, the list is built from local information only, so what it knows about origin dates from the last fetch. When this fetch is more than a day old, the details tell its age : press ++f++ to check origin.

The other commands (`git-info`, `git-clone`, `good-morning`, `env-info`) are still available for a non-interactive usage.

## Navigation

| Key | Action |
|---|---|
| ++up++ / ++down++ or ++j++ / ++k++ | Move |
| ++g++ / ++shift+g++ | First / last repository |
| ++tab++ | Go to the details, ++tab++ again to go back |
| ++enter++ or ++left++ / ++right++ | Collapse / expand the folder |
| ++z++ | Collapse / expand all the folders |
| ++slash++ | Filter repositories by path |
| ++esc++ | Clear the filter, then the selection |
| ++question++ | Help |
| ++q++ | Quit |

## Selection

Actions are run on the selected repositories or, if none is selected, on the current one. With the cursor on a folder, they are run on all its repositories.

A collapsed folder tells what its repositories need, and the details list them all by state, the ones needing attention first.

| Key | Action |
|---|---|
| ++space++ | Select / unselect the current repository, or all the ones of the current folder |
| ++a++ | Select / unselect all the displayed repositories |
| ++u++ | Select the repositories which can be pulled |

## Actions

A confirmation is asked before pulling or running commands on several repositories.

| Key | Action |
|---|---|
| ++r++ | Refresh local status |
| ++f++ | `git fetch` |
| ++p++ | `git pull`, skipped for repositories with local changes |
| ++m++ | Run the morning routine defined in the [configuration file](config.md) |
| ++exclam++ | Run a command |

## Branches

When a repository has several local branches, the details list them : the one the repository is on first, marked with a star, then the most recently committed ones. Each tells where it stands compared to origin and how old its last commit is, so that work left on a branch which was never pushed is not forgotten.

Press ++tab++ to go to the details, then :

| Key | Action |
|---|---|
| ++up++ / ++down++ or ++j++ / ++k++ | Choose a branch |
| ++enter++ | Switch to the chosen branch with `git switch` |
| ++page-up++ / ++page-down++ | Scroll the details |

When git refuses to switch, because of local changes for instance, the details tell why.

## Available options

### Search depth: `--depth` or `-d`

By default, it searches no more then 3 levels of subdirectories, you can override this with the `-d` flag of the `tui` command.

```bash
heimdall tui -d 5
```
