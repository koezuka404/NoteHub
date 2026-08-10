# ローカル docker-compose 用。依存関係は起動時に entrypoint でインストールする。
FROM node:22-alpine
WORKDIR /app

COPY package*.json ./
COPY docker-entrypoint.sh ./
RUN chmod +x docker-entrypoint.sh

EXPOSE 5173
ENTRYPOINT ["sh", "./docker-entrypoint.sh"]
