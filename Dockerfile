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

COPY --from=common ${GO_WORK_DIR}/common/libs ${GO_WORK_DIR}/common/libs
COPY --from=common ${GO_WORK_DIR}/common/util ${GO_WORK_DIR}/common/util
COPY --from=common ${GO_WORK_DIR}/common/cmake ${GO_WORK_DIR}/common/cmake
COPY --from=common /usr/local/include /usr/local/include
COPY --from=common /usr/local/lib /usr/local/lib
COPY --from=common ${GO_WORK_DIR}/../EISMessageBus ${GO_WORK_DIR}/../EISMessageBus
COPY --from=common ${GO_WORK_DIR}/../ConfigManager ${GO_WORK_DIR}/../ConfigManager

COPY . ./InfluxDBConnector/

RUN cp ${GO_WORK_DIR}/InfluxDBConnector/config/influxdb.conf /etc/influxdb/ && \
    cp ${GO_WORK_DIR}/InfluxDBConnector/config/influxdb_devmode.conf /etc/influxdb/
RUN go build -o /EIS/go/bin/InfluxDBConnector InfluxDBConnector/InfluxDBConnector.go
ARG EIS_UID
ARG EIS_USER_NAME
RUN chown ${EIS_UID} ${GO_WORK_DIR}
RUN chown -R ${EIS_UID} /etc/ssl/influxdb && \
    chown -R ${EIS_UID} /etc/ssl/ca 

RUN mkdir -p ${GOPATH}/temp/IEdgeInsights/InfluxDBConnector && \
    mv ${GO_WORK_DIR}/InfluxDBConnector/influx_start.sh ${GOPATH}/temp/IEdgeInsights/InfluxDBConnector/ && \
    rm -rf ${GOPATH}/src && \
    rm -rf ${GOPATH}/bin/dep && \
    rm -rf ${GOPATH}/pkg && \
    rm -rf /usr/local/go && \
    mv ${GOPATH}/temp ${GOPATH}/src

RUN chown -R ${EIS_UID} ${GOPATH}/src

#Removing build dependencies
RUN apt-get remove -y wget && \
    apt-get remove -y git && \
    apt-get remove curl && \
    apt-get autoremove -y

COPY schema.json .

ENTRYPOINT ["InfluxDBConnector"]

