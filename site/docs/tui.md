# TUI : `heimdall` or `tui`

Running `heimdall` without any command (or `heimdall tui`, `heimdall ui`) opens a full screen interface to navigate through your git repositories, see their state and run actions on one or several of them.

```bash
heimdall -w ~/work
```

The left pane lists the repositories found in the [work directory](flags.md#work-directory----work-dir-or--w), the right one gives the details of the current repository : branch, remote, local changes, incoming commits and the output of the last commands run on it.

!!!info "No network at startup"
    To be displayed quickly, the list is built from local information only. Remote changes are computed from the last fetch and are displayed with a `~` until the repository is fetched from the TUI with ++f++.

The other commands (`git-info`, `git-clone`, `good-morning`, `env-info`) are still available for a non-interactive usage.

## Navigation

| Key | Action |
|---|---|
| ++up++ / ++down++ or ++j++ / ++k++ | Move |
| ++g++ / ++shift+g++ | First / last repository |
| ++tab++ | Focus the details pane to scroll in it, ++tab++ again to go back |
| ++slash++ | Filter repositories by path |
| ++esc++ | Clear the filter, then the selection |
| ++question++ | Help |
| ++q++ | Quit |

## Selection

Actions are run on the selected repositories or, if none is selected, on the current one.

| Key | Action |
|---|---|
| ++space++ | Select / unselect the current repository |
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

## Available options

### Search depth: `--depth` or `-d`

By default, it searches no more then 3 levels of subdirectories, you can override this with the `-d` flag of the `tui` command.

```bash
heimdall tui -d 5
```
