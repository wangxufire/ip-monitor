# ip-monitor

### Install
```
go get github.com/wangxufire/ip-monitor@latest
```
or 
```
go install github.com/wangxufire/ip-monitor@latest
```
or 
```
GO111MODULE=off go get github.com/wangxufire/ip-monitor
GO111MODULE=off go install github.com/wangxufire/ip-monitor
```

### Run
```shell
ip-monitor -bark ${bark_device_code} -period 600
```

### modify tencent cloud DNSPod record
```shell
ip-monitor -bark ${bark_device_code} -period 600 -domain xxx.com -secretId xxx -secretKey xxx
```
