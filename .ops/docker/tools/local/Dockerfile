FROM golang:1.22-alpine
ENV CGO_ENABLED 0

RUN set -ex \
    && apk update \
    && apk  \
      add \
      bash \
      inotify-tools \
      git \
    && git clone https://github.com/go-delve/delve \
    && cd delve \
    && go install github.com/go-delve/delve/cmd/dlv

WORKDIR /go/src/app
ADD ./goapp .

COPY .ops/docker/tools/local/entrypoint.sh /
RUN chmod 777 /entrypoint.sh

ENTRYPOINT ["sh", "/entrypoint.sh"]
