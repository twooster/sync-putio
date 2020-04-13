FROM golang:1.14-alpine AS build

RUN apk add --no-cache ca-certificates git

ENV GO111MODULE=on

WORKDIR /workdir

COPY . ./

RUN go build -o ./sync-putio

FROM scratch

COPY --from=build /workdir/sync-putio /sync-putio

ENTRYPOINT /sync-putio
