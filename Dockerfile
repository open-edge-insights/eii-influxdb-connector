# Dockerfile for InfluxDBConnector
ARG IEI_VERSION
FROM ia_gobase:${IEI_VERSION}
LABEL description="InfluxDBConnector image"

RUN mkdir -p ${GO_WORK_DIR}/log && \
    apt-get update

ENV INFLUXDB_GO_PATH ${GOPATH}/src/github.com/influxdata/influxdb
RUN mkdir -p ${INFLUXDB_GO_PATH} && \
    git clone https://github.com/influxdata/influxdb ${INFLUXDB_GO_PATH} && \
    cd ${INFLUXDB_GO_PATH} && \
    git checkout -b v1.6.0 tags/v1.6.0

# Installing influxdb
ARG INFLUXDB_VERSION
RUN wget https://dl.influxdata.com/influxdb/releases/influxdb_${INFLUXDB_VERSION}_amd64.deb && \
    dpkg -i influxdb_${INFLUXDB_VERSION}_amd64.deb && \
    rm -rf influxdb_${INFLUXDB_VERSION}_amd64.deb

COPY libs/EISMessageBus ./libs/EISMessageBus
RUN cd /IEI/go/src/IEdgeInsights/libs/EISMessageBus && \
    rm -rf build deps && mkdir -p build && cd build && \
    cmake -DWITH_GO=ON .. && \
    make && \
    make install

ENV MSGBUS_DIR $GO_WORK_DIR/libs/EISMessageBus
ENV LD_LIBRARY_PATH $LD_LIBRARY_PATH:$MSGBUS_DIR/build/
ENV PKG_CONFIG_PATH $PKG_CONFIG_PATH:$MSGBUS_DIR/build/
ENV CGO_CFLAGS -I$MSGBUS_DIR/include/
ENV CGO_LDFLAGS -L$MSGBUS_DIR/build -leismsgbus
ENV LD_LIBRARY_PATH ${LD_LIBRARY_PATH}:/usr/local/lib

RUN ln -s /IEI/go/src/IEdgeInsights/libs/EISMessageBus/go/EISMessageBus/ $GOPATH/src/EISMessageBus

COPY Util ./Util
COPY libs/ConfigManager ./libs/ConfigManager
COPY libs/common ./libs/common
COPY InfluxDBConnector ./InfluxDBConnector

RUN go build -o /IEI/go/bin/InfluxDBConnector InfluxDBConnector/InfluxDBConnector.go
ARG IEI_UID
ARG IEI_USER_NAME
RUN chown ${IEI_UID} /IEI/go/src/IEdgeInsights
RUN chown -R ${IEI_UID} /etc/ssl/
ENTRYPOINT ["InfluxDBConnector"]
HEALTHCHECK NONE

