# UNITYMETRICS

Unitymetrics is a tool written in Go for collecting metrics from a Dell EMC Unity array and translating them in InfluxDB's line protocol. 

It can be use to send metrics in a InfluxDB database with the help of Telegraf.

## How to find the available metrics

In the Unity API, metrics are define by a path. For example, if you want to collect the remaining memory available on the storage processors, you'll have to use the path `sp.*.memory.summary.freeBytes`. 

You can find a list of the metrics [here](https://gist.github.com/equelin/37486519972f8161c480f47ae5904390).

If you look at the different path, you will figure that some of them contains `*` or `+` characters.

When there is a `*` in the path, you can use the path as-is in your request, the `*` will be automatically replaced with all the possibilities. For example, if you want to use the path `sp.*.memory.summary.freeBytes`. The API will interpret it as if you were requesting the free memory for the SPA and the SPB. If you need this information only for one of the SPs, you can use the path `sp.spa.memory.summary.freeBytes` 

When there is a `+` in the path, you have to replace it with the relevant item by yourself before requesting the API. For example, if you want to retrieve the CPU utilization of the SPA, you have to modify the path `kpi.sp.+.utilization` like this `kpi.sp.spa.utilization`   

## How to install it
### From prebuilt release

You can find prebuilt unitymetrics binaries on the [releases page](https://github.com/equelin/unitymetrics/releases).

Download and install a binary locally like this:

``` console
% curl -L $URL_TO_BINARY | gunzip > /usr/local/bin/unitymetrics
% chmod +x /usr/local/bin/unitymetrics
```

### From source

To build unitymetrics from source, first install the [Go toolchain](https://golang.org/dl/).

Make sure to set the environment variable [GOPATH](https://github.com/golang/go/wiki/SettingGOPATH).

You can then download the latest unitymetrics source code from github using:

``` console
% go get -u github.com/equelin/unitymetrics
```

Make sure `$GOPATH/bin` is in your `PATH`.

You can build unitymetrics using:

``` console
% cd $GOPATH/src/github.com/equelin/unitymetrics/
% make
```

## How to use it

See usage with:

```
./unitymetrics -h
```

#### Run a Dell Unity metrics collection with the default metrics and a sampling interval

```
./unitymetrics -unity unity01.example.com -user admin -password AwesomePassword
```

#### Run a Dell Unity metrics collection with the default metrics and sampling interval of 10 seconds

```
./unitymetrics -unity unity01.example.com -user admin -password AwesomePassword -interval 10
```

#### Run a Dell Unity metrics collection with specific metrics and sampling interval of 10 seconds

```
./unitymetrics -unity unity01.example.com -user admin -password AwesomePassword -interval 10 -paths kpi.sp.spa.utilization,sp.*.cpu.summary.busyTicks,sp.*.cpu.uptime,sp.*.storage.pool.*.sizeFree,
```

## Using unitymetrics with Telegraf

The `exec` input plugin of Telegraf executes the `commands` on every interval and parses metrics from their output in any one of the accepted [Input Data Formats](https://github.com/influxdata/telegraf/blob/master/docs/DATA_FORMATS_INPUT.md).

`unitymetrics` output the metrics in InfluxDB's line protocol. Telegraf will parse them and send them to the InfluxDB database.

> Don't forget to configure Telegraf to output data to InfluxDB !

Here is an example of a working telegraf's config file:

```Toml
###############################################################################
#                            INPUT PLUGINS                                    #
###############################################################################

[[inputs.exec]]
  # Shell/commands array
  # Full command line to executable with parameters, or a glob pattern to run all matching files.
  commands = ["unitymetrics -user admin -password AwesomePassword -unity unity01.okcomputer.lab -interval 10 -paths kpi.sp.spa.utilization,sp.*.cpu.summary.busyTicks,sp.*.cpu.uptime,sp.*.storage.pool.*.sizeFree,sp.*.storage.pool.*.sizeSubscribed,sp.*.storage.pool.*.sizeTotal,sp.*.storage.pool.*sizeUsed,sp.*.memory.summary.totalBytes,sp.*.memory.summary.totalUsedBytes"]

  # Timeout for each command to complete. 
  timeout = "20s"

  # Data format to consume.
  # NOTE json only reads numerical measurements, strings and booleans are ignored.
  data_format = "influx"
```

# Author
**Erwan Quélin**
- <https://github.com/equelin>
- <https://twitter.com/erwanquelin>

# License

Copyright 2018 Erwan Quelin and the community.

Licensed under the MIT License.

