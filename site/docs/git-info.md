# Git Info : `git-info` or `gi`

This option helps you with your git repositories. It will list them and tell you if they are up-to-date or not.
The command do a local and a remote checks.

Some options are available and described in this page. But there are also [global flags](flags.md) available.

![Simple demo](./assets/heimdall-git-info-demo.gif)

While the directories are being searched, the line being refreshed tells how many repositories were found so far and where the search is, `🔍 7 repositories found, scanning ~/work/group`. The count grows instead of being an `x of y` : how many there are is only known once the search is over. Each repository found is kept on a line of its own, so that the search is never a silent wait.

## Available options

![Options](./assets/heimdall-git-info-help.gif)

### Search depth: `--depth` or `-d`

By default, it searches no more then 3 levels of subdirectories, you can override this with the `-d` flag.

![Simple demo](./assets/heimdall-git-info-demo-d.gif)

### Interactive mode: `--interactive-mode` or `-i`

With interactive mode, you can easily:
* Pick the folder you want to inspect
* Display local changes of a picked folder after analyzing
* Display remote commits of a picked folder after analyzing
* (soon) Update one or several folders

```bash
heimdall git-info -i
```

!!!example "Sample with interactive mode"
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
