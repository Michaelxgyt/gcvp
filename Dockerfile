# Используем официальный образ Xray-core
FROM teddysun/xray:latest
# Устанавливаем только самые необходимые утилиты
RUN apk update && apk add --no-cache bash jq curl
WORKDIR /app; COPY config_template.json .; COPY entrypoint.sh .; RUN chmod +x entrypoint.sh; RUN mkdir -p /etc/xray/; CMD ["/app/entrypoint.sh"]
