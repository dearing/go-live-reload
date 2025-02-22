# go-live-reload

(alpha) go tool to build and run continuously

>[!CAUTION]
>this software is fresh out of oven, so buyer beware! (but I'm already using it myself)

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
  "name": "myserver",
  "description": "my simple http server",
  "builds": [
    {
      "Name": "myserver",
      "SrcDir": ".",
      "OutDir": "build",
      "BuildArgs": [
        "build",
        "-o",
        "build/myserver"
      ],
      "BuildEnvirons": null,
      "RunCommand": "./build/myserver",
      "RunArgs": [
        "--bind",
        ":8081"
      ],
      "RunEnvirons": null,
      "RunWorkDir": "build",
      "Globs": [
        "test/*.go",
        "test/wwwroot/*"
      ],
      "HeartBeat": 1000000000
    }
  ]
}
```