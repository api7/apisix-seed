FROM alpine:latest as pre-build

ARG APISIX_SEED_VERSION=main

RUN set -x \
    && apk add --no-cache --virtual .builddeps git \
    && git clone https://github.com/api7/apisix-seed.git -b ${APISIX_SEED_VERSION} /usr/local/apisix-seed \
    && cd /usr/local/apisix-seed && git clean -Xdf \
    && rm -f ./.githash && git log --pretty=format:"%h" -1 > ./.githash

FROM golang:1.17

COPY --from=pre-build /usr/local/apisix-seed /tmp/apisix-seed/

RUN if [ "$ENABLE_PROXY" = "true" ] ; then go env -w GOPROXY=https://goproxy.io,direct ; fi \
    && go env -w GO111MODULE=on \
    && cd /tmp/apisix-seed/ \
    && make build \
    &&  make install

ENV PATH=$PATH:/usr/local/apisix-seed

ENV APISIX_SEED_WORKDIR /usr/local/apisix-seed

CMD [ "/usr/local/apisix-seed/apisix-seed" ]

