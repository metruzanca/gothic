# The service root on Railway is this directory, so the local replace
# directives in go.mod (../.. and ../../ui) point outside the build context.
# Drop them and build against the published modules instead.
FROM golang:1.26-alpine AS build

WORKDIR /src

COPY . .

RUN go mod edit \
      -dropreplace=github.com/metruzanca/gothic \
      -dropreplace=github.com/metruzanca/gothic/ui \
    && CGO_ENABLED=0 go build -ldflags="-w -s" -o /out/gothicbook .

FROM alpine:3.22

WORKDIR /app

COPY --from=build /out/gothicbook ./gothicbook

ENV PORT=8080
EXPOSE 8080

CMD ["./gothicbook"]
