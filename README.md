# go-notepad

轻量离线网页记事本，无广告、无登录、不上传数据，支持Markdown快捷排版，适配全平台硬件。

## 特性

- 多记事本支持，通过后缀区分
- 首页展示所有记事本列表
- Markdown 格式按钮，所见即所得
- 500ms 自动保存
- 单文件存储，纯文本

## 部署

### Docker
```bash
docker run -d \
  -p 3000:3000 \
  -v $(pwd)/data:/app/data \
  --restart always \
  zenbox01/go-notepad
```

### 二进制
在 Release 下载对应架构压缩包，解压直接运行，访问：http://ip:3000

## 注意事项

- 单人使用，不支持多用户
- 明文存储，禁止存放极高敏感密钥
- 局域网访问地址：http://设备IP:3000
