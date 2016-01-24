package server

import (
	"net"
	"time"
)

var ALLSPFROUTER string = "224.0.0.5"
var ALLDROUTER string = "224.0.0.6"

type OspfHdrMetadata struct {
	pktType  OspfType
	pktlen   uint16
	backbone bool
	routerId []byte
}

func NewOspfHdrMetadata() *OspfHdrMetadata {
	return &OspfHdrMetadata{}
}

type DstIPType uint8

const (
	Normal       DstIPType = 1
	AllSPFRouter DstIPType = 2
	AllDRouter   DstIPType = 3
)

type IpHdrMetadata struct {
	srcIP     []byte
	dstIP     []byte
	dstIPType DstIPType
}

func NewIpHdrMetadata() *IpHdrMetadata {
	return &IpHdrMetadata{}
}

var (
	snapshot_len int32         = 65549 //packet capture length
	promiscuous  bool          = false //mode
	timeout_pcap time.Duration = 5 * time.Second
)

const (
	OSPF_HELLO_MIN_SIZE = 20
	OSPF_HEADER_SIZE    = 24
	IP_HEADER_MIN_LEN   = 20
	OSPF_PROTO_ID       = 89
	OSPF_VERSION_2      = 2
)

type OspfType uint8

const (
	HelloType         OspfType = 1
	DBDescriptionType OspfType = 2
	LSRequestType     OspfType = 3
	LSUpdateType      OspfType = 4
	LSAckType         OspfType = 5
)

type IntfToNeighMsg struct {
	IntfConfKey  IntfConfKey
	RouterId     uint32
	RtrPrio      uint8
	NeighborIP   net.IP
	nbrDeadTimer time.Duration
	TwoWayStatus bool
}

type NbrStateChangeMsg struct {
	RouterId uint32
}

const (
	EOption  = 0x02
	MCOption = 0x04
	NPOption = 0x08
	EAOption = 0x20
	DCOption = 0x40
)
