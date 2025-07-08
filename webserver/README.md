# Go deployment to Azure

Before deployment check os and architecture on dashboard or with CLI on azure website by checking environment variables.

* compile:
`GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -v server.go`

* zip:
`zip -r ../webserver.zip .`

* deploy:
`az functionapp deployment source config-zip -g 1-926bb12b-playground-sandbox -n gofunc --src webserver.zip`



## Configuration

When using HTTP as a trigger option need to be enabled:
```
    "enableForwardingHttpRequest" : true
```

Runs server when executing function app and then from them retrive and pass to app data in form of http requests.
```
    "defaultExecutablePath": "./server.exe",
```
