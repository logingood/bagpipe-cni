# BaGPipe BGP CNI plugin
## Overview

This plugins allows to confiure container networking with VXLAN and advertise container's network as VXLAN + IP + Mac object using EVPN BGP route. 
You must have bagpipe-bgp installed on the host machine:

[BaGPipe BGP](https://github.com/Orange-OpenSource/bagpipe-bgp)

Probably have a route reflector (BGP router that supports AFI=25 SAFI=70) because bagpipe-bgp allows only to run two nodes without RR.

Nice example of go-based RR: [GoBGP](http://osrg.github.io/gobgp/) and EVPN lab example can be found here [EVPN with BaGPipe BGP and GoBGP RR](https://github.com/osrg/gobgp/blob/master/docs/sources/evpn.md)

## Install

The easiest way to install plugin is to clone CNI repositority: [CNI](https://github.com/appc/cni)

Make sure that GOPATH environment variable is set

```
cd $GOPATH
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

Supported IP manager plugins:

* [host-local IP address manager](https://github.com/containernetworking/cni/tree/master/plugins/ipam/host-local)
* [consul IP address manager](https://github.com/murat1985/cni-ipam-consul)

Just put the example below in file /etc/cni/net.d/10-mynet.conf 

```
{
  "name": "mynet",
  "type": "bagpipe",
  "importrt": "64512:90",
  "exportrt": "64512:90",
  "isGateway": false,
  "ipMasq": false,
  "mtu": "1500", 
    "ipam": {
      "type": "host-local",
      "subnet": "10.1.2.0/24",
      "routes": [
        { "dst": "0.0.0.0/0" }
      ]
    }
}
```

## Network configuration reference

* `name` (string, required): the name of the network.
* `type` (string, required): "bagpipe".
* `importrt` (string, required): import community
* `exportrt` (string, required): export community
* `mtu` (integer, optional): explicitly set MTU to the specified value. Defaults to the value chosen by the kernel.

## Usage with Kubernetes

I have published a blog post, where described [PoC: BaGPipe CNI and Kubernetes](http://murat1985.github.io/kubernetes/cni/2016/05/15/kubernetes.html)

## Usage with Docker

Assuming that cni installed in the $GOPATH/cni and bagpipe CNI plugin is installed in plugins/main/bagpipe
docker-run.sh script could be found in scripts directory of [CNI](https://github.com/appc/cni/blob/master/scripts/docker-run.sh) repository

```
cd $GOPATH/cni
CNI_PATH=`pwd`/bin
./build; cd scripts; CNI_PATH=$CNI_PATH ./docker-run.sh busybox sleep 1000 ; cd ..
```

Adding and Deleting of bagpipe tunnels along with container links fully implemented.

## Diagram 

![alt text](https://github.com/murat1985/bagpipe-cni/blob/master/diagrams/CNI-Bagpipe.png "BaGPipe BGP CNI plugin")

## TODO
1. GW allocation
