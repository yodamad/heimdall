# Welcome to Heimdall

A CLI to help with your git directories (for now 😉).

Based on the myth of the Nordic God, [Heimdall](https://en.wikipedia.org/wiki/Heimdall), the CLI is here to ease with your multiple Git repositories.

For now, Heimdall has 2 main commands :

- [Git-info](git-info.md) to help you manage all your git repositories and now their current branch, if they have some local changes or if they are behind the remote repository
- [Git-clone](git-clone.md) to clone a git repository and keep the same path 

Run without any command, Heimdall opens a [TUI](tui.md) to navigate through your git repositories and run actions on one or several of them.

## How to install

__*On MacOS:*__

Heimdall is available through `brew`

```bash
brew tap yodamad/tools
brew install heimdall
```

__*On Linux:*__ ⚠️ Use it at your own risk *for now* ⚠️

There are available on [Release page](https://github.com/yodamad/heimdall/releases) but not well tested to be honest

__*On Windows:*__ ❌ Not available for now, some compatibilities problems.

!!!tip "WSL2 option"
    It works with the linux version

## How to configure

All the configuration elements are detailed in the [configuration file](config.md) section.

You can override this value with the `--config-file` or `-f` flag defined in the [global flags](flags.md#config-file----config-file-or--f).
