# Instapaper service integration for Exist.io (Go Implementation)

Track articles read per day with [Instapaper](https://instapaper.com/) as an attribute in [Exist.io](https://exist.io/).
Quantify your everyday reading!

This is a Go implementation of the original Python project.

## Setup

Install a copy of the repository:

```sh
git clone https://github.com/ihoru/existio_instapaper.git
```

## Build

Navigate to the golang directory and build the application:

```sh
cd existio_instapaper
go build -o existio_instapaper
```

## How it works

The program submits data to Exist and then exits. It needs to be run at least
once per day (a few minutes before midnight in your local time zone). On every
run, it checks for and counts new articles in your Instapaper archive, and
submits that count to Exist.

It is recommended that you use the provided systemd unit files to manage the
program's lifecycle. It will run the program every two hours during the day,
plus just before midnight.

## Configuration

Provide configuration options as environmental variables. All configuration
options are required.

```
EXIST_CLIENT_ID=
EXIST_CLIENT_SECRET=
EXIST_OAUTH2_RETURN="http://localhost:9009/"
INSTAPAPER_ARCHIVE_RSS=  
# Example: https://instapaper.com/archive/rss/123/XXX
```

You can obtain the two first values and supply the third by
[registering your client as an Exist app](https://exist.io/account/apps/edit/).

You can find your Instapaper archive RSS link by logging into your Instapaper
account, visiting the [Archive page](https://instapaper.com/archive), and
viewing the page's source code.

You can set these environment variables directly or create a `.env` file in the same directory as the executable.

## Command Line Options

```
Usage of ./exist-instapaper:
  -days int
        Number of days to consider for reading stats (default 3)
  -today int
        Value to set for today's stats (default 0)
  -verbose
        Enable verbose logging
  -clean
        Clean all state files and exit
```

## Troubleshooting

If you encounter an error like `gob: encoded unsigned integer out of range` when running the application, it means there's an issue with the state files. This can happen if you've updated the application or if the files have become corrupted.

You can resolve this by running the application with the `-clean` flag to remove all state files:

```sh
./exist-instapaper -clean
```

This will remove all state files and exit. The next time you run the application normally, it will create fresh state files.

## systemd

The systemd directory contains example systemd service unit files for running the
program periodically as a systemd service. You may need to adjust the paths to the
`ExecStart` program and the `EnvironmentFile` file where you store your config.
