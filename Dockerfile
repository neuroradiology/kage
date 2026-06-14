# Consumed by GoReleaser: it copies the already cross-compiled binary out of the
# build context rather than compiling, so the image build is fast and uses the
# same static binary every other artifact ships.
#
# kage always drives a real headless Chrome, so unlike a plain CLI image this one
# bundles Chromium. KAGE_CHROME points kage at the system binary so it never
# tries to download its own.
#
# GoReleaser builds one multi-platform image with buildx and stages each
# platform's binary under a $TARGETPLATFORM directory (e.g. linux/amd64/) in the
# build context, so the COPY line selects the right one through the automatic
# TARGETPLATFORM build arg.
FROM alpine:3.21

ARG TARGETPLATFORM

# chromium for rendering; ca-certificates for HTTPS; tzdata for sane timestamps;
# the font package so rendered pages have glyphs to lay out.
RUN apk add --no-cache chromium ca-certificates tzdata font-noto \
 && adduser -D -H -u 10001 kage \
 && mkdir -p /out \
 && chown kage:kage /out

COPY $TARGETPLATFORM/kage /usr/bin/kage

USER kage
WORKDIR /out

# Point kage at the bundled Chromium and write mirrors under /out by default:
#
#   docker run -v "$PWD/out:/out" ghcr.io/tamnd/kage clone example.com
ENV KAGE_CHROME=/usr/bin/chromium-browser

VOLUME ["/out"]

ENTRYPOINT ["/usr/bin/kage"]
