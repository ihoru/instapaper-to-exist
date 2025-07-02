# Instapaper service integration for Exist.io (Go Implementation)

Track articles read per day with [Instapaper](https://instapaper.com/) as an attribute in [Exist.io](https://exist.io/).
Quantify your everyday reading!

This is a Go implementation of the original Python project.

## How it works

The program submits data to Exist.io and then exits. It needs to be run at least
once per day (a few minutes before midnight in your local time zone). On every
run, it checks for and counts new articles in your Instapaper archive, and
submits that count to Exist.io.

It is recommended that you set up a scheduled task (cron job) to run the program periodically
throughout the day, plus just before midnight to ensure all articles are counted.

## Installation

### Option 1: Clone and Build

```sh
# Clone the repository
git clone https://github.com/ihoru/instapaper-to-exist.git

# Navigate to the project directory
cd instapaper-to-exist

# Build the application
go build -o instapaper-to-exist
```

### Option 2: Using Go Install

```sh
go install github.com/ihoru/instapaper-to-exist@latest
```

## Configuration

Provide configuration options as environmental variables. The following variables are required:

```
EXIST_CLIENT_ID=            # Your Exist.io client ID
EXIST_CLIENT_SECRET=        # Your Exist.io client secret
INSTAPAPER_ARCHIVE_RSS=     # Your Instapaper archive RSS URL
```

The following variables are optional with defaults:

```
EXIST_OAUTH2_RETURN="http://localhost:9009/"  # OAuth2 return URL
EXIST_ATTRIBUTE_NAME="Articles read"          # Name of the attribute in Exist.io
```

You can obtain the client ID and secret by
[registering your client as an Exist app](https://exist.io/account/apps/edit/).

You can find your Instapaper archive RSS link by logging into your Instapaper
account, visiting the [Archive page](https://instapaper.com/archive), and
viewing the page's source code.

You can set these environment variables directly or create a `.env` file in the same directory as the executable.

## Command Line Options

```
Usage of ./instapaper-to-exist:
  -days int
        Number of days to consider for changing stats (default 3)
  -verbose
        Enable verbose logging
  -today int
        Value to set for today's stats [-1 to skip] (default -1)
  -yesterday int
        Value to set for yesterday's stats [-1 to skip] (default -1)
```

## State Management

The application stores state information in the user's home directory under `~/.local/state/instapaper-to-exist/`. 
This includes:

- OAuth2 tokens for Exist.io
- List of processed articles
- Reading statistics by date

If you encounter issues with corrupted state files, the application will automatically remove them and create new ones.

## Troubleshooting

If you encounter an error like `gob: encoded unsigned integer out of range` when running the application, it means there's an issue with the state files. The application should handle this automatically by removing the corrupted files and creating new ones.

If you continue to experience issues, you can manually remove the state files:

```sh
rm -rf ~/.local/state/instapaper-to-exist/
```

## Scheduling with Cron

To run the application periodically, you can set up a cron job. For example, to run it every two hours during the day and just before midnight:

```
# Run every two hours from 8am to 10pm
0 8,10,12,14,16,18,20,22 * * * /path/to/instapaper-to-exist

# Run at 11:55pm to capture end-of-day stats
55 23 * * * /path/to/instapaper-to-exist
```

Make sure to set the environment variables in your crontab or reference a script that sets them.
