# pair

Configures your git author info with one or more people.

## Install

Clone this repository and put it in your `$PATH`, or just download the `pair`
pre-built binary for OS X and put that somewhere in your `$PATH`.

## Usage

```
# Print the current git author.
$ pair
Michael Bluth <mb@squareup.com>

# Set the current git author from users in the pairs file.
$ pair mb lb
Lindsay Bluth and Michael Bluth <git+lb+mb@squareup.com>

# Set the current git author according to your user, perhaps useful in .bashrc.
$ pair $USER
```

## Configuration

pair uses environment variables to configure its behavior.

### `PAIR_FILE`

Set `PAIR_FILE` to a YAML file containing a map of usernames to full names, e.g.

```
---
lb: Lindsay Bluth
mb: Michael Bluth
```

The default location for this file is `~/.pairs`.

### `PAIR_GIT_CONFIG`

Set `PAIR_GIT_CONFIG` to the path to the git configuration file to use for
setting and getting author info. The default location for this file is
`~/.gitconfig_local`.

### `PAIR_EMAIL`

Set `PAIR_EMAIL` to an email address to use as the base for all derived emails. For example,

```
$ export PAIR_EMAIL=git@example.com
$ pair mb
Michael Bluth <mb@example.com>
$ pair lb mb
Lindsay Bluth and Michael Bluth <git+lb+mb@example.com>
```

The default value for this template is `git@squareup.com`.
