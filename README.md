[![Travis CI status image](https://travis-ci.org/egnyte/ax.svg?branch=master)](https://travis-ci.org/egnyte/ax)
# Ax
It's a structured logging world we live in, but do we really have to look at JSON? Not with Ax.

## Installation/Upgrades
For now there's no pre-built binaries, so to run this you need a reasonably recent version of Go, then download it into your GOPATH:

    go get -u github.com/egnyte/ax/...

This will also put the `ax` binary into your `$GOPATH/bin` so make sure that's in your `$PATH`.

To update Ax to the latest and greatest, just rerun the command above.

## Development

After the above `go get` call, you will have a git checkout of the repo under `$GOPATH/src/github.com/egnyte/ax`. If you want to work on Ax, just for the repo and update `.git/config` appropriately.

To run tests:

    make test

To "go install" ax (this will put the resulting binary in `$GOPATH/bin` so put that in your `$PATH`)

    make

## Setup
Once you have `ax` installed, the first thing you'll want to do is setup bash or zsh command completion (I'm not kidding).

For bash, add to `~/.bash_profile`:

    eval "$(ax --completion-script-bash)"

For zsh, add to `~/.zshrc`:

    eval "$(ax --completion-script-zsh)"

After this, you can auto complete commands, flags, environments, docker container names and even attribute names by hittig TAB. Use it, love it, never go back.

## Setup with Kibana
To setup Ax for use with Kibana, run:

    ax env add

This will prompt you for a name, backend-type (kibana in this case), URL and if this URL is basic auth protected a username and password, and then an index.

To see if it works, just run:

    ax --env yourenvname

Or, most likely your new env is the default (check with `ax env`) and you can just run:

    ax

This should show you the (200) most recent logs.

If you're comfortable with YAML, you can run `ax env edit` which will open an editor with the `~/.config/ax/ax.yaml` file (either the editor set in your `EDITOR` env variable, with a fallback to `nano`). In there you can easily create more environments quickly.

## Use with Docker
To use Ax with docker, simply use the `--docker` flag and a container name pattern. I usually use auto complete here (which works for docker containers too):

    ax --docker turbo_

To query logs for all containers with "turbo\_" in the name. This assumes you have the `docker` binary in your path and setup properly.

## Use with log files or processes
You can also pipe logs directly into Ax:

    tail -f /var/log/something.log | ax

# Filtering and selecting attributes
Looking at all logs is nice, but it only gets really interesting if you can start to filter stuff and by selecting only certain attributes.

To search for all logs containing the phrase "Traceback":

    ax "Traceback"

To search for all logs with the phrase "Traceback" and where the attribute "domain" is set to "zef":

    ax --where domain=zef "Traceback"

Again, after running Ax once on an environment it will cache attribute names, so you get completion for those too, usually.

Ax also supports the `!=` operator:

    ax --where domain!=zef

If you have a lot of extra attributes in your log messages, you can select just a few of them:

    ax --where domain=zef --select message --select tag

# "Tailing" logs
Use the `-f` flag:

    ax -f --where domain=zef

# Different output formats
Don't like the default textual output, perhaps you prefer YAML:

    ax --output yaml

or pretty JSON:

    ax --output pretty-json

# Getting help

    ax --help
    ax query --help
