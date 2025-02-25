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
Usage of [go tool] go-live-reload:
  -heartbeat duration
        duration between checks (default 1s)
  -init-config
        initialize and save a new config file
  -load-config string
        load a config file (default "go-live-reload.json")
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
