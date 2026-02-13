FROM debian:10

ARG DEBIAN_FRONTEND=noninteractive

# Disable SSL verification for apt
RUN echo 'Acquire::https::Verify-Peer "false";' > /etc/apt/apt.conf.d/99disable-ssl-verify

# Set custom apt sources
RUN echo 'deb [trusted=yes] https://nexus-proxy.redkit-lab.work/repository/apt-proxy-debian-buster/ buster main contrib non-free' > /etc/apt/sources.list && \
    echo 'deb [trusted=yes] https://nexus-proxy.redkit-lab.work/repository/apt-proxy-debian-buster-updates/ buster-updates main contrib non-free' >> /etc/apt/sources.list && \
    echo 'deb [trusted=yes] https://nexus-proxy.redkit-lab.work/repository/apt-proxy-debian-buster-security/ buster/updates main contrib non-free' >> /etc/apt/sources.list

RUN apt-get update && apt-get install -y --no-install-recommends \\
    make \
    wget \
    ca-certificates \
    git \
    gcc \
    libc6-dev \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /gosetup && cd /gosetup && wget https://go.dev/dl/go1.25.4.linux-amd64.tar.gz && rm -rf /usr/local/go && tar -C /usr/local -xzf go1.25.4.linux-amd64.tar.gz && ln -sf /usr/local/go/bin/go /usr/bin/go && go version
RUN git config --global --add safe.directory '*'
