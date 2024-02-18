FROM clickhouse/clickhouse-server:23.12.4.15-alpine

RUN set -ex  \
    && apk update \
    && apk add \
        openssh-client \
        openssh-keygen \
        rsync

RUN ssh-keygen -b 2048 -t rsa -f /root/.ssh/id_rsa -q -N "" \
    && cp /root/.ssh/id_rsa.pub /root/.ssh/authorized_keys \
    && ln -s /usr/local/bin/clickhouse-tools/clickhouse-tools /usr/bin/clickhouse-tools \
    && mkdir -p /var/lib/clickhouse/backup/

ENTRYPOINT ["/entrypoint.sh"]
