# Good Morning : `good-morning` or `gm`

This option will help you run your morning routine on several git repositories quite easily.

Some options are available and described in this page. But there are also [global flags](flags.md) available.

![Simple demo](./assets/heimdall-good-morning-demo.gif)

The repositories are gone through one after the other, and the line being refreshed tells how far along the routine is, `☀️ [3/12] Good Morning my-repo`. Above it, one line is kept per repository already done, `✔` when every command worked and `✖` with the name of the first one which did not. The routine ends with `☕️ All done ! 12 repositories, 2 with a failing command`, before the table of the results.

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

### Override option: `--override-cmds` or `-o`

Set this flag to override the default morning routine command dynamically

```bash
heimdall good-morning -o
```

![Demo -o](./assets/heimdall-good-morning-override.gif)

### Override commands: `--run-cmds` or `r`

*Associated with `-o`*, you can set commands to override default morning routine commands

```bash
heimdall good-morning -o -r "git status, git pull"
```

![Demo -o -r](./assets/heimdall-good-morning-override-cmds.gif)