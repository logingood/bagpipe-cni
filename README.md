# bagpipe plugin

## Overview

## Example configuration

```
{
  "importrt": "12345:101",
  "exportrt": "12345:101",
  "IsGW": "false",
  "IPMasq": "false",
  "MTU": "1500", 
	"ipam": {
		"type": "host-local",
		"subnet": "10.1.2.0/24",
	}
}
```

## Network configuration reference

* `name` (string, required): the name of the network.
* `type` (string, required): "bagpipe".
* `importrt` (string, required): import community
* `exportrt` (string, required): export community
* `mtu` (integer, optional): explicitly set MTU to the specified value. Defaults to the value chosen by the kernel.

## TODO
1. GW allocation
2. Delete bagpipe bgp tunnel
