# go-live-reload

go tool to build and run continuously

This tool will read a configuration which contains a set of build instructions. These instructions will compile and run until a kill signal is sent `(ctrl+c) or (cmd+c)` to the tool where it will in turn send kill signals to the runners. The configurations also include a set of glob patterns to watch for file modifications. These will be scanned based on the `heartbeat` definition and if a mismatch in the count of files or any of those files having a differing modification timestamp, send the kill signal to that specific runner in the set, rebuild and run again. If a build fails, the runner will halt until a `hearbeat` detects a change. See the example config below to get an idea.

>[!TIP]
>Test it out on [mywebserver](https://github.com/dearing/mywebserver?tab=readme-ov-file#try-out).

---
## global install

```
go install github.com/dearing/go-live-reload@latest
```
>[!NOTE]
>if your project is Go 1.24+, you can pin the tool to your module workspace instead of installing it in the host's path
## go tool usage
```
go get -tool github.com/dearing/go-live-reload@latest
go tool go-live-reload --version
go tool go-live-reload --init-config
go tool go-live-reload
```
## tool maintenance tips
```
      pin => go get -tool github.com/dearing/go-live-reload@v0.0.2
   update => go get -tool github.com/dearing/go-live-reload@latest
downgrade => go get -tool github.com/dearing/go-live-reload@v0.0.1
uninstall => go get -tool github.com/dearing/go-live-reload@none
```
---

## usage

```
Usage: go-live-reload [options]

This tool takes a set of build groups and runs them in parallel. Each build group
is defined in the configuration file and contains a set of build and run commands
along with arguments and environment variables. The build group will then watch for
changes based on the "match" values and restart just itself when a modification
is detected or if a new file is added or removed. This is based comparing the
current matches to the previous matches every heartbeat duration. If you find the
tool is restarting too frequently or there is too much IO pressure, you can increase
the heartbeat duration to reduce the frequency of checks.

Tips:

1) The --overwrite-heartbeat option is used to temporarily overwrite all build group
heartbeats with the specified duration. This is useful for tweaking the heartbeat
based on the host system's performance. Valid options are those that can be parsed
by Go's time.ParseDuration function. You can observe matches and duration with the
--log-level=debug option.

ex: go-live-reload --overwrite-heartbeat=500ms --log-level=debug

2) The --build-groups option is used to specify a comma separated list of build groups
to run. If no build groups are specified, all build groups defined in the config
will be ran. If no matches are found, the tool will exit with an error.

ex: go-live-reload --build-groups=frontend,backend

3) The ENV lists are appended to the current environment variables. If you need to
overwrite an environment variable, you can do so by specifying the same key in
the ENV list. If you need to clear the environment, set the value to an empty list.
Clearing and then appending is not supported by this tool.

Options:

  -build-groups string
        comma separated list of build groups to run
  -config-file string
        load a config file (default "go-live-reload.json")
  -init-config
        initialize and save a new config file
  -log-level string
        log level (debug, info, warn, error) (default "info")
  -overwrite-heartbeat duration
        temporarily overwrite all build group heartbeats
  -version
        print debug info and exit
```

## example config

```json
{
  "name": "mywebserver",
  "description": "A simple live reload server",
  "builds": [
    {
      "Name": "mywebserver",
      "SrcDir": ".",
      "OutDir": "build",
      "BuildArgs": [
        "build",
        "-o",
        "build/mywebserver"
      ],
      "BuildEnvirons": null,
      "RunCommand": "./mywebserver",
      "RunArgs": [
        "--bind",
        ":8081"
      ],
      "RunEnvirons": null,
      "RunWorkDir": "build",
      "Globs": [
        "*.go","embeded/template/*", "embeded/wwwroot/*/*", "embeded/wwwroot/*"
      ],
      "HeartBeat": 1000000000
    }
  ]
}
```
