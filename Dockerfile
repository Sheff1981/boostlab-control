FROM golang:1.27-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/boostlab-control ./cmd/control

FROM alpine:3.22
RUN adduser -D -H -s /sbin/nologin boostlab
COPY --from=build /out/boostlab-control /usr/local/bin/boostlab-control
USER boostlab
EXPOSE 8090/tcp
ENTRYPOINT ["/usr/local/bin/boostlab-control"]
