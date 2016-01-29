package server

import (
	"asicd/asicdConstDefs"
	"fmt"
	"l3/bfd/config"
	"l3/bfd/protocol"
	"net"
	//"time"
	//"l3/bfd/rpc"
	//"l3/rib/ribdCommonDefs"
	//"github.com/google/gopacket/pcap"
)

func (server *BFDServer) initDefaultIntfConf(ifIndex int32, ipIntfProp IpIntfProperty) {
	intf, exist := server.bfdGlobal.Interfaces[ifIndex]
	if !exist {
		intf.conf.InterfaceId = ifIndex
		intf.conf.LocalMultiplier = protocol.DEFAULT_DETECT_MULTI
		intf.conf.DesiredMinTxInterval = protocol.DEFAULT_DESIRED_MIN_TX_INTERVAL
		intf.conf.RequiredMinRxInterval = protocol.DEFAULT_REQUIRED_MIN_RX_INTERVAL
		intf.conf.RequiredMinEchoRxInterval = protocol.DEFAULT_REQUIRED_MIN_ECHO_RX_INTERVAL
		intf.conf.DemandEnabled = false
		intf.conf.AuthenticationEnabled = false
		intf.conf.AuthenticationType = 0
		intf.conf.AuthenticationKeyId = 0
		intf.conf.SequenceNumber = 0
		intf.conf.AuthenticationData = ""
		intf.property.IfName = ipIntfProp.IfName
		intf.property.IpAddr = ipIntfProp.IpAddr
		intf.property.NetMask = ipIntfProp.NetMask
		intf.property.MacAddr = ipIntfProp.MacAddr
	} else {
		server.logger.Info(fmt.Sprintln("Intf Conf is not initialized ", ifIndex))
	}
}

func (server *BFDServer) createIPIntfConfMap(msg IPv4IntfNotifyMsg) {
	ip, ipNet, err := net.ParseCIDR(msg.IpAddr)
	if err != nil {
		server.logger.Err(fmt.Sprintln("Unable to parse IP address", msg.IpAddr))
		return
	}
	ifName, err := server.getLinuxIntfName(msg.IfId, msg.IfType)
	if err != nil {
		server.logger.Err("No Such Interface exists")
		return
	}
	server.logger.Info(fmt.Sprintln("create IPIntf for ", msg))

	macAddr, err := getMacAddrIntfName(ifName)
	if err != nil {
		server.logger.Err(fmt.Sprintln("Unable to get MacAddress of Interface exists", ifName))
		return
	}
	ipIntfProp := IpIntfProperty{
		IfName:  ifName,
		IpAddr:  ip,
		NetMask: ipNet.Mask,
		MacAddr: macAddr,
	}
	ifIndex := asicdConstDefs.GetIfIndexFromIntfIdAndIntfType(int(msg.IfId), int(msg.IfType))
	server.initDefaultIntfConf(ifIndex, ipIntfProp)
	_, exist := server.bfdGlobal.Interfaces[ifIndex]
	if !exist {
		server.logger.Err("No such inteface exists")
		return
	}
	if server.bfdGlobal.Enabled {
		server.StartSendRecvPkts(ifIndex)
	}
}

func (server *BFDServer) deleteIPIntfConfMap(msg IPv4IntfNotifyMsg) {
	server.logger.Info(fmt.Sprintln("delete IPIntfConfMap for ", msg))

	ifIndex := asicdConstDefs.GetIfIndexFromIntfIdAndIntfType(int(msg.IfId), int(msg.IfType))
	_, exist := server.bfdGlobal.Interfaces[ifIndex]
	if !exist {
		server.logger.Err("No such inteface exists")
		return
	}
	if server.bfdGlobal.Enabled {
		server.StopSendRecvPkts(ifIndex)
	}
	delete(server.bfdGlobal.Interfaces, ifIndex)
}

func (server *BFDServer) updateIPIntfConfMap(ifConf config.IntfConfig) {
	intf, exist := server.bfdGlobal.Interfaces[ifConf.InterfaceId]
	//  we can update only when we already have entry
	if exist {
		intf.conf.InterfaceId = ifConf.InterfaceId
		intf.conf.LocalMultiplier = ifConf.LocalMultiplier
		intf.conf.DesiredMinTxInterval = ifConf.DesiredMinTxInterval
		intf.conf.RequiredMinRxInterval = ifConf.RequiredMinRxInterval
		intf.conf.RequiredMinEchoRxInterval = ifConf.RequiredMinEchoRxInterval
		intf.conf.DemandEnabled = ifConf.DemandEnabled
		intf.conf.AuthenticationEnabled = ifConf.AuthenticationEnabled
		intf.conf.AuthenticationType = ifConf.AuthenticationType
		intf.conf.AuthenticationKeyId = ifConf.AuthenticationKeyId
		intf.conf.SequenceNumber = ifConf.SequenceNumber
		intf.conf.AuthenticationData = ifConf.AuthenticationData
	}
}

func (server *BFDServer) processIntfConfig(ifConf config.IntfConfig) {
	intf, exist := server.bfdGlobal.Interfaces[ifConf.InterfaceId]
	if !exist {
		server.logger.Err("No such L3 interface exists")
		return
	}
	if server.bfdGlobal.Enabled {
		server.StopSendRecvPkts(ifConf.InterfaceId)
	}

	server.updateIPIntfConfMap(ifConf)

	intf, _ = server.bfdGlobal.Interfaces[ifConf.InterfaceId]
	if server.bfdGlobal.Enabled {
		server.StartSendRecvPkts(intf.conf.InterfaceId)
	}
}

func (server *BFDServer) StopSendRecvPkts(ifIndex int32) {
}

func (server *BFDServer) StartSendRecvPkts(ifIndex int32) {
}