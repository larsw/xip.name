// Copyright 2014-2016 Peter Hellberg.
// Released under the terms of the MIT license.

// xip.name is a small name server which sends back any IP address found in the provided hostname.
//
// When queried for type A, it sends back the parsed IPv4 address.
// In the additional section the client host:port and transport are shown.
//
// Basic use pattern:
//
//    dig @xip.name foo.10.0.0.82.xip.name A
//
//    ; <<>> DiG 9.8.3-P1 <<>> @xip.name foo.10.0.0.82.xip.name A
//    ; (1 server found)
//    ;; global options: +cmd
//    ;; Got answer:
//    ;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 13574
//    ;; flags: qr rd; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 1
//    ;; WARNING: recursion requested but not available
//
//    ;; QUESTION SECTION:
//    ;foo.10.0.0.82.xip.name.		IN	A
//
//    ;; ANSWER SECTION:
//    foo.10.0.0.82.xip.name.	0	IN	A	10.0.0.82
//
//    ;; ADDITIONAL SECTION:
//    xip.name.		0	IN	TXT	"Client: 188.126.74.76:52575 (udp)"
//
//    ;; Query time: 27 msec
//    ;; SERVER: 188.166.43.179#53(188.166.43.179)
//    ;; WHEN: Wed Dec 31 02:55:51 2014
//    ;; MSG SIZE  rcvd: 128
//
// Initially based on the reflect example found at https://github.com/miekg/exdns
//
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/kelseyhightower/envconfig"
	"github.com/miekg/dns"
)

// IPDecoder xxx
type IPDecoder net.IP

// Decode xxx
func (ipd *IPDecoder) Decode(value string) error {
	*ipd = IPDecoder(net.ParseIP(value))
	return nil
}

// To4 ...
func (ipd *IPDecoder) To4() net.IP {
	return net.IP(*ipd).To4()
}

// Config xxx
type Config struct {
	Verbose bool      `default:"false"`
	Fqdn    string    `default:"xip.name."`
	Addr    string    `default:":53"`
	IP      IPDecoder `default:"127.0.0.1"`
}

var (
	config    Config
	ipPattern = regexp.MustCompile(`(\b\d{1,3}[\.-]\d{1,3}[\.-]\d{1,3}[\.-]\d{1,3})`)
	defaultIP net.IP
)

func main() {
	fmt.Printf("%s starting up...\n", os.Args[0])

	err := envconfig.Process("XIP", &config)

	if err != nil {
		log.Fatal(err.Error())
	}

	verboseFlag := flag.Bool("verbose", false, "Verbose")
	fqdnFlag := flag.String("fqdn", "xip.name.", "FQDN to handle")
	addrFlag := flag.String("addr", ":53", "The addr to bind on")
	ipFlag := flag.String("ip", "127.0.0.1", "The IP of xip.name")

	flag.Parse()

	if *verboseFlag == true {
		config.Verbose = true
	}

	if *fqdnFlag != "xip.name." {
		config.Fqdn = *fqdnFlag
	}

	if *addrFlag != ":53" {
		config.Addr = *addrFlag
	}

	if *ipFlag != "127.0.0.1" {
		config.IP.Decode(*ipFlag)
	}

	if config.Verbose {
		fmt.Printf("Verbose: %v\n", config.Verbose)
		fmt.Printf("FQDN:    %s\n", config.Fqdn)
		fmt.Printf("Address: %s\n", config.Addr)
		fmt.Printf("IP:      %s\n", config.IP.To4())
	}

	defaultIP = config.IP.To4()

	// Ensure that a FQDN is passed in (often the trailing . is omitted)
	config.Fqdn = dns.Fqdn(config.Fqdn)

	dns.HandleFunc(config.Fqdn, handleDNS)

	go serve(config.Addr, "tcp")
	go serve(config.Addr, "udp")

	fmt.Println("Ready to receive requests, CTRL-C to shutdown.")

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	fmt.Printf("Signal (%v) received, stopping\n", s)
}

func handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	m := &dns.Msg{}
	m.SetReply(r)

	if len(r.Question) == 0 {
		return
	}

	q := r.Question[0]
	//	t := dnsTXT(clientString(w.RemoteAddr()))
	rr := dnsRR(q.Name)

	soa := &dns.SOA{
		Hdr: dns.RR_Header{
			Name:   config.Fqdn,
			Rrtype: dns.TypeSOA,
			Class:  dns.ClassINET,
			Ttl:    1440,
		},
		Ns:      config.Fqdn,
		Serial:  2014123101,
		Mbox:    config.Fqdn,
		Refresh: 21600,
		Retry:   7200,
		Expire:  604800,
		Minttl:  3600,
	}

	switch q.Qtype {
	// case dns.TypeTXT:
	// 	//m.Answer = append(m.Answer, t)
	// 	m.Extra = append(m.Extra, rr)
	default:
		fallthrough
	// Start of Authority
	case dns.TypeSOA:
		m.Answer = append(m.Answer, soa)

	// A, AAAA & CNAME questions
	case dns.TypeAAAA, dns.TypeA, dns.TypeCNAME:
		m.Answer = append(m.Answer, rr)
		//m.Extra = append(m.Extra, t)

	// Transfer questions
	case dns.TypeAXFR, dns.TypeIXFR:
		c := make(chan *dns.Envelope)
		tr := new(dns.Transfer)
		defer close(c)

		err := tr.Out(w, r, c)
		if err != nil {
			if config.Verbose {
				fmt.Printf("%v\n", err)
			}

			return
		}

		c <- &dns.Envelope{RR: []dns.RR{soa, rr, soa}}
		w.Hijack()

		return
	}

	if config.Verbose {
		fmt.Printf("%v\n", m.String())
	}

	w.WriteMsg(m)
}

func serve(addr, net string) {
	if err := newServer(addr, net).ListenAndServe(); err != nil {
		fmt.Printf("Failed to setup the %q server: %s\n", net, err.Error())
	} else {
		fmt.Printf("Listening on %q/%s\n", addr, net)
	}
}

func newServer(addr, net string) *dns.Server {
	return &dns.Server{
		Addr:       addr,
		Net:        net,
		TsigSecret: nil,
	}
}

func dnsSOA(name string, ns string) (soa *dns.SOA) {
	return &dns.SOA{
		Hdr: dns.RR_Header{
			Name:   name,
			Rrtype: dns.TypeSOA,
			Class:  dns.ClassINET,
			Ttl:    300,
		},
		Ns: ns}
}

func dnsRR(name string) (rr *dns.A) {
	rr = &dns.A{
		Hdr: dns.RR_Header{
			Name:   name,
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    300,
		},
		A: defaultIP,
	}

	if ipStr := ipPattern.FindString(name); ipStr != "" {
		ipStr = strings.ReplaceAll(ipStr, "-", ".")
		rr.A = net.ParseIP(ipStr).To4()
	}
	return rr
}
