# Global flags

Heimdall provides global flags available for all commands.

You can display options with `-h` option

![Simple demo](./assets/heimdall-demo.gif)

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

By default, Heimdall will write its logs into a file call `heimdall.log` in the default directory.

You can override this value with this flag to specify the path of your log file locally.

## Work directory : `--work-dir` or `-w`

By default, Heimdall will run in the home directory. You can override this directory in the [configuration-file](#config-file-config-file-or-f).

But you can also define it at run time with this option. It will override the potential value existing in the configuration file.

## Verbose mode : `--verbose` or `-v`

By enabling this option, more logs are traced within the log file and in the console output. This helps to debug if you face some problems running Heimdall.