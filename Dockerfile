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

ARG EIS_VERSION
ARG DOCKER_REGISTRY
FROM ${DOCKER_REGISTRY}ia_eisbase:${EIS_VERSION} as eisbase
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

FROM ${DOCKER_REGISTRY}ia_common:$EIS_VERSION as common

FROM eisbase

COPY --from=common ${GO_WORK_DIR}/common/libs ${GO_WORK_DIR}/common/libs
COPY --from=common ${GO_WORK_DIR}/common/util ${GO_WORK_DIR}/common/util
COPY --from=common ${GO_WORK_DIR}/common/cmake ${GO_WORK_DIR}/common/cmake
COPY --from=common /usr/local/include /usr/local/include
COPY --from=common /usr/local/lib /usr/local/lib
COPY --from=common ${GO_WORK_DIR}/../EISMessageBus ${GO_WORK_DIR}/../EISMessageBus
COPY --from=common ${GO_WORK_DIR}/../ConfigManager ${GO_WORK_DIR}/../ConfigManager
COPY --from=common ${GO_WORK_DIR}/../EnvConfig ${GO_WORK_DIR}/../EnvConfig

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

