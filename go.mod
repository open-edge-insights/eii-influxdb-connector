module influxdbconnector

go 1.15

require (
        github.com/influxdata/influxdb v1.6.0
)
replace influxdbconnector => ./

replace github.com/open-edge-insights/eii-configmgr-go => ../../ConfigMgr/

replace github.com/open-edge-insights/eii-messagebus-go => ../../EIIMessageBus/

