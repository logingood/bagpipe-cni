package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime"

	"github.com/containernetworking/cni/pkg/ip"
	"github.com/containernetworking/cni/pkg/ipam"
	"github.com/containernetworking/cni/pkg/ns"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/utils"
	"github.com/safchain/ethtool"
	"github.com/vishvananda/netlink"
)

// Adding additional configuration for EVPN - importRT and exportRT
type NetConf struct {
	types.NetConf
	ImportRT string `json:"importrt"`
	ExportRT string `json:"exportrt"`
	IsGW     bool   `json:"isGateway"`
	IPMasq   bool   `json:"ipMasq"`
	MTU      int    `json:"mtu"`
}

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

func loadNetConf(bytes []byte) (*NetConf, error) {
	n := &NetConf{
		MTU: 1500,
	}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, fmt.Errorf("failed to load netconf: %v", err)
	}
	return n, nil
}

func createBGPConf(n *NetConf, LocalPort string, gw string, ipa string, mac string) (b []byte, err error) {

	// JSON payload struct for bagpipeBGP
	type Message struct {
		Import_rt        []string         `json:"import_rt"`
		Vpn_type         string           `json:"vpn_type"`
		Vpn_instance_id  string           `json:"vpn_instance_id"`
		Ip_address       string           `json:"ip_address"`
		Export_rt        []string         `json:"export_rt"`
		Local_port       *json.RawMessage `json:"local_port"`
		Readvertise      string           `json:"readvertise"`
		Gateway_ip       string           `json:"gateway_ip"`
		Mac_address      string           `json:"mac_address"`
		Advertise_subnet bool             `json:"advertise_subnet"`
	}

	// Linuxif Raw JSON object
	type LcPorts struct {
		LinuxIf string `json:"linuxif"`
	}

	tmp := LcPorts{LocalPort}
	l, err := json.Marshal(tmp)
	LcPort := json.RawMessage(l)

	Imports := []string{n.ImportRT}
	Exports := []string{n.ExportRT}

	m := Message{
		Import_rt:        Imports,
		Vpn_type:         "evpn",
		Vpn_instance_id:  LocalPort,
		Ip_address:       ipa,
		Export_rt:        Exports,
		Local_port:       &LcPort,
		Gateway_ip:       gw,
		Mac_address:      mac,
		Advertise_subnet: false,
	}

	b, err = json.Marshal(m)
	return b, err
}

func sendBagpipeReq(n *NetConf, Request string, LocalPort string, gw string, ipa string, mac string) error {
	// send json payload
	url := fmt.Sprintf("http://127.0.0.1:8082/%s_localport", Request)

	var jsonStr []byte
	// create JSON object for BagpipeBGP
	jsonStr, err := createBGPConf(n, LocalPort, gw, ipa, mac)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	return err
}

func setupVeth(netns ns.NetNS, ifName string, mtu int) (contMacAddr string, hostVethName string, err error) {

	err = netns.Do(func(hostNS ns.NetNS) error {
		// create the veth pair in the container and move host end into host netns
		hostVeth, _, err := ip.SetupVeth(ifName, mtu, hostNS)
		if err != nil {
			return nil
		}

		hostVethName = hostVeth.Attrs().Name
		interfaces, _ := net.Interfaces()

		// Lookup MAC address of eth0 inside namespace
		for _, inter := range interfaces {
			if inter.Name != "lo" {
				contMacAddr = fmt.Sprintf("%v", inter.HardwareAddr)
			}

		}
		return nil
	})
	if err != nil {
		return contMacAddr, hostVethName, err
	}

	if err != nil {
		return contMacAddr, hostVethName, fmt.Errorf("failed to lookup %q: %v", hostVethName, err)
	}

	return contMacAddr, hostVethName, err
}

func calcGatewayIP(ipn *net.IPNet) net.IP {
	nid := ipn.IP.Mask(ipn.Mask)
	return ip.NextIP(nid)
}

func cmdAdd(args *skel.CmdArgs) error {
	n, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}
	var vethName string
	var macAddr string

	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer netns.Close()

	if macAddr, vethName, err = setupVeth(netns, args.IfName, n.MTU); err != nil {
		return err
	}

	// run the IPAM plugin and get back the config to apply
	result, err := ipam.ExecAdd(n.IPAM.Type, args.StdinData)
	if err != nil {
		return err
	}

	if result.IP4 == nil {
		return errors.New("IPAM plugin returned missing IPv4 config")
	}

	if result.IP4.Gateway == nil && n.IsGW {
		result.IP4.Gateway = calcGatewayIP(&result.IP4.IP)
	}

	err = netns.Do(func(_ ns.NetNS) error {
		return ipam.ConfigureIface(args.IfName, result)
	})
	if err != nil {
		return err
	}

	if n.IPMasq {
		chain := utils.FormatChainName(n.Name, args.ContainerID)
		comment := utils.FormatComment(n.Name, args.ContainerID)
		if err = ip.SetupIPMasq(ip.Network(&result.IP4.IP), chain, comment); err != nil {
			return err
		}
	}

	var ip_gw, ip_addr string

	ip_gw = fmt.Sprintf("%v", &result.IP4.Gateway)
	ip_addr = fmt.Sprintf("%v", &result.IP4.IP)

	sendBagpipeReq(n, "attach", vethName, ip_gw, ip_addr, macAddr)

	result.DNS = n.DNS
	return result.Print()
}

func cmdDel(args *skel.CmdArgs) error {

	// bagpipe detach should be implemented
	n, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}

	err = ipam.ExecDel(n.IPAM.Type, args.StdinData)
	if err != nil {
		return err
	}

	var contMacAddr string
	var hostVethName string
	var ip_gw string
	var ip_addr string
	var if_index uint64

	//	err = ns.WithNetNSPath(args.Netns, false, func(hostNS *os.File) error {
	err = ns.WithNetNSPath(args.Netns, func(_ ns.NetNS) error {
		link, _ := netlink.LinkByName(args.IfName)

		// Getting container MAC address
		contMacAddr = fmt.Sprintf("%s", link.Attrs().HardwareAddr)
		// Getting container IP address, should be rewritten PoC
		cont_IP, _ := netlink.AddrList(link, netlink.FAMILY_V4)
		prefix, _ := cont_IP[0].IPNet.Mask.Size()
		ip_addr = fmt.Sprintf("%s/%d", cont_IP[0].IPNet.IP, prefix)

		// Getting peer interface index in Root namespace
		stats, _ := ethtool.Stats(args.IfName)
		if_index = stats["peer_ifindex"]

		// Getting default gateway address - should be rewritten PoC
		gw_route, _ := netlink.RouteGet(net.ParseIP("1.2.3.4"))
		ip_gw = fmt.Sprintf("%s", gw_route[0].Gw)

		return nil
	})

	// Getting hostlink based on if_index in Root NS
	hostlink, _ := netlink.LinkByIndex(int(if_index))
	hostVethName = hostlink.Attrs().Name

	// Dettaching the route from BaGPipe BGP
	sendBagpipeReq(n, "detach", hostVethName, ip_gw, ip_addr, contMacAddr)
	// Removing the link
	return ip.DelLinkByName(hostVethName)
}

func main() {
	skel.PluginMain(cmdAdd, cmdDel)
}
