# Git clone : `git-clone` or `gc`

`git-clone` command clone the given URL into a directory that has the same path. 

For example, if you want to clone `https://github.com/yodamad/heimdall`, it will be cloned into `<work_dir>/yodamad/heimdall`.

Several options are available to customize the created path and [global flags](flags.md) are also available.

!!!example "Usage"
    ```bash
      /\  /\___(_)_ __ ___   __| | __ _| | |
     / /_/ / _ \ | '_ ` _ \ / _` |/ _` | | |
    / __  /  __/ | | | | | | (_| | (_| | | |
    \/ /_/ \___|_|_| |_| |_|\__,_|\__,_|_|_|
    
    Version dev
    
    Usage:
      heimdall git-clone [flags]
    
        Aliases:
      git-clone, gc
    
    Flags:
      -h, --help                   help for git-clone
      -i, --include-hostname       Include hostname in path created ?
      -k, --keep-hostname-suffix   Include hostname suffix (.com, .fr,...) in path created ?
    
    Global Flags:
    ...
    ```

## Include hostname: `--include-hostname` or `i`

Enabling this option will keep in the path created the hostname of the URL to be cloned, removing the suffix from it.

For example, running `heimdall git-clone -i https://github.com/yodamad/heimdall` will clone into `<work_dir>/github/yodamad/heimdall`.

## Keep hostname suffix: `--keep-hostname-suffix` or `k`

Enabling this option will keep in the suffix of the hostname of the URL to be cloned.

For example, running `heimdall git-clone -i -k https://github.com/yodamad/heimdall` will clone into `<work_dir>/github.com/yodamad/heimdall`.

!!!warning
    The option will be ignored if the `--include-hostname` option is not set