package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

//---------------------------------------------------------

func main() {
	senders()
	listener()
	cleanPeers()
}

//---------------------------------------------------------

const serdisPort = 56765

var (
	peers      map[string]int = make(map[string]int)
	peersMutex sync.Mutex
)

func senders() {
	serdisMarker := []byte("serdis")
	for _, bip := range broadcastIPv4s() {
		addr := &net.UDPAddr{
			IP:   bip,
			Port: serdisPort,
		}
		udpSocket, err := net.DialUDP("udp4", nil, addr)
		if err != nil {
			continue
		}
		go func() {
			for {
				udpSocket.Write(serdisMarker)
				time.Sleep(5 * time.Second)
			}
		}()
	}
}

func listener() {
	socket, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: serdisPort,
	})
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		data := make([]byte, 256)
		for {
			_, addr, _ := socket.ReadFromUDP(data)
			peersMutex.Lock()
			peers[addr.IP.String()] = 60
			peersMutex.Unlock()
		}
	}()
}

func cleanPeers() {
	for {
		fmt.Println("--Peers--")
		peersMutex.Lock()
		for k, v := range peers {
			fmt.Println(k, "->", v)
			if v <= 0 {
				delete(peers, k)
			} else {
				peers[k] = v - 10
			}
		}
		peersMutex.Unlock()
		fmt.Println("---------")
		time.Sleep(10 * time.Second)
	}
}

//---------------------------------------------------------
// IPv4 Tools
//---------------------------------------------------------

func broadcastIPv4s() []net.IP {
	ips := []net.IP{}
	interfaces, _ := net.Interfaces()
	for _, intf := range interfaces {
		if (intf.Flags & net.FlagBroadcast) == 0 {
			continue
		}
		addrs, _ := intf.Addrs()
		for _, addr := range addrs {
			_, ipnet, _ := net.ParseCIDR(addr.String())
			bip := broadcastIPv4(ipnet)
			if len(bip) > 0 {
				ips = append(ips, bip)
			}
		}
	}
	return ips
}

func broadcastIPv4(n *net.IPNet) net.IP {
	if n.IP.To4() == nil {
		return net.IP{}
	}
	ip := make(net.IP, len(n.IP.To4()))
	a := binary.BigEndian.Uint32(n.IP.To4())
	b := binary.BigEndian.Uint32(net.IP(n.Mask).To4())
	binary.BigEndian.PutUint32(ip, a|^b)
	return ip
}

//---------------------------------------------------------
