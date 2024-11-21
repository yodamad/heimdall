# Heimdall

A CLI to help with your git directories (for now 😉).

Based on the myth of the Nordic God, [Heimdall](https://en.wikipedia.org/wiki/Heimdall), the CLI is here to ease with your multiple Git repositories.

A quick demo of the interactive mode.

![Simple demo](./assets/demo.gif)

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

## Available options

You can display options with `-h` option

```bash
heimdall -h
```

```text
            _               _       _ _
  /\  /\___(_)_ __ ___   __| | __ _| | |
 / /_/ / _ \ | '_ ` _ \ / _` |/ _` | | |
/ __  /  __/ | | | | | | (_| | (_| | | |
\/ /_/ \___|_|_| |_| |_|\__,_|\__,_|_|_|

Version dev

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
  -i, --i                 interactive mode
  -l, --log-dir string    log directory (default ".")
  -r, --root-dir string   root directory (default ".")
  -v, --verbose           verbose output

Use "heimdall [command] --help" for more information about a command.
```

### `git-info` or `gi`

This option helps you with your git repositories. It will list them and tell you if they are up-to-date or not.

```shell
  /\  /\___(_)_ __ ___   __| | __ _| | |
 / /_/ / _ \ | '_ ` _ \ / _` |/ _` | | |
/ __  /  __/ | | | | | | (_| | (_| | | |
\/ /_/ \___|_|_| |_| |_|\__,_|\__,_|_|_|

Version dev

Usage:
  heimdall git-info [flags]

	Aliases:
  git-info, gi

Flags:
  -d, --depth int   search depth (default 3)
  -h, --help        help for git-info

Global Flags:
  -i, --i                 interactive mode
  -l, --log-dir string    log directory (default ".")
  -r, --root-dir string   root directory (default ".")
  -v, --verbose           verbose output
```

The command do a local and a remote checks.

By default, it will search in the current directory, but you can override this with `-r` flag. Also, it searches no more then 3 levels of subdirectories, you can override this with the `-d` flag.

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

### `-i` : Interactive mode

With interactive mode, you can easily:
* Pick the folder you want to inspect
* Display local changes of a picked folder after analyzing
* Display remote commits of a picked folder after analyzing
* (soon) Update one or several folders

```bash
heimdall git-info -i
```

```shell
            _               _       _ _
  /\  /\___(_)_ __ ___   __| | __ _| | |
 / /_/ / _ \ | '_ ` _ \ / _` |/ _` | | |
/ __  /  __/ | | | | | | (_| | (_| | | |
\/ /_/ \___|_|_| |_| |_|\__,_|\__,_|_|_|

Version 0.0.4

🔍 Search in directory /Users/admin_local/work/gitlab/fun-with/fun-with-k8s [Y/n] :
Searching in '/Users/admin_local/work/gitlab/fun-with/fun-with-k8s' ...
⚠️ Error analyzing /Users/admin_local/work/gitlab/fun-with/fun-with-k8s/external-api, skip it...
...
Found 4 folder(s) (Skip 1 folders because of errors, use '-v' to check in details)
+------------------------------------------------------------------------------------------+--------+---------------+----------------+
| PATH                                                                                     | BRANCH | LOCAL_CHANGES | REMOTE_CHANGES |
+------------------------------------------------------------------------------------------+--------+---------------+----------------+
| /Users/admin_local/work/gitlab/fun-with/fun-with-k8s/external-dns                        |  main  |       🔴      |       🟢       |
+------------------------------------------------------------------------------------------+--------+---------------+----------------+
| /Users/admin_local/work/gitlab/fun-with/fun-with-k8s/fun-with-fluxcd                     |  main  |       🟢      |       🟢       |
+------------------------------------------------------------------------------------------+--------+---------------+----------------+
| /Users/admin_local/work/gitlab/fun-with/fun-with-k8s/fun-with-kyverno                    |  main  |       🔴      |       🟢       |
+------------------------------------------------------------------------------------------+--------+---------------+----------------+
| /Users/admin_local/work/gitlab/fun-with/fun-with-k8s/fun-with-vault-and-external-secrets |  main  |       🟢      |       🟢       |
+------------------------------------------------------------------------------------------+--------+---------------+----------------+
...
Interactive mode options:
[X] 📤 Display local changes of a repository
[ ] 🔃 Update one or several repositories (git pull)
[ ] ✅ I'm done
Pick one:
[X] /Users/admin_local/work/gitlab/fun-with/fun-with-k8s/external-dns
[ ] /Users/admin_local/work/gitlab/fun-with/fun-with-k8s/fun-with-kyverno
...
🚦 1 files
hashnode-demo.yaml - M
...
What to do next::
[X] 🔄 Check another folder
[ ] ✅ I'm done
...
Interactive mode options:
[ ] 📤 Display local changes of a repository
[ ] 🔃 Update one or several repositories (git pull)
[X] ✅ I'm done
```

### `-f`, `--config-file` : Input config file

For some operations, some information are required. For instance, for some Git instances, an authentication token is needed.

This file has to be a `yaml` formatted file.

In this config file, you can configure these elements either in hardcoded value (bad for security) or reference an environment variable.
The value pointing to an environment variable has to be prefixed by `env.`.

```bash
heimdall git-info -f /work/heimdall.yml
```

A sample file

```yaml
tokens:
  gitlab.mycompany.com: MY_TOKEN # Bad !!
  secured.mycompany.com: env.ENV_VAR_TOKEN
```