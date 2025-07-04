FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

# RUN apt-get update || true \
#      && apt-get install -y --no-install-recommends gnupg ca-certificates \
#      && gpg --keyserver keyserver.ubuntu.com --recv-keys 871920D1991BC93C \
#      && gpg --export 871920D1991BC93C | gpg --dearmour -o /usr/share/keyrings/ubuntu-jammy.gpg

RUN apt-get update && apt-get -y upgrade && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*


# RUN echo "deb [signed-by=/usr/share/keyrings/ubuntu-jammy.gpg] http://archive.ubuntu.com/ubuntu jammy main restricted universe multiverse" > /etc/apt/sources.list \
#      && apt-get update \
#      && apt-get install -y --no-install-recommends ca-certificates && \
#                                rm -rf /var/lib/apt/lists/*
#

WORKDIR /app

COPY cmd/calert.bin .
COPY static/ /app/static/
COPY config.toml .

ARG CALERT_GID="999"
ARG CALERT_UID="999"

RUN chmod +x /app/calert.bin

EXPOSE 6000

ENTRYPOINT [ "./calert.bin" ]
