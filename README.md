# Heimdall

A CLI to help with your git directories (for now 😉).

Based on the myth of the Nordic God, [Heimdall](https://en.wikipedia.org/wiki/Heimdall), the CLI is here to ease with your multiple Git repositories.

For now, Heimdall has 4 main commands :

- [Git-info](https://yodamad.github.io/heimdall/git-info) to help you manage all your git repositories and now their current branch, if they have some local changes or if they are behind the remote repository
- [Git-clone](https://yodamad.github.io/heimdall/git-clone) to clone a git repository or all repositories of a group (in GitHub or GitLab) and keep the same path
- [Good-morning](https://yodamad.github.io/heimdall/good-morning) to run your morning routine on all your git repositories (like `git pull` or `git status` for example)
- [Env-info](https://yodamad.github.io/heimdall/env-info) to display useful information about your environment (like kubectl contexts, helm repositories, docker contexts...)

The complete documentation can be found on the dedicated [docsite](https://yodamad.github.io/heimdall/) and here is a quick demo of the interactive mode ⤵️

![Simple demo](site/docs/assets/heimdall-git-info-demo.gif)

## How to install

__*On MacOS:*__

Heimdall is available through `brew`

```bash
brew tap yodamad/tools
brew install heimdall
```

__*On Linux:*__ ⚠️ Use it at your own risk *for now* ⚠️

There are available on [Release page](https://github.com/yodamad/heimdall/releases) but not well tested to be honest

__*On Windows:*__ ❌ Not available for now, some compatibilities problems.
