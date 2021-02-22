package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	quic "github.com/libp2p/go-libp2p-quic-transport"
	"github.com/libp2p/go-tcp-transport"
	ma "github.com/multiformats/go-multiaddr"
)

func main() {
	log.SetLogLevel("p2p-holepunch", "INFO")

	relayId, err := peer.Decode("Qma71QQyJN7Sw7gz1cgJ4C66ubHmvKqBasSegKRugM5qo6")
	if err != nil {
		panic(err)
	}
	relayInfo := []peer.AddrInfo{
		{
			ID:    relayId,
			Addrs: []ma.Multiaddr{ma.StringCast("/ip4/54.255.209.104/tcp/12001"), ma.StringCast("/ip4/54.255.209.104/udp/12001/quic")},
		},
	}

	ctx := context.Background()
	h, err := libp2p.New(ctx, libp2p.ForceReachabilityPrivate(), libp2p.EnableAutoRelay(),
		libp2p.StaticRelays(relayInfo), libp2p.EnableHolePunching(),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport),
		libp2p.ListenAddrs(ma.StringCast("/ip4/0.0.0.0/tcp/0"), ma.StringCast("/ip4/0.0.0.0/udp/0/quic")),
	)

	if err != nil {
		panic(err)
	}
	sub, err := h.EventBus().Subscribe(new(event.EvtNATDeviceTypeChanged))
	if err != nil {
		panic(err)
	}

	// bootstrap with dht so we can connect to more peers and discover our own addresses.
	d, err := dht.New(ctx, h, dht.Mode(dht.ModeClient), dht.BootstrapPeers(dht.GetDefaultBootstrapPeerAddrInfos()...))
	if err != nil {
		panic(err)
	}
	d.Bootstrap(ctx)

	// wait till we have a relay addrs
LOOP:
	for {
		time.Sleep(5 * time.Second)
		addrs := h.Addrs()
		for _, a := range addrs {
			if _, err := a.ValueForProtocol(ma.P_CIRCUIT); err == nil {
				break LOOP
			}
		}
	}

	// get NAT types for TCP & UDP
	for i := 0; i < 2; i++ {
		select {
		case ev := <-sub.Out():
			evt := ev.(event.EvtNATDeviceTypeChanged)
			if evt.NatDeviceType == network.NATDeviceTypeCone {
				fmt.Printf("\n your NAT device supports NAT traversal via hole punching for %s connections", evt.TransportProtocol)
			} else {
				fmt.Printf("\n your NAT device does NOT support NAT traversal via hole punching for %s connections", evt.TransportProtocol)
				return
			}

		case <-time.After(60 * time.Second):
			panic(errors.New("could not find NAT type"))
		}
	}

	fmt.Println("\n server peer id is: ", h.ID().Pretty())
	fmt.Println("-----------------------------------------------------------------------------------------------------------------------------------")
	fmt.Println("server addrs are:")
	for _, a := range h.Addrs() {
		fmt.Println(a)
	}
	fmt.Println("-----------------------------------------------------------------------------------------------------------------------------------")

	fmt.Println("\n------------------------------------------------------------------------------------------------------------------------------------")
	fmt.Println("accepting connections now")

	// block
	for {

	}
}
