FROM golang:1.22-alpine AS build
RUN apk add --no-cache git ca-certificates
RUN git clone --depth 1 https://github.com/MachuraHarry/pipe.git /src
WORKDIR /src
RUN go build -o /pipe ./cmd/pipe

FROM alpine:3.19
RUN apk add --no-cache ca-certificates libc6-compat
COPY --from=build /pipe /usr/local/bin/pipe
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
