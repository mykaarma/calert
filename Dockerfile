FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get -y upgrade && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY cmd/calert.bin .
COPY static/ /app/static/
COPY config.sample.toml config.toml

ARG CALERT_GID="999"
ARG CALERT_UID="999"

RUN addgroup --system --gid $CALERT_GID calert && \
    adduser --uid $CALERT_UID --system --ingroup calert calert && \
    chown -R calert:calert /app

USER calert
EXPOSE 6000

ENTRYPOINT [ "./calert.bin" ]
