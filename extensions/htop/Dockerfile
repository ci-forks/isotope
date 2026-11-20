# htop, the interactive process viewer, as an Isotope system extension.
# See extensions/README.md for how an extension is built and published.
#
# htop needs ncurses, which the Hadron base ships only as libtinfo. A shared
# ncurses in the extension would put a second copy over files of the running
# system, which the conflict check of `isotope ext` refuses. So ncurses is
# built as a static library and linked into htop, and the extension carries
# only the htop binary, its desktop entry and its icons. At run time htop reads
# the system's terminfo database.

# Set by .github/workflows/extensions.yml from scripts/bump-hadron.sh, so the
# extension builds on the same Hadron release as the image.
ARG HADRON_VERSION
# ncurses from the checksummed source cache Hadron itself builds from, at the
# version the Hadron base ships (NCURSES_VERSION in kairos-io/hadron).
ARG NCURSES_VERSION=6.6

FROM ghcr.io/kairos-io/hadron-sources/ncurses:${NCURSES_VERSION} AS ncurses-download

FROM ghcr.io/kairos-io/hadron-toolchain:${HADRON_VERSION} AS sources
WORKDIR /sources
# renovate: datasource=github-releases depName=htop-dev/htop
ARG HTOP_VERSION=3.5.3
RUN curl -fL --retry 3 https://github.com/htop-dev/htop/releases/download/${HTOP_VERSION}/htop-${HTOP_VERSION}.tar.xz -o htop.tar.xz
COPY --from=ncurses-download /sources/downloads/ncurses.tar.gz /sources/
RUN tar -xf htop.tar.xz && mv htop-* htop
RUN tar -xf ncurses.tar.gz && mv ncurses-* ncurses

# Static, wide-character ncurses. It is installed into /deps, so none of it
# reaches the extension.
FROM sources AS ncurses
WORKDIR /sources/ncurses
ENV COMMON_CONFIGURE_ARGS="${COMMON_CONFIGURE_ARGS//--enable-shared/}"
RUN ./configure ${COMMON_CONFIGURE_ARGS} \
    --without-shared \
    --with-normal \
    --enable-widec \
    --enable-pc-files \
    --with-pkg-config-libdir=/usr/lib/pkgconfig \
    --with-terminfo-dirs=/etc/terminfo:/usr/share/terminfo \
    --with-default-terminfo-dir=/usr/share/terminfo \
    --without-ada \
    --without-cxx-binding \
    --without-manpages \
    --without-progs \
    --without-tests \
    --without-debug
RUN make -j"$(nproc)"
RUN make DESTDIR=/deps install.libs install.includes

FROM sources AS build
COPY --from=ncurses /deps/ /
WORKDIR /sources/htop
RUN ./configure ${COMMON_CONFIGURE_ARGS} \
    --enable-unicode \
    --disable-sensors \
    --disable-capabilities \
    --disable-delayacct \
    --disable-hwloc \
    --disable-unwind
RUN make -j"$(nproc)"
RUN make DESTDIR=/output install
RUN rm -rf /output/usr/share/man /output/usr/share/doc

# What the extension contains: everything under /usr of this stage.
FROM scratch AS default
COPY --from=build /output/ /

# Checks the workflow runs before it publishes anything. Kept offline.
FROM ghcr.io/kairos-io/hadron:${HADRON_VERSION} AS test
COPY --from=default / /extension/
# A system extension is merged over /usr. It must not carry a file the base
# already has, or it would replace the base's copy.
RUN set -eu; \
    clash=$(cd /extension && find . ! -type d | while read -r f; do [ ! -e "/${f#./}" ] || echo "${f#./}"; done); \
    if [ -n "$clash" ]; then echo "the extension overlays base files: $clash" >&2; exit 1; fi
RUN cp -a /extension/. / && rm -rf /extension
RUN htop --version
# ncurses is linked in: only the C library is dynamic.
RUN set -eux; \
    libs=$(ldd /usr/bin/htop | grep -v -e 'ld-musl' -e 'libc\.' -e 'linux-vdso' || true); \
    [ -z "$libs" ]
# Draw one screen on a pseudo terminal, to prove ncurses finds the system's
# terminfo database and htop reads /proc.
RUN set -eux; \
    export TERM=xterm-256color HOME=/tmp; \
    script -q -c 'htop -n 1 -d 1' /dev/null </dev/null >/tmp/htop.out 2>&1 & pid=$!; \
    sleep 3; kill "$pid" 2>/dev/null || true; \
    grep -q 'Load average' /tmp/htop.out; \
    grep -q 'PID USER' /tmp/htop.out
