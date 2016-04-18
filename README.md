# BaGPipe BGP CNI plugin
## Overview

This plugins allows to confiure container networking with VXLAN and advertise container's network as VXLAN + IP + Mac object using EVPN BGP route. 
You must have bagpipe-bgp installed on the host machine:

https://github.com/Orange-OpenSource/bagpipe-bgp

Probably have a route reflector (BGP router that supports AFI=25 SAFI=70) because bagpipe-bgp allows only to run two nodes without RR.

Nice example of go-based RR: https://github.com/osrg/gobgp and EVPN lab example can be found here https://github.com/osrg/gobgp/blob/master/docs/sources/evpn.md

## Install

The easiest way to install plugin:
clone CNI repositority: https://github.com/appc/cni 

```
git clone https://github.com/appc/cni
cd cni/plugins/main
```

Install bagpipe CNI plugin into plugins/main/bagpipe

```
git clone https://github.com/murat1985/cni-bagpipe-bgp bagpipe
cd ../../
```

Build plugins

```
./build
```

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
