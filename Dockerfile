FROM ubuntu:latest

ENV XCADDY_VERSION=0.3.0 \
    GOLANG_VERSION=18.1 \
    APPPORT=:2011

COPY docker-files /

RUN set -x \
  && apt-get -y update \
  && apt-get install -y curl \
  && mkdir build \
  && cd build \
  && curl -sSLO https://github.com/caddyserver/xcaddy/releases/download/v${XCADDY_VERSION}/xcaddy_${XCADDY_VERSION}_linux_amd64.tar.gz \
  && curl -sSLO https://go.dev/dl/go1.${GOLANG_VERSION}.linux-amd64.tar.gz \
  && rm -rf /usr/local/go \
  && tar -C /usr/local -xzf go1.${GOLANG_VERSION}.linux-amd64.tar.gz \
  && export PATH=$PATH:/usr/local/go/bin \
  && tar xfvz xcaddy_${XCADDY_VERSION}_linux_amd64.tar.gz \
  && ./xcaddy build --with github.com/kirsch33/realip \
    --with github.com/caddyserver/jsonc-adapter \
    --with github.com/git001/caddyv2-upload \
  && pwd \
  && mv caddy /usr/local/bin/ \
  && cd .. \
  && apt-get -y autoremove \
  && apt-get -y autoclean \
  && rm -rf build /usr/local/go /var/cache/apk/* \
  && /usr/local/bin/caddy list-modules

WORKDIR /opt/webroot/

# CMD ["executable","param1","param2"] (exec form, this is the preferred form)
CMD ["/usr/local/bin/caddy","run","-adapter","jsonc","-config","config/Caddyfile-upload.json"]