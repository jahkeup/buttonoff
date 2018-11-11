# button: on,off - Button Press Packet Sniffing Daemon

This project aims to support the sniffing and publishing of detected
press events to an MQTT broker.

## Building

```bash
go get github.com/jahkeup/buttonoff/cmd/buttonoffd
```

## Using


With configuration provided:

```bash
buttonoffd -interface eth0 -broker mqtt.example.com:1883 -config ./buttonoff.toml
```

Configuration can be generated with:

```bash
# Write a simple defualt config to ./generated-config.toml
buttonffd -overwrite-default -config ./generated-config.toml
```

Some flags can be overriden in the configuration file. See the
generated file for some examples.

The `[general]` section can specify `dropunconfigured = true` to
drop sniffed events instead of publishing to the mqtt broker - the
default is to publish all events to the broker.

