# Dockerfile for InfluxDBConnector
ARG EIS_VERSION
FROM ia_gobase:${EIS_VERSION}
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



ADD InfluxDBConnector.go ./InfluxDBConnector/InfluxDBConnector.go
ADD common ./InfluxDBConnector/common
ADD configManager ./InfluxDBConnector/configManager
ADD pubManager ./InfluxDBConnector/pubManager
ADD subManager ./InfluxDBConnector/subManager
ADD dbManager ./InfluxDBConnector/dbManager

RUN go build -o /EIS/go/bin/InfluxDBConnector InfluxDBConnector/InfluxDBConnector.go
ARG EIS_UID
ARG EIS_USER_NAME
RUN chown ${EIS_UID} /EIS/go/src/IEdgeInsights
RUN chown -R ${EIS_UID} /etc/ssl/
ENTRYPOINT ["InfluxDBConnector"]
HEALTHCHECK NONE

