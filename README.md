# Heimdall

A CLI to help with your git directories (for now 😉).

Based on the myth of the Nordic God, [Heimdall](https://en.wikipedia.org/wiki/Heimdall), the CLI is here to ease with your multiple Git repositories.

## How to install

__*On MacOS:*__

Heimdall is available through `brew`

```bash
brew tap yodamad/tools
brew install heimdall
```

__*On others:*__ ⚠️ Use it at your own risk *for now* ⚠️

There are available on [Release page](https://github.com/yodamad/heimdall/releases) but not well tested to be honest

## Available options

You can display options with `-h` option

```bash
heimdall -h
```

```text
Heimdall is a CLI tool to help you with your git folders.
You can check, update, ... everything easily


Usage:
  heimdall [flags]
  heimdall [command]

Examples:
heimdall -h

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  git-info    List all directories containing a `.git` folder
  help        Help about any command

Flags:
  -h, --help              help for heimdall
  -r, --root-dir string   root directory (default ".")
  -v, --verbose           verbose output

Use "heimdall [command] --help" for more information about a command.
```

### `git-info` or `gi`

This option helps you with your git repositories. It will list them and tell you if they are up-to-date or not.

The command do a local and a remote checks

```bash
heimdall git-info -r /home/user/work/
```

```shell
Searching in /home/user/work/...
+---------------------------------------+--------+---------------+----------------+
| PATH                                  | BRANCH | LOCAL_CHANGES | REMOTE_CHANGES |
+---------------------------------------+--------+---------------+----------------+
| /home/user/work/project1              |  main  |       🔴      |      🔴(1)     |
+---------------------------------------+--------+---------------+----------------+
| /home/user/work/project2              |  main  |       🔴      |       🟢       |
+---------------------------------------+--------+---------------+----------------+
| /home/user/work/project3              | master |       🟢      |       🟢       |
+---------------------------------------+--------+---------------+----------------+
| /home/user/work/project4              |  main  |       🔴      |       🟢       |
+---------------------------------------+--------+---------------+----------------+
```
