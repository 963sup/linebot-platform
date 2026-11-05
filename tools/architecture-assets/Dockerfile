ARG BASE_IMAGE=ubuntu:24.04@sha256:fe81560990fae86dc4743202b2cb4d3df98d0bcf949173cecef15ce1127ef852
FROM ${BASE_IMAGE}

ARG INKSCAPE_PACKAGE
ARG NOTO_CJK_PACKAGE
COPY packages.lock /tmp/architecture-assets-packages.lock
RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
       "${INKSCAPE_PACKAGE}" "${NOTO_CJK_PACKAGE}" ca-certificates \
    && test "$(inkscape --version | awk '{print $2}')" = "1.2.2" \
    && dpkg-query -W -f='${binary:Package}=${Version}\n' | LC_ALL=C sort > /tmp/installed-packages.lock \
    && cmp /tmp/architecture-assets-packages.lock /tmp/installed-packages.lock \
    && rm -rf /var/lib/apt/lists/* /tmp/architecture-assets-packages.lock /tmp/installed-packages.lock

COPY render.sh /usr/local/bin/render-architecture-overview
ENTRYPOINT ["/usr/local/bin/render-architecture-overview"]
