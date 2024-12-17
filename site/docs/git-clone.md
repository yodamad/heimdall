# Git clone : `git-clone` or `gc`

`git-clone` command clone the given URL into a directory that has the same path. 

For example, if you want to clone `https://github.com/yodamad/heimdall`, it will be cloned into `<work_dir>/yodamad/heimdall`.

Several options are available to customize the created path and [global flags](flags.md) are also available.

![Demo](./assets/heimdall-git-clone-demo.gif)

## Available options

![Options](./assets/heimdall-git-clone.gif)

### Include hostname: `--include-hostname` or `i`

Enabling this option will keep in the path created the hostname of the URL to be cloned, removing the suffix from it.

For example, running `heimdall git-clone -i https://github.com/yodamad/heimdall` will clone into `<work_dir>/github/yodamad/heimdall`.

![Demo](./assets/heimdall-git-clone-demo-i.gif)

### Keep hostname suffix: `--keep-hostname-suffix` or `k`

Enabling this option will keep in the suffix of the hostname of the URL to be cloned.

For example, running `heimdall git-clone -i -k https://github.com/yodamad/heimdall` will clone into `<work_dir>/github.com/yodamad/heimdall`.

!!!warning
    The option will be ignored if the `--include-hostname` option is not set

![Demo](./assets/heimdall-git-clone-demo-k.gif)