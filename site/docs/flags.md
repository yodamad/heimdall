# Global flags

Heimdall provides global flags available for all commands.

You can display options with `-h` option

![Simple demo](./assets/heimdall-help.gif)

## Config file : `--config-file` or `-f`

By default, Heimdall will look for a configuration file in the default config directory with the name `heimdall.yml`.

You can override this value with this flag to specify the path of your configuration file locally.

```bash
heimdall git-info -f /work/heimdall.yml
```

## Log directory : `--log-dir` or `-l`

By default, Heimdall will write its logs into a file call `heimdall.log` in the default directory.

You can override this value with this flag to specify the path of your log file locally.

## Work directory : `--work-dir` or `-w`

By default, Heimdall will run in the home directory. You can override this directory in the [configuration-file](#config-file-config-file-or-f).

But you can also define it at run time with this option. It will override the potential value existing in the configuration file.

## Verbose mode : `--verbose` or `-v`

By enabling this option, more logs are traced within the log file and in the console output. This helps to debug if you face some problems running Heimdall.

## No color mode : `--no-color` or `-n`

By default, Heimdall will display colored output. 
If you want to disable this feature, you can use this flag.