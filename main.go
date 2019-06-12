package main

import (
	"flag"
	"fmt"
	"net"
	"sync"

	"github.com/golang/glog"
	"golang.org/x/net/dns/dnsmessage"
)

type clients struct {
	sync.RWMutex
	list map[string][]net.UDPAddr
}

func (c *clients) set(key string, addr net.UDPAddr) {
	c.Lock()
	if _, ok := c.list[key]; ok {
		c.list[key] = append(c.list[key], addr)
	} else {
		c.list[key] = []net.UDPAddr{addr}
	}
	c.Unlock()
}

func (c *clients) get(key string) ([]net.UDPAddr, bool) {
	c.RLock()
	retval, ok := c.list[key]
	c.RUnlock()
	return retval, ok
}

func (c *clients) remove(key string) bool {
	c.Lock()
	delete(c.list, key)
	c.Unlock()
	return true
}

var defaultIP [4]byte
var externalDNS net.UDPAddr
var clientList clients

func main() {
	flag.Parse()
	flag.Lookup("logtostderr").Value.Set("true")
	// Initialize Client List
	clientList = clients{
		list: make(map[string][]net.UDPAddr),
	}
	// Set this to the IP address of the portal web server
	// All unauthenticated users will be redirected here
	defaultIP = [4]byte{192, 168, 254, 254}
	// Set External DNS
	externalDNS = net.UDPAddr{
		Port: 53,
		IP:   net.ParseIP("1.1.1.1"),
	}

	// Listen on UDP 53
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 53})
	if err != nil {
		glog.Fatalln(err)
	}

	// Infinte loop to process dns queries to port 53
	for {
		tmp := make([]byte, 1024)
		_, clientaddr, err := conn.ReadFromUDP(tmp)
		if err != nil {
			glog.Fatalln(err)
			continue
		}
		glog.Infoln("Client :", clientaddr.String())
		var msg dnsmessage.Message
		// try to unpack dns message
		err = msg.Unpack(tmp)
		if err != nil {
			glog.Errorln(err)
			continue
		}
		// if there are no dns questions in the message
		if len(msg.Questions) == 0 {
			continue
		}

		// if message is from external dns
		if msg.Header.Response {
			key := fmt.Sprint(msg.ID)
			if addrs, ok := clientList.get(key); ok {
				for _, addr := range addrs {
					go sendMessage(conn, msg, addr)
				}
				clientList.remove(key)
			}
			continue
		}
		// TODO: Check if client IP address is authenticated by checking its corresponding MAC address
		// TODO: Get MAC Address based on IP Address
		// TODO: Check MAC Address if authenticated in DB
		isAuthenticated := false
		if isAuthenticated {
			// add client request to list
			clientList.set(fmt.Sprint(msg.ID), *clientaddr)
			// authenticated, forward message to external dns
			sendMessage(conn, msg, externalDNS)
		} else {
			// not authenticated reply default ip
			dnsAnswer := []dnsmessage.Resource{
				{
					Header: dnsmessage.ResourceHeader{
						Name:  msg.Questions[0].Name,
						Type:  dnsmessage.TypeA,
						Class: dnsmessage.ClassINET,
					},
					Body: &dnsmessage.AResource{A: defaultIP},
				},
			}
			msg.Answers = append(msg.Answers, dnsAnswer...)
			sendMessage(conn, msg, *clientaddr)
		}
	}
}

func sendMessage(conn *net.UDPConn, msg dnsmessage.Message, addr net.UDPAddr) {
	packed, err := msg.Pack()
	if err != nil {
		glog.Errorln(err)
	}

	_, err = conn.WriteToUDP(packed, &addr)
	if err != nil {
		glog.Errorln(err)
	}
}
