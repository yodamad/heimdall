# Good Morning : `good-morning` or `gm`

This option will help you run your morning routine on several git repositories quite easily.

Some options are available and described in this page. But there are also [global flags](flags.md) available.

![Simple demo](./assets/heimdall-good-morning-demo.gif)

## Available options

![Options](./assets/heimdall-good-morning-help.gif)

### Search depth: `--depth` or `-d`

By default, it searches no more then 3 levels of subdirectories, you can override this with the `-d` flag.

### Force mode: `--force` or `-f`

By default, heimdall will ask you to confirm before running morning routine.
Setting `-f` flag will force the execution

```bash
heimdall good-morning -f
```
