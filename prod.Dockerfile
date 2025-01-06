FROM ubuntu:22.04
WORKDIR /workfiles

COPY bin/cacheserver_linux_amd64 ./cache_service

ENV SHELL=/bin/bash
EXPOSE 8084
RUN chmod +x "/workfiles/cache_service"
ENTRYPOINT [ "/workfiles/cache_service" ]


