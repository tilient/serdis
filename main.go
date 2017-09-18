package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

//---------------------------------------------------------

const serdisPort = 56765

//---------------------------------------------------------

func main() {
	self := makeNode()
	self.broadcast()
	self.listen()
	self.run()
}

//---------------------------------------------------------

type (
	Link struct {
		ip net.IP
	}
	LinkList []*Link
	Node     struct {
		id         string
		name       string
		links      LinkList
		peers      map[string]int
		peersMutex sync.Mutex
	}
	NodeList []*Node
)

func makeNode() *Node {
	node := Node{
		id:    "1234567890",
		name:  "wiffel",
		links: LinkList{},
		peers: make(map[string]int),
	}
	return &node
}

//---------------------------------------------------------

func (n *Node) broadcast() {
	info := []byte("serdis\n" + n.id + "\n" + n.name + "\n")
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
				udpSocket.Write(info)
				time.Sleep(5 * time.Second)
			}
		}()
	}
}

func (n *Node) listen() {
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
			nr, addr, _ := socket.ReadFromUDP(data)
			//fmt.Printf("-recv-\n%s------\n", string(data[:nr]))
			parts := strings.Split(string(data[:nr]), "\n")
			//fmt.Printf("parts: %s\n", parts)
			id := parts[1]
			name := parts[2]
			ip := addr.IP
			n.peersMutex.Lock()
			n.peers[ip.String()] = 60
			n.peersMutex.Unlock()
			n.receivedPing(id, name, ip)
		}
	}()
}

func (lst LinkList) pingIP(pingIP net.IP) LinkList {
	for _, link := range lst {
		//fmt.Printf("== %s ?= %s ==\n", link.ip, pingIP)
		if link.ip.Equal(pingIP) {
			return lst
		}
	}
	return append(lst, &Link{ip: pingIP})
}

func (n *Node) receivedPing(id, name string, ip net.IP) {
	if id == n.id {
		//fmt.Println("** received from myself **")
		n.links = n.links.pingIP(ip)
	} else {
		fmt.Println("** received from PEER **")
	}
}

func (n *Node) run() {
	for {
		fmt.Println("--Peers--")
		n.peersMutex.Lock()
		for k, v := range n.peers {
			fmt.Println(k, "->", v)
			if v <= 0 {
				delete(n.peers, k)
			} else {
				n.peers[k] = v - 10
			}
		}
		n.peersMutex.Unlock()
		fmt.Println("---------")
		fmt.Printf("-- %s (%s) --\n", n.name, n.id)
		for _, l := range n.links {
			fmt.Printf("-- %s \n", l.ip)
		}
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
