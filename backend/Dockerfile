
FROM golang:1.23-alpine AS build
WORKDIR /src

RUN apk add --no-cache git

COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/notehub-backend .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata postgresql-client
WORKDIR /app
COPY --from=build /out/notehub-backend .

EXPOSE 8080
ENTRYPOINT ["./notehub-backend"]