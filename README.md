# udp-forwarder

Simple udp packet forwarder preserving (spoofing)  the original IP that sent the packet

Configuration file example:

```json
{
    "interface_name" : "eth0",
    "ip_address_receiver" : "192.168.1.32:1162",
    "default_gateway" : "192.168.1.1",
    "destinations" : [
        "192.168.1.31:2123"
    ]
}
```