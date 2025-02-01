# Configuration

This file has to be a `yaml` formatted file.

In this config file, you can configure these elements either in hardcoded value (bad for security) or reference an environment variable.
The value pointing to an environment variable has to be prefixed by `env.`.

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

For each platform, you can define the following elements (some can reference an environment variable):

| Element | Description | Support env. variable |
|---------|-------------|--|
| `token` | The authentication token to access the platform | ✅ |
| `type` | The type of platform (github, gitlab) | ❌ |
| `public_key` | The path to the public key to use for SSH connection | ❌ |
| `public_key_passphrase` | The passphrase to use for the public key | ✅ |