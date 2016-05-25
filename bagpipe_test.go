// Copyright 2015 CNI authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"net"
	"encoding/json"
//	"syscall"

	"github.com/containernetworking/cni/pkg/ns"
	//"github.com/containernetworking/cni/pkg/ip"
	//"github.com/containernetworking/cni/pkg/skel"
///	"github.com/containernetworking/cni/pkg/testutils"
	"github.com/containernetworking/cni/pkg/types"

//	"github.com/vishvananda/netlink"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("bridge Operations", func() {
	var originalNS ns.NetNS

	BeforeEach(func() {
		// Create a new NetNS so we don't modify the host
		var err error
		originalNS, err = ns.NewNS()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(originalNS.Close()).To(Succeed())
	})

	It("creates BGP json for Bagpipe", func() {

		conf := &NetConf{
			NetConf: types.NetConf{
				Name: "testConfig",
				Type: "bagpipe",
			},
			ImportRT: "12345:90",
			ExportRT: "12345:90",
			IsGW:   false,
			IPMasq: false,
			MTU:    5000,
		}
		defer GinkgoRecover()

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
		mes, err := createBGPConf(conf, "veth1234", "10.22.0.1", "10.22.0.2", "00:01:02:03:04:05")
		Expect(err).NotTo(HaveOccurred())
		var js Message
		json.Unmarshal([]byte(mes), &js)
		Expect(js.Import_rt[0]).To(Equal("12345:90"))
		Expect(js.Export_rt[0]).To(Equal("12345:90"))
		Expect(js.Ip_address).To(Equal("10.22.0.2"))
		Expect(js.Mac_address).To(Equal("00:01:02:03:04:05"))
		Expect(js.Vpn_instance_id).To(Equal("veth1234"))

	})

	It("sets Veth UP", func() {

		IFNAME := "eth0"

		conf := &NetConf{
			NetConf: types.NetConf{
				Name: "testConfig",
				Type: "bagpipe",
			},
			ImportRT: "12345:90",
			ExportRT: "12345:90",
			IsGW:   false,
			IPMasq: false,
			MTU:    5000,
		}

		var contMacAddr string

		macAddr, _, err := setupVeth(originalNS, IFNAME, conf.MTU)
		Expect(err).NotTo(HaveOccurred())
		err = originalNS.Do(func(ns.NetNS) error {
			interfaces, _ := net.Interfaces()
			for _, inter := range interfaces {
				if inter.Name != "lo" {
					contMacAddr = fmt.Sprintf("%v", inter.HardwareAddr)
				}
			}
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(macAddr).To(Equal(contMacAddr))

		defer GinkgoRecover()
	})

})
