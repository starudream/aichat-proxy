# AIChat Proxy

Status: **WIP**

<p align="center">
<img src="https://img.shields.io/github/actions/workflow/status/starudream/aichat-proxy/docker-camoufox.yml?style=for-the-badge&label=camoufox" alt="golang">
<img src="https://img.shields.io/docker/v/starudream/aichat-proxy-camoufox?style=for-the-badge&label=camoufox" alt="golang">
<img src="https://img.shields.io/docker/image-size/starudream/aichat-proxy-camoufox?style=for-the-badge&label=camoufox" alt="golang">
<br>
<img src="https://img.shields.io/github/actions/workflow/status/starudream/aichat-proxy/docker-aichat-proxy.yml?style=for-the-badge&label=aichat-proxy" alt="golang">
<img src="https://img.shields.io/docker/v/starudream/aichat-proxy?style=for-the-badge&label=aichat-proxy" alt="golang">
<img src="https://img.shields.io/docker/image-size/starudream/aichat-proxy?style=for-the-badge&label=aichat-proxy" alt="golang">
<br>
<img src="https://img.shields.io/github/last-commit/starudream/aichat-proxy?style=for-the-badge" alt="license">
<img src="https://img.shields.io/github/license/starudream/aichat-proxy?style=for-the-badge" alt="license">
<br><br>
<img src="https://socialify.git.ci/starudream/aichat-proxy/image?font=Inter&forks=1&issues=1&language=1&name=1&owner=1&pattern=Circuit%20Board&pulls=1&stargazers=1&theme=Auto" alt="project">
</p>

## Support Chatbot

- [x] [Baidu](https://yiyan.baidu.com)
- [x] [Deepseek](https://chat.deepseek.com)
- [x] [Doubao](https://www.doubao.com/chat)
- [x] [Google](https://aistudio.google.com)
- [x] [Kimi](https://www.kimi.com)
- [x] [Qwen](https://chat.qwen.ai)
- [x] [Yuanbao](https://yuanbao.tencent.com)
- [x] [ZhiPu](https://chat.z.ai)

## Docker Compose

```yaml
services:
  aichat-proxy:
    image: ghcr.io/starudream/aichat-proxy:develop
    container_name: aichat-proxy
    hostname: aichat
    restart: always
    healthcheck:
      test: [ "CMD-SHELL", "test -z \"$(supervisorctl status | awk '{print $2}' | grep -v 'RUNNING')\" || exit 1" ]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 30s
    ports:
      - "9540:9540"
      - "9560:9560"
    volumes:
      - "./aichat-proxy/.env:/app/.env:ro"
      - "./aichat-proxy/certs:/app/certs"
      - "./aichat-proxy/userdata:/app/userdata"
    environment:
      - HTTP_PROXY=http://10.10.10.10:7890
      - HTTPS_PROXY=http://10.10.10.10:7890
```

## [License](./LICENSE)
