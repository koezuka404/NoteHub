# ローカル docker-compose 用。本番ホスティング方式は未確定のため開発サーバーを起動する。
FROM node:22-alpine
WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .

RUN chmod +x docker-entrypoint.sh

EXPOSE 5173
ENTRYPOINT ["./docker-entrypoint.sh"]
