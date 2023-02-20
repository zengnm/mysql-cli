# mysql-cli

打包环境依赖： golang + goreleaser + upx, 执行以下命令:

```shell
goreleaser --clean --snapshot --skip-publish
```

默认为linux + amd64环境打包，有其他需求请在.goreleaser.yaml文件取消注释或者修改。