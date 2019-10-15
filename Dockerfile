# Dockerfile for InfluxDBConnector
ARG EIS_VERSION
FROM ia_eisbase:${EIS_VERSION} as eisbase
LABEL description="InfluxDBConnector image"

WORKDIR ${GO_WORK_DIR}

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

RUN mkdir -p /etc/ssl/influxdb && \
    mkdir -p /etc/ssl/ca

FROM ia_common:$EIS_VERSION as common

FROM eisbase

COPY --from=common /libs ${GO_WORK_DIR}/libs
COPY --from=common /util ${GO_WORK_DIR}/util

RUN cd ${GO_WORK_DIR}/libs/EISMessageBus && \
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

RUN ln -s ${GO_WORK_DIR}/libs/EISMessageBus/go/EISMessageBus/ $GOPATH/src/EISMessageBus

COPY . ./InfluxDBConnector/

RUN go build -o /EIS/go/bin/InfluxDBConnector InfluxDBConnector/InfluxDBConnector.go
ARG EIS_UID
ARG EIS_USER_NAME
RUN chown ${EIS_UID} ${GO_WORK_DIR}
RUN chown -R ${EIS_UID} /etc/ssl/influxdb && \
    chown -R ${EIS_UID} /etc/ssl/ca 
ENTRYPOINT ["InfluxDBConnector"]

