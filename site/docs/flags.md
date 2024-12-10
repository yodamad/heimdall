# Global flags

Heimdall provides global flags available for all commands.

You can display options with `-h` option

```bash
heimdall -h
```

!!!example
    ```bash
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
    git-clone   Git clone given repository to a folder based on the path of the repo
    git-info    List all directories containing a `.git` folder
    help        Help about any command
    
    Flags:
    -c, --config-file string   config file (default "/Users/yodamad/.heimdall/heimdall.yml")
    -h, --help                 help for heimdall
    -l, --log-dir string       log directory (default "/Users/yodamad/.heimdall/")
    -v, --verbose              verbose output
    -w, --work-dir string      work directory (default "/Users/yodamad")
    ```

## Config file : `--config-file` or `-f`

For some operations, some information are required. For instance, for some Git instances, an authentication token is needed.

This file has to be a `yaml` formatted file.

In this config file, you can configure these elements either in hardcoded value (bad for security) or reference an environment variable.
The value pointing to an environment variable has to be prefixed by `env.`.

```bash
heimdall git-info -f /work/heimdall.yml
```

!!!example "A sample file"
    ```yaml
    work_dir: /home/johndoe/work/
    tokens:
      gitlab.mycompany.com: MY_TOKEN # Bad !!
      secured.mycompany.com: env.ENV_VAR_TOKEN
    ```

## Log directory : `--log-dir` or `-l`

## Work directory : `--work-dir` or `-w`

By default, Heimdall will run in the home directory. You can override this directory in the [configuration-file](#config-file-config-file-or-f).

But you can also define it at run time with this option. It will override the potential value existing in the configuration file.

## Verbose mode : `--verbose` or `-v`