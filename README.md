# UNITYMETRICS

Unitymetrics is a tool written in Go for collecting usage and performance metrics from a Dell EMC Unity array and translating them in InfluxDB's line protocol.

It can be useful to send metrics in a InfluxDB database with the help of Telegraf.

## How to find the available metrics

In the Unity API, metrics are define by a path. For example, if you want to collect the remaining memory available on the storage processors, you'll have to use the path `sp.*.memory.summary.freeBytes`.

You can find a list of the metrics [here](https://gist.github.com/equelin/37486519972f8161c480f47ae5904390).

If you look at the different path, you will figure that some of them contains `*` or `+` characters.

When there is a `*` in the path, you can use the path as-is in your request, the `*` will be automatically replaced with all the possibilities. For example, if you want to use the path `sp.*.memory.summary.freeBytes`. The API will interpret it as if you were requesting the free memory for the SPA and the SPB. If you need this information only for one of the SPs, you can use the path `sp.spa.memory.summary.freeBytes` 

When there is a `+` in the path, you can replace it with the relevant item by yourself before requesting the API or by a `*` for breaking the results by this item. For example, if you want to specifically retrieve the CPU utilization of the SPA, you have to modify the path `kpi.sp.+.utilization` like this `kpi.sp.spa.utilization`.

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

```bash
./unitymetrics -h
```

### Run a Dell EMC Unity metrics collection for an historical path 

```bash
./unitymetrics -unity unity01.example.com -user admin -password AwesomePassword -histkpipaths kpi.sp.*.utilization
```

### Run a Dell EMC Unity metrics collection with a real time metric and sampling interval of 10 seconds

```bash
./unitymetrics -unity unity01.example.com -user admin -password AwesomePassword -rtpaths sp.*.memory.summary.freeBytes -interval 10
```

### Run a Dell EMC Unity metrics collection with multiple metrics and sampling interval of 10 seconds

```bash
./unitymetrics -unity unity01.example.com -user admin -password AwesomePassword -interval 10 -histkpipaths kpi.sp.*.utilization,kpi.lun.+.sp.+.rw.+.throughput,kpi.lun.*.sp.+.rw.+.throughput,kpi.lun.+.sp.+.responseTime,kpi.lun.*.sp.+.responseTime,kpi.lun.+.sp.+.queueLength,kpi.lun.*.sp.+.queueLength
```

### Run a Dell EMC Unity metrics collection for collecting capacity statistics

```bash
./unitymetrics -unity unity01.example.com -user admin -password AwesomePassword -capacity
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
  commands = ["unitymetrics -user admin -password Mypassword -unity unity01.example.com -histkpipaths kpi.sp.*.utilization,kpi.lun.+.sp.+.rw.+.throughput,kpi.lun.*.sp.+.rw.+.throughput,kpi.lun.+.sp.+.responseTime,kpi.lun.*.sp.+.responseTime,kpi.lun.+.sp.+.queueLength,kpi.lun.*.sp.+.queueLength -capacity"]

  # Timeout for each command to complete.
  timeout = "60s"

  # Data format to consume.
  # NOTE json only reads numerical measurements, strings and booleans are ignored.
  data_format = "influx"

  interval = "60s"
```

If needed, you can specify more than one input plugin. It might be useful if you want to gather different statistics with different intervals or if you want to query different arrays.

```Toml
###############################################################################
#                            INPUT PLUGINS                                    #
###############################################################################

[[inputs.exec]]
  # Shell/commands array
  # Full command line to executable with parameters, or a glob pattern to run all matching files.
  commands = ["unitymetrics -user admin -password Mypassword -unity unity01.example.com -histkpipaths kpi.sp.*.utilization,kpi.lun.+.sp.+.rw.+.throughput,kpi.lun.*.sp.+.rw.+.throughput,kpi.lun.+.sp.+.responseTime,kpi.lun.*.sp.+.responseTime,kpi.lun.+.sp.+.queueLength,kpi.lun.*.sp.+.queueLength -capacity"]

  # Timeout for each command to complete.
  timeout = "60s"

  # Data format to consume.
  # NOTE json only reads numerical measurements, strings and booleans are ignored.
  data_format = "influx"

  interval = "60s"

[[inputs.exec]]
  # Shell/commands array
  # Full command line to executable with parameters, or a glob pattern to run all matching files.
  commands = ["unitymetrics -user admin -password Mypassword -unity unity01.example.com -interval 50 -rtpaths sp.*.memory.summary.freeBytes,sp.*.memory.summary.totalBytes,sp.*.memory.summary.totalUsedBytes,sp.*.cpu.uptime"]

  # Timeout for each command to complete.
  timeout = "60s"

  # Data format to consume.
  # NOTE json only reads numerical measurements, strings and booleans are ignored.
  data_format = "influx"

  interval = "60s"

[[inputs.exec]]
  # Shell/commands array
  # Full command line to executable with parameters, or a glob pattern to run all matching files.
  commands = ["unitymetrics -user admin -password Mypassword -unity unity02.example.com -histkpipaths kpi.sp.*.utilization,kpi.lun.+.sp.+.rw.+.throughput,kpi.lun.*.sp.+.rw.+.throughput,kpi.lun.+.sp.+.responseTime,kpi.lun.*.sp.+.responseTime,kpi.lun.+.sp.+.queueLength,kpi.lun.*.sp.+.queueLength -capacity"]

  # Timeout for each command to complete.
  timeout = "60s"

  # Data format to consume.
  # NOTE json only reads numerical measurements, strings and booleans are ignored.
  data_format = "influx"

  interval = "60s"
```

# Author
**Erwan Quélin**
- <https://github.com/equelin>
- <https://twitter.com/erwanquelin>

# License

Copyright 2018 Erwan Quelin and the community.

Licensed under the MIT License.

