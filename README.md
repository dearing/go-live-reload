# go-live-reload

(alpha) go tool to build and run continuously

>[!CAUTION]
>this software is fresh out of over; caveat emptor

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
      pin => go get -tool github.com/dearing/go-live-reload@v1.0.1
   update => go get -tool github.com/dearing/go-live-reload@latest
downgrade => go get -tool github.com/dearing/go-live-reload@v1.0.0
uninstall => go get -tool github.com/dearing/go-live-reload@none
```
---