# mysql-cli

在macOs上打包命令
```shell
# linux/amd64
env GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" . && upx -9 mysql-cli -o mycli-linux-amd64
# macho/amd64
go build -ldflags="-s -w" . && upx -9 mysql-cli -o mycli-macho-amd64
```