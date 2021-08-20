FROM golang:1.16-buster AS build-production

RUN mkdir /app
WORKDIR /app

ARG COMMAND

COPY ../ .

# Keep Go's build cache between builds.
# https://github.com/golang/go/issues/27719#issuecomment-514747274
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o ./app -ldflags "-extldflags -static" ./cmd/${COMMAND}

RUN chmod a+x /app/app

## Add the statically linked binary to a distroless image
FROM gcr.io/distroless/base-debian10 as production

COPY --from=build-production /app/app /bin/app

ENTRYPOINT ["/bin/app"]