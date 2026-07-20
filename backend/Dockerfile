# --- build stage -------------------------------------------------------------
FROM golang:1.22-alpine AS build
WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/notehub-backend .

# --- runtime stage -------------------------------------------------------------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/notehub-backend .

EXPOSE 8080
ENTRYPOINT ["./notehub-backend"]
