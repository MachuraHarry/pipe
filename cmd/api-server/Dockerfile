FROM golang:1.25-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o /pipe-api ./cmd/api-server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=build /pipe-api /usr/local/bin/pipe-api
EXPOSE 3001
CMD ["pipe-api", "-addr", ":3001", "-timeout", "30s"]
