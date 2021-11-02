FROM golang:1.17-bullseye AS builder

# To improve performance when iterating locally with docker builds,
# create a cached layer that doesn't depend on the source code.

RUN mkdir /build && \
    cd /build && \
    apt-get update && \
    apt-get install -y xz-utils && \
    curl -sLO https://github.com/upx/upx/releases/download/v3.96/upx-3.96-amd64_linux.tar.xz && \
    tar --strip-components=1 -xf upx-3.96-amd64_linux.tar.xz


# Create a cached layer that depends on go.mod and go.sum. This will speed up
# future docker builds because we'll be able to skip module downloads almost always
# (since go.mod and go.sum don't change often).

COPY go.mod go.sum /build/
RUN cd /build && GOOS=linux go mod download -x && GOOS=darwin go mod download -x


# This layer will rebuild on any change.

COPY . /build
RUN cd /build && \
    GOOS=linux go build -ldflags="-s -w" -o eyecue-codemap-linux . && \
    GOOS=darwin go build -ldflags="-s -w" -o eyecue-codemap-darwin . && \
    ./upx --brute eyecue-codemap-linux eyecue-codemap-darwin


# The final container image only needs the executables.

FROM scratch
COPY --from=builder /build/eyecue-codemap-linux /build/eyecue-codemap-darwin /bin/
