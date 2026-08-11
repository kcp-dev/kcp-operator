# Builds the mock virtual workspace server used by the e2e tests to stand in for a virtual
# workspace that is not kcp's own. See test/mock-virtualworkspace for what it does.
#
# Build context is the repository root:
#   docker build --file hack/ci/testdata/mock-vw/Dockerfile .
FROM --platform=${BUILDPLATFORM} docker.io/golang:1.26.4 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
COPY sdk/go.mod sdk/go.mod
COPY sdk/go.sum sdk/go.sum
RUN go mod download

COPY sdk/ sdk/
COPY internal/ internal/
COPY test/ test/

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -a -o mock-virtualworkspace ./test/mock-virtualworkspace/

FROM gcr.io/distroless/static:nonroot
LABEL org.opencontainers.image.source=https://github.com/kcp-dev/kcp-operator
LABEL org.opencontainers.image.description="Mock virtual workspace server for kcp-operator e2e tests"
LABEL org.opencontainers.image.licenses=Apache-2.0

WORKDIR /
COPY --from=builder /workspace/mock-virtualworkspace .
USER 65532:65532

ENTRYPOINT ["/mock-virtualworkspace"]
