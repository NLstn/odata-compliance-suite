# GoReleaser (dockers_v2) builds the static binaries and runs `docker buildx`
# with them already staged in the build context under per-platform directories
# (e.g. linux/amd64/compliance-test), so this image only needs to copy the
# right one in. $TARGETPLATFORM is provided automatically by buildx.
# (For a standalone `docker build` without GoReleaser, see the build-from-source
# stage below — uncomment it.)
FROM gcr.io/distroless/static:nonroot

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/compliance-test /usr/local/bin/compliance-test

# Args after the image name are passed straight to the suite, e.g.:
#   docker run ghcr.io/nlstn/odata-compliance-suite:v1 -server http://host:9090
ENTRYPOINT ["/usr/local/bin/compliance-test"]

# --- Build-from-source alternative (not used by the release pipeline) ---------
# Uncomment to build the image directly from source with `docker build .`:
#
# FROM golang:1.24-alpine AS build
# WORKDIR /src
# COPY . .
# RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /compliance-test .
#
# FROM gcr.io/distroless/static:nonroot
# COPY --from=build /compliance-test /usr/local/bin/compliance-test
# ENTRYPOINT ["/usr/local/bin/compliance-test"]
