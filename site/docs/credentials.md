# Credentials : `credentials`, `creds` or `cred`

`credentials` command checks the tokens of the platforms of your [configuration file](config.md) against their API, and updates the ones which are missing, expired or about to expire.

A new credential is stored in the keyring of your operating system, and the configuration file only keeps a reference to it (`keyring.<entry>`). Several platforms can share one entry, so that a token used by several hosts is rotated in one place.

[Global flags](flags.md) are available too.

![Demo](./assets/heimdall-credentials-demo.gif)

!!!warning "A keyring is needed"
    The keyring needs a running Secret Service (GNOME Keyring, KWallet...) on Linux, and Windows is not supported. Without a keyring, nothing can be stored : use `--reference env.MY_VAR` to point a platform at an environment variable instead. The platforms configured with `env.` or a literal value keep working.

Run without any subcommand, `heimdall credentials` checks every platform, tells what is wrong and offers to fix it.

```bash
heimdall credentials
```

## List: `list` or `ls`

Lists the platforms of the configuration file, with their type, where their token is read from (`keyring: name`, `env: NAME`, `config file` or `not set`), their SSH key and where its passphrase is read from. Nothing is sent over the network and no secret is ever displayed.

```bash
heimdall credentials list
```

## Check: `check [host...]`

Checks the token of every platform, or of the given ones, against its API. Each platform gets a verdict : valid, expiring, rejected, no token, unresolved (the reference points at nothing), unsupported or unreachable.

```bash
heimdall credentials check
heimdall credentials check github.com
```

| Flag | Description |
|------|-------------|
| `--expiring-within` | Report a token expiring within this duration, 14 days (`336h`) by default |
| `--quiet` or `-q` | Print nothing, only set the exit code |
| `--strict` | Fail when a token is about to expire too |

The exit code tells the outcome, so that the command can be used in a script or a cron job :

| Code | Meaning |
|------|---------|
| `0` | Everything is fine |
| `1` | Something needs fixing : a token is rejected, missing or cannot be resolved. With `--strict`, a token about to expire too |
| `2` | A platform could not be reached, so its token could not be checked |

```bash
heimdall credentials check --quiet --strict || echo "Time to rotate a token"
```

## Update: `update [host...]`

Stores a new credential for one or several platforms. Without a host, a list of the platforms is displayed after a check : the ones which need attention are already selected, pick the ones to update with ++space++.

```bash
heimdall credentials update github.com
heimdall credentials update
```

Heimdall asks for the secret twice, without displaying it, stores it in the keyring and writes the reference in the configuration file. Then it checks the updated platforms again. A host which is not configured yet is accepted, this is how a platform is added : Heimdall asks for its type, or takes it from `--type`.

| Flag | Description |
|------|-------------|
| `--all` or `-a` | Update every configured platform |
| `--entry` or `-e` | Name of the keyring entry to store the secret under |
| `--from-stdin` | Read the secret from the standard input instead of asking for it |
| `--reference` | Point the platforms at an existing secret (`env.MY_VAR` or `keyring.my-entry`) instead of storing one |
| `--ssh-passphrase` | Update the passphrase of the SSH key instead of the token |
| `--type` | Type of a platform which is not configured yet (`github` or `gitlab`) |

### Several platforms at once

When several platforms are chosen, whether picked in the list, given on the command line or with `--all`, Heimdall asks for one secret, stores it in one keyring entry and points every chosen host at it. The entry is named after the host when there is only one, `shared` otherwise, or whatever you give with `--entry`.

```bash
heimdall credentials update github.com github.mycompany.com --entry github-token
```

Both hosts now read `keyring.github-token` : rotating this token later is one command.

### Scripting

To store a secret without any prompt, give it on the standard input :

```bash
echo "$NEW_TOKEN" | heimdall credentials update github.com --from-stdin
```

To store nothing and point a platform at an environment variable (the only option without a keyring) :

```bash
heimdall credentials update gitlab.com --reference env.GITLAB_TOKEN
```

### SSH key passphrase

With `--ssh-passphrase`, the passphrase of the SSH key is updated instead of the token. There is nothing to check a passphrase against, so no check follows.

```bash
heimdall credentials update github.com --ssh-passphrase
```

## Migrate: `migrate`

Moves the tokens written in the configuration file into the keyring : each token is stored in an entry named after its host, and the file keeps a reference to it. Heimdall lists the platforms concerned and asks for a confirmation first.

```bash
heimdall credentials migrate
```

!!!warning "Migrating does not make an exposed token safe"
    `migrate` removes a token from the configuration file, but **not** from your backups, nor from the history of the repository if the file is versioned. A token which was exposed has to be revoked and reissued : use `heimdall credentials update` for the new one.
