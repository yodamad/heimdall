# Configuration

This file has to be a `yaml` formatted file.

In this config file, you can configure these elements in three ways, from the safest to the least safe :

- `keyring.<entry>` : the value is read from an entry of the keyring of your operating system
- `env.<NAME>` : the value is read from an environment variable
- a hardcoded value, written as is in the file (bad for security)

The prefix decides : a value starting with `keyring.` is read from the keyring, a value starting with `env.` from the environment, and any other value is used as it is. A value which merely contains `env.` or `keyring.` elsewhere than at the start (a token like `glpat-env.something`) is a literal.

!!!example "A sample file"
    ```yaml
    work_dir: /home/johndoe/work/
    platforms:
      gitlab.com:
        token: MY_TOKEN # Bad !!
        type: gitlab
        public_key: /path/to/public_key
        public_key_passphrase: env.PUBLIC_KEY_PASSPHRASE
      github.com:
        token: env.ENV_VAR_TOKEN
        type: github
    ```

For each platform, you can define the following elements (some can reference an environment variable or the keyring):

| Element | Description | Support `env.` and `keyring.` |
|---------|-------------|--|
| `token` | The authentication token to access the platform | ✅ |
| `type` | The type of platform (github, gitlab) | ❌ |
| `public_key` | The path to the public key to use for SSH connection | ❌ |
| `public_key_passphrase` | The passphrase to use for the public key | ✅ |

!!!note "`public_key_passphrase` or `public_key_password`"
    `public_key_passphrase` is the canonical name and the one to use. A first version of Heimdall only read `public_key_password`, so this one is still accepted. When both are set, `public_key_passphrase` wins.

## Keyring

A `keyring.` reference names an entry of the keyring of your operating system (Keychain on macOS, Secret Service on Linux). The configuration file only holds the name of the entry, never the secret.

Several platforms can point at the same entry, so that a token used by several hosts is rotated in one place :

```yaml
platforms:
  github.com:
    type: github
    token: keyring.github-token
  github.mycompany.com:
    type: github
    token: keyring.github-token # same entry : one token to rotate
  gitlab.com:
    type: gitlab
    token: keyring.gitlab-token
    public_key_passphrase: keyring.ssh-passphrase
```

The entries are managed with [`heimdall credentials`](credentials.md), which stores a new secret in the keyring and writes the reference in this file for you.

!!!warning "A token written in the file is plain text on disk"
    A literal token can be read by anyone, or anything, which can read the file, and ends up in your backups and, if the file is versioned, in its history. `heimdall credentials migrate` moves the tokens written in the file into the keyring, see [credentials](credentials.md#migrate-migrate).

!!!info "No keyring ?"
    The keyring needs a running Secret Service on Linux and is not available on Windows. Without it, the `env.` references and the literal values keep working, and a `keyring.` reference resolves to nothing.

## Morning routine

The commands of your morning routine, run by [good-morning](good-morning.md) and by the ++m++ key of the [TUI](tui.md), are defined in the `morning_routine` element.

```yaml
morning_routine:
  shell: zsh
  commands: git fetch, git pull
```

| Element | Description |
|---------|-------------|
| `commands` | The commands to run in each repository, separated by a comma |
| `shell` | The shell used to run them, `bash` by default |
| `ohmyzsh` | Set it to `true` to run them in a login interactive shell (`-lic`), so that the aliases of your shell are available |
