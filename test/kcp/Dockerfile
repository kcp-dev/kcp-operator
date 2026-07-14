# This Dockerfile creates a container image with kcp's source code in it, so
# we can deploy it as a Pod inside a kind cluster and then run the kcp e2e
# tests against the kcp-operator managed environment.

FROM docker.io/golang:1.26.2

ENV HTTEST_VERSION="0.3.4"
RUN curl --fail -LO https://codeberg.org/xrstf/httest/releases/download/v${HTTEST_VERSION}/httest_${HTTEST_VERSION}_linux_$(dpkg --print-architecture).tar.gz && \
    tar xzf httest_*.tar.gz && \
    mv httest_*/httest /usr/local/bin

WORKDIR /apps/kcp
RUN git clone --depth 1 https://github.com/kcp-dev/kcp . && \
    go build -v ./test/...

ENV NO_GORUN=1

CMD [ "bash", "-c", "go test -parallel 1 ./test/e2e/... -args --kcp-kubeconfig $KUBECONFIG" ]
