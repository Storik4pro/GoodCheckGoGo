package lookup

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil/sysresolv"
	"github.com/miekg/dns"
)

type DnsResponse struct {
	Response bool
	Zero     bool
	Answer   []DnsAnswer
}

type DnsAnswer struct {
	A    string
	AAAA string
}

func DnsLookup(_resolver string, _addrToResolve string, _ipv int, _timeout int, _retries int, _skipVerify bool) (DnsResponse, error) {
	var c DnsResponse

	o := &upstream.Options{
		Timeout:            time.Duration(_timeout) * time.Second,
		InsecureSkipVerify: _skipVerify,
		HTTPVersions:       []upstream.HTTPVersion{upstream.HTTPVersion2, upstream.HTTPVersion11},
	}

	if _resolver == "" {
		systemResolvers, err := sysresolv.NewSystemResolvers(nil, 53)
		if err != nil {
			return c, fmt.Errorf("can't get system resolvers: %v", err)
		}
		_resolver = systemResolvers.Addrs()[0].String()
	}

	u, err := upstream.AddressToUpstream(_resolver, o)
	if err != nil {
		return c, fmt.Errorf("can't create an upstream: %v", err)
	}
	defer u.Close()

	// rr, ok := dns.StringToType[_rrType]
	// if !ok {
	// 	return c, fmt.Errorf("invalid RRTYPE '%s'", _rrType)
	// }

	var q = dns.Question{
		Name:   dns.Fqdn(_addrToResolve),
		Qclass: dns.ClassINET,
	}
	q.Qtype = dns.TypeA
	if _ipv == 6 {
		q.Qtype = dns.TypeAAAA
	}

	req := &dns.Msg{}
	req.Id = dns.Id()
	req.RecursionDesired = true
	req.Question = []dns.Question{q}

	retries := _retries
	var resp *dns.Msg
	for retries >= 0 {
		resp, err = u.Exchange(req)
		if err == nil {
			u.Close()
			break
		} else {
			log.Printf("Can't resolve '%s' (attempts left %d): %v", _addrToResolve, retries, err)
			if retries == 0 {
				u.Close()
				log.Println("No attempts left")
				return c, fmt.Errorf("can't resolve")
			}
			retries--
		}
	}

	var b []byte
	b, err = json.Marshal(resp)
	if err != nil {
		return c, fmt.Errorf("can't marshal json: %v", err)
	}

	err = json.Unmarshal(b, &c)
	if err != nil {
		return c, fmt.Errorf("can't unmarshal json: %v", err)
	}

	return c, nil
}
