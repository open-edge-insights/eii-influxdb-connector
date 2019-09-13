# `InfluxDBConnector Module`

1. InfluxDBConnector will subscribe to the InfluxDB and start the zmq
   publisher, zmq subscriber threads, and zmq request reply thread
   based on PubTopics, SubTopics and QueryTopics configuration.
2. zmq subscriber thread connects to the PUB socket of zmq bus on which
   the data is published by VideoAnalytics and push it to the InfluxDB 
3. zmq publisher thread will publish the point data ingested by the telegraf
   and the classifier result coming out of the point data analytics.
4. zmq reply request service will receive the InfluxDB select query and 
   response with the historical data.

## `Configuration`

All the InfluxDBConnector module configuration are added into etcd (distributed
key-value data store) under `AppName` as mentioned in the
environment section of this app's service definition in docker-compose.

If `AppName` is `InfluxDBConnector`, then the app's config would look like as below
 for `/InfluxDBConnector/config` key in Etcd:
 ```
    "influxdb": {
            "retention": "1h30m5s",
            "username": "admin",
            "password": "admin123",
            "dbname": "datain",
            "ssl": "True",
            "verifySsl": "False",
            "port": "8086"
        }
 ```

 For more details on Etcd and MessageBus endpoint configuration, visit [Etcd_and_MsgBus_Endpoint_Configuration](../Etcd_and_MsgBus_Endpoint_Configuration].md).


## `Installation`

* Follow [provision/README.md](../docker_setup/provision/README.md) for EIS provisioning

* Run InfluxDBConnector

  Present working directory to try out below commands is: `[repo]/InfluxDBConnector`

    1. Build and Run VideoAnalytics as container
        ```
        $ cd [repo]/docker_setup
        $ docker-compose up --build ia_influxdbconnector
        ```
