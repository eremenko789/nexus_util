FROM nexus.redkit-lab.work:8084/rkl/dev-scada-debian-10:v1.3
RUN mkdir -p /gosetup && cd /gosetup && wget https://go.dev/dl/go1.25.4.linux-amd64.tar.gz &&  rm -rf /usr/local/go && tar -C /usr/local -xzf go1.25.4.linux-amd64.tar.gz && ln -sf /usr/local/go/bin/go /usr/bin/go && go version 
RUN git config --global --add safe.directory '*'
