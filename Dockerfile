# Copyright (c) 2020 Intel Corporation.

# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:

# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.

# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.

# Dockerfile for InfluxDBConnector

ARG EII_VERSION
ARG UBUNTU_IMAGE_VERSION
ARG ARTIFACTS="/artifacts"
FROM ia_common:$EII_VERSION as common
FROM ia_eiibase:${EII_VERSION} as builder
LABEL description="InfluxDBConnector image"

ARG INFLUXDB_GO_PATH=${GOPATH}/src/github.com/influxdata/influxdb
RUN mkdir -p ${INFLUXDB_GO_PATH} && \
    git clone https://github.com/influxdata/influxdb ${INFLUXDB_GO_PATH} && \
    cd ${INFLUXDB_GO_PATH} && \
    git checkout -b v1.6.0 tags/v1.6.0

# Installing influxdb
ARG INFLUXDB_VERSION
RUN wget -q --show-progress https://dl.influxdata.com/influxdb/releases/influxdb_${INFLUXDB_VERSION}_amd64.deb && \
    dpkg -i influxdb_${INFLUXDB_VERSION}_amd64.deb && \
    rm -rf influxdb_${INFLUXDB_VERSION}_amd64.deb

WORKDIR ${GOPATH}/src/IEdgeInsights
ARG CMAKE_INSTALL_PREFIX
ENV CMAKE_INSTALL_PREFIX=${CMAKE_INSTALL_PREFIX}
COPY --from=common ${CMAKE_INSTALL_PREFIX}/include ${CMAKE_INSTALL_PREFIX}/include
COPY --from=common ${CMAKE_INSTALL_PREFIX}/lib ${CMAKE_INSTALL_PREFIX}/lib
COPY --from=common /eii/common/util/influxdb ./InfluxDBConnector/util/influxdb
COPY --from=common /eii/common/util/util.go ./InfluxDBConnector/util/util.go
#COPY --from=common /eii/common/go.mod common/
COPY --from=common ${GOPATH}/src ${GOPATH}/src
COPY --from=common /eii/common/libs/EIIMessageBus/go/EIIMessageBus $GOPATH/src/EIIMessageBus
COPY --from=common /eii/common/libs/ConfigMgr/go/ConfigMgr $GOPATH/src/ConfigMgr

COPY . ./InfluxDBConnector
RUN cp InfluxDBConnector/config/influxdb.conf /etc/influxdb/ && \
    cp InfluxDBConnector/config/influxdb_devmode.conf /etc/influxdb/

ENV PATH="$PATH:/usr/local/go/bin" \
    PKG_CONFIG_PATH="$PKG_CONFIG_PATH:${CMAKE_INSTALL_PREFIX}/lib/pkgconfig" \
    LD_LIBRARY_PATH="${LD_LIBRARY_PATH}:${CMAKE_INSTALL_PREFIX}/lib"

# These flags are needed for enabling security while compiling and linking with cpuidcheck in golang
ENV CGO_CFLAGS="$CGO_FLAGS -I ${CMAKE_INSTALL_PREFIX}/include -O2 -D_FORTIFY_SOURCE=2 -Werror=format-security -fstack-protector-strong -fPIC" \
    CGO_LDFLAGS="$CGO_LDFLAGS -L${CMAKE_INSTALL_PREFIX}/lib -z noexecstack -z relro -z now"

ARG ARTIFACTS
RUN mkdir $ARTIFACTS && \
    cd InfluxDBConnector && \
    GO111MODULE=on go build -o $ARTIFACTS/InfluxDBConnector InfluxDBConnector.go

RUN mv InfluxDBConnector/schema.json $ARTIFACTS && \
    mv InfluxDBConnector/startup.sh $ARTIFACTS && \
    mv InfluxDBConnector/influx_start.sh $ARTIFACTS

FROM ubuntu:$UBUNTU_IMAGE_VERSION as runtime
ARG ARTIFACTS

RUN apt update && apt install --no-install-recommends -y libcjson1 libzmq5 zlib1g

WORKDIR /app
ARG CMAKE_INSTALL_PREFIX
ENV CMAKE_INSTALL_PREFIX=${CMAKE_INSTALL_PREFIX}
COPY --from=builder ${CMAKE_INSTALL_PREFIX}/lib ${CMAKE_INSTALL_PREFIX}/lib
COPY --from=builder /usr/bin/influxd /usr/bin/influxd
COPY --from=builder /usr/bin/influx /usr/bin/influx
COPY --from=builder /etc/influxdb /etc/influxdb
COPY --from=builder $ARTIFACTS .

ARG EII_UID
ARG EII_USER_NAME
RUN groupadd $EII_USER_NAME -g $EII_UID && \
    useradd -r -u $EII_UID -g $EII_USER_NAME $EII_USER_NAME
RUN mkdir -p /etc/ssl/influxdb && \
    mkdir -p /etc/ssl/ca && \
    mkdir -p /tmp/influxdb/log && \
    mkdir -p /influxdata && \
    touch /tmp/influxdb/log/influxd.log && \
    chown -R ${EII_UID} /etc/ssl/influxdb && \
    chown -R ${EII_UID} /etc/ssl/ca && \
    chown -R ${EII_UID}:${EII_UID} /influxdata && \
    chown -R ${EII_UID}:${EII_UID} /tmp/influxdb && \
    chmod -R 760 /influxdata && \
    chmod -R 760 /tmp/influxdb
USER $EII_USER_NAME

ENV LD_LIBRARY_PATH ${LD_LIBRARY_PATH}:${CMAKE_INSTALL_PREFIX}/lib
HEALTHCHECK NONE
ENTRYPOINT ["./startup.sh"]
