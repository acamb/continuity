FROM alpine
COPY ./bin/continuity-server-static* /usr/bin/continuity-server
COPY continuity-entrypoint.sh /
RUN chmod +x /continuity-entrypoint.sh
RUN chmod +x /usr/bin/continuity-server

RUN apk update && apk upgrade

RUN addgroup -S continuity-server && \
    adduser -S -D -H -G continuity-server -s /sbin/nologin continuity-server && \
    mkdir -p /opt/continuity && chown continuity-server:continuity-server /opt/continuity
ENV GIN_MODE=release
CMD ["/bin/sh","-c","/continuity-entrypoint.sh"]
