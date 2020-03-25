package main

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

func TestHandleDNS(t *testing.T) {
	for _, tt := range []struct {
		w dns.ResponseWriter
		r *dns.Msg
		n string
	}{
		{&fakeResponseWriter{}, &dns.Msg{Question: []dns.Question{
			dns.Question{Name: "10.10.10.10.xip.name.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
		}}, "10.10.10.10.xip.name."},
		{&fakeResponseWriter{}, &dns.Msg{Question: []dns.Question{
			dns.Question{Name: "xip.name.", Qtype: dns.TypeAAAA, Qclass: dns.ClassINET},
		}}, "xip.name."},
	} {
		handleDNS(tt.w, tt.r)

		f, ok := tt.w.(*fakeResponseWriter)
		if !ok {
			t.Fatalf("tt.w is not a *fakeResponseWriter")
		}

		if got, want := f.Msg.Answer[0].Header().Name, tt.n; got != want {
			t.Fatalf("f.Msg.Answer[0].Header().Name = %q, want %q", got, want)
		}
	}
}

func TestNewServer(t *testing.T) {
	for _, tt := range []struct {
		net  string
		addr string
	}{
		{"abc", ":53"},
		{"xyz", "127.0.0.2:5353"},
	} {
		s := newServer(tt.addr, tt.net)

		if got, want := s.Net, tt.net; got != want {
			t.Fatalf("s.Net = %q, want %q", got, want)
		}

		if got, want := s.Addr, tt.addr; got != want {
			t.Fatalf("s.Addr = %q, want %q", got, want)
		}
	}
}

func TestDnsRR(t *testing.T) {
	for _, tt := range []struct {
		name string
		want string
	}{
		{"abc", "abc\t300\tIN\tA\t"},
		{"xyz", "xyz\t300\tIN\tA\t"},
		{"nr1.10.0.0.1", "nr1.10.0.0.1\t300\tIN\tA\t10.0.0.1"},
		{"sub.10.0.0.1", "sub.10.0.0.1\t300\tIN\tA\t10.0.0.1"},
	} {
		rr := dnsRR(tt.name)

		if got, want := rr.String(), tt.want; got != want {
			t.Fatalf("rr.String() = %q, want %q", got, want)
		}
	}
}

type fakeResponseWriter struct {
	Msg *dns.Msg
}

func (f *fakeResponseWriter) LocalAddr() net.Addr {
	panic("not implemented LocalAddr")
}

func (f *fakeResponseWriter) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5678, Zone: ""}
}

func (f *fakeResponseWriter) WriteMsg(msg *dns.Msg) error {
	f.Msg = msg
	return nil
}

func (f *fakeResponseWriter) Write([]byte) (int, error) {
	panic("Write not implemented")
}

func (f *fakeResponseWriter) Close() error {
	panic("Close not implemented")
}

func (f *fakeResponseWriter) TsigStatus() error {
	panic("TsigStatus not implemented")
}

func (f *fakeResponseWriter) TsigTimersOnly(bool) {
	panic("TsigTimersOnly not implemented")
}

func (f *fakeResponseWriter) Hijack() {
	panic("Hijack not implemented")
}

type unknownAddr struct{}

func (unknownAddr) Network() string {
	return ""
}

func (unknownAddr) String() string {
	return ""
}
