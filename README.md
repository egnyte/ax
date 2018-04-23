# Ax

![Logo](https://raw.githubusercontent.com/egnyte/ax/master/ax.png)

[![Travis CI status image](https://travis-ci.org/egnyte/ax.svg?branch=master)](https://travis-ci.org/egnyte/ax)

It's a structured logging world we live in, but do we really have to look at JSON logs? Not with Ax.

Ax features:

* Read logs from various sources, currently:
  * [Kibana](https://www.elastic.co/products/kibana)
  * [AWS Cloudwatch Logs](https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/WhatIsCloudWatchLogs.html)
  * [GCP Stackdriver Logs](https://cloud.google.com/logging/)
  * Piped input
  * Docker containers
* Filter logs based on attribute (field) values as well as text phrase search
* Select only the attributes you are interested in
* The ability to "follow" logs (Ax keeps running and shows new results as they come in)
* Various output format (pretty text, JSON, pretty JSON, YAML) that can be used for further processing
* Command completion for all commands and flags (e.g. completing attribute names)

## Installation
Ax can be installed in two ways:

1. through downloading pre-compiled binaries (for official releases)
2. through fetching the latest version from Github and compiling using the Go tools

### Pre-compiled binaries
On Linux or Mac (this will attempt to install the binary into `/usr/local/bin` by default):

    curl -sfL https://raw.githubusercontent.com/egnyte/ax/master/install.sh | sh

If you want to install the `ax` binary into another location, simply set the `BINDIR` environment variable, e.g.:

    curl -sfL https://raw.githubusercontent.com/egnyte/ax/master/install.sh | BINDIR=. sh

to install in the current directory.

If you don't trust piping random shell scripts from the internet into a shell, feel free to download the `install.sh` script first, inspect it, then run it through bash manually or, simply go through the [Ax releases](https://github.com/egnyte/ax/releases) page and download the tarball of your choice.

Upgrades can be installed by simply re-running the above command.

### Bleeding edge with Go-tools
For now there's no pre-built binaries, so to run this you need a reasonably recent version of Go, then download it into your GOPATH:

    go get -u github.com/egnyte/ax/...

This will also put the `ax` binary into your `$GOPATH/bin` so make sure that's in your `$PATH`.

To update Ax to the latest and greatest, just rerun the command above.

## Development
After the above `go get` call, you will have a git checkout of the repo under `$GOPATH/src/github.com/egnyte/ax`. If you want to work on Ax, just fork the repo and update `.git/config` appropriately.

To make sure you're building Ax with the approriate versions of its dependencies run:

    dep ensure

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

## Setup with Kibana, Cloudwatch or Stackdriver
To setup Ax for use with Kibana, Cloudwatch or Stackdriver, run:

    ax env add

This will prompt you for a name, backend-type and various other things depending on your backend of choice. After a successful setup, you should be ready to go.

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

# Advanced filtering

Ax also allows you to filter by the existence of a field in a message, or to test field values for membership in a set of values.

To search for all logs with a `domain` field:

    ax --where-exists domain

Or for all logs without a traceback field:

    ax --where-not-exists traceback

To search for messages from a subset of domains:

    ax --where-on-of domain:zef --where-one-of domain:fredek

To search for all messages *except* ones from specific domains:
    ax --where-not-on-of domain:boring --where-not-one-of domain:dull

**NOTE** Advanced filtering is currently only implemented for the stream and Docker backends. Attempting to use them with other backend will raise an error.
# "Tailing" logs

Use the `-f` flag:

    ax -f --where domain=zef

# Different output formats

Don't like the default textual output, perhaps you prefer YAML:

    ax --output yaml

or pretty JSON:

    ax --output pretty-json

# Customizing colors for "text" output

In your `~/.config/ax/ax.yaml` file (`ax env edit`) you can override the default colors as follows:

    colors:
        timestamp:
            fg: magenta
        message:
            bold: true
        attributekey:
            faint: true
            fg: green
        attributevalue:
            faint: true
            fg: blue

For each "color" you can set:

* `fg` — foreground color (`red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`)
* `bg` — background color (same options)
* `bold` — bold font (`true` or `false`)
* `italic` — italic font (`true` or `false`)
* `underline` — underline font (`true` or `false`)
* `faint` — faint (color) font (`true` or `false`)

# Getting help

    ax --help
    ax query --help

# Found anything broken?

Report it as a Github issue!
