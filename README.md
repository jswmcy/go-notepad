# go-notepad

## 项目简介
轻量离线网页记事本，无广告、无登录、不上传数据，支持Markdown快捷排版，适配全平台硬件。

## 部署方式

### Docker部署
```bash
docker run -d \
  -p 3000:3000 \
  -v $(pwd)/data:/app/data \
  --restart always \
  zenbox01/go-notepad
```

### 二进制部署
在Release下载对应架构压缩包，解压直接运行，访问：http://ip:3000

## 使用说明
- 打开网页自动加载笔记，停止输入500ms自动保存
- 顶部按钮一键生成Markdown格式，无需手动编写语法
- 笔记存放目录：data/note.txt，可手动备份

## 注意事项
- 单人使用，不支持多用户
- 明文存储，禁止存放极高敏感密钥
- 局域网访问地址：http://设备IP:3000
