package requestsnative

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"math/rand/v2"
	"strings"
	"sync"

	// "crypto/x509"
	"fmt"
	"goodcheckgogo/checklist"
	"goodcheckgogo/options"
	"goodcheckgogo/strategy"
	"goodcheckgogo/utils"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

var (
	_dialer = &net.Dialer{
		KeepAlive:     -1,
		FallbackDelay: -1,
	}

	_tlsConfig = &tls.Config{
		//InsecureSkipVerify: options.MyOptions.SkipCertVerify.Value,
	}

	_transport = &http.Transport{
		DisableKeepAlives:   true,
		DisableCompression:  true,
		IdleConnTimeout:     1 * time.Second,
		TLSClientConfig:     _tlsConfig,
		MaxIdleConns:        -1,
		MaxIdleConnsPerHost: -1,
	}

	_quicConfig = &quic.Config{
		//MaxIncomingStreams:    int64(threads),
		//MaxIncomingUniStreams: int64(threads),
		KeepAlivePeriod: 0,
		MaxIdleTimeout:  2 * time.Second,
	}

	_transportH3 = &http3.Transport{
		TLSClientConfig: _tlsConfig,
		QUICConfig:      _quicConfig,
	}

	_client = &http.Client{
		//Timeout: time.Duration(options.MyOptions.ConnTimeout.Value) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			domain1parts := strings.Split(via[0].URL.Hostname(), ".")
			domain2parts := strings.Split(req.URL.Hostname(), ".")
			if domain1parts[len(domain1parts)-2] != domain2parts[len(domain2parts)-2] {
				log.Println("Suspicious redirection detected, treating as failure:", via[0].URL, "->", req.URL)
				return fmt.Errorf("bad redirection")
			} else {
				log.Println("Safe redirection detected:", via[0].URL, "->", req.URL)
				return http.ErrUseLastResponse
			}
		},
	}
)

var poolAlreadyReaded = false

func SetTransport(threads int, timeout int) {
	_quicConfig.MaxIncomingStreams = int64(threads)
	_quicConfig.MaxIncomingUniStreams = int64(threads)
	_client.Timeout = time.Duration(timeout) * time.Second
	_tlsConfig.InsecureSkipVerify = options.MyOptions.SkipCertVerify.Value
	if !options.MyOptions.SkipCertVerify.Value && !poolAlreadyReaded {
		_tlsConfig.InsecureSkipVerify = false
		pool, err := x509.SystemCertPool()
		if err != nil {
			log.Printf("Can't read system certificates pool: %v\n", err)
		} else {
			log.Printf("Certificates pool successfully readed")
			_tlsConfig.RootCAs = pool
		}
		poolAlreadyReaded = true
	} else {
		_tlsConfig.InsecureSkipVerify = true
	}
}

func CloseIdle() {
	_client.CloseIdleConnections()
}

func CheckConnectivityNative() error {

	_request, err := http.NewRequest("GET", options.MyOptions.NetConnTestURL.Value, nil)
	if err != nil {
		return fmt.Errorf("can't form request: %v", err)
	}

	// switch strategy.Protocol {
	// case "TCP":
	_transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return _dialer.DialContext(ctx, fmt.Sprintf("tcp%d", strategy.IPV), addr)
	}
	_client.Transport = _transport
	// case "UDP":
	// 	_transportH3.Dial = func(ctx context.Context, addr string, tlsConf *tls.Config, quicConf *quic.Config) (quic.EarlyConnection, error) {
	// 		udpAddr, err := net.Dial(strategy.ProtoFull, addr)
	// 		if err != nil {
	// 			log.Printf("error setting UDP transport dialer: %v", err)
	// 			return nil, err
	// 		}
	// 		log.Printf("Dialing %s...", udpAddr.RemoteAddr().String())
	// 		return quic.DialAddrEarly(ctx, udpAddr.RemoteAddr().String(), tlsConf, quicConf)
	// 	}
	// 	_client.Transport = _transportH3
	// }

	if !options.MyOptions.SkipCertVerify.Value {
		log.Printf("Making normal request to '%s' (Native)\n", options.MyOptions.NetConnTestURL.Value)
	} else {
		log.Printf("Making insecure request to '%s' (Native)\n", options.MyOptions.NetConnTestURL.Value)
	}

	_response, err := _client.Do(_request)
	if err != nil || _response == nil {
		return fmt.Errorf("can't get proper response: %v", err)
	}
	defer _response.Body.Close()

	if _response.StatusCode == 0 {
		return fmt.Errorf("can't properly verify connection: response code 0")
	} else {
		log.Printf("Connection seems ok; response code: %d %s\n", _response.StatusCode, http.StatusText(_response.StatusCode))
		return nil
	}
}

func ExtractClusterNative(mappingURL string) string {
	log.Printf("Attempting to extract cluster codename from '%s'...\n", mappingURL)

	resp, err := http.Get(mappingURL)
	if err != nil || resp.Body == nil {
		return ""
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil || len(b) == 0 {
		return ""
	}
	s := string(b)
	ss := strings.Split(s, " ")[2]

	return ss
}

// func SetProxy() {
// 	if strategy.Proxy != "noproxy" {
// 		log.Println("\nSetting up proxy for native mode...")
// 		s := strings.Split(strategy.Proxy, ":")[0]
// 		h := strings.Split(strategy.Proxy, "/")[1]
// 		var proxyUrl = &url.URL{
// 			Scheme: s,
// 			Host:   h,
// 		}
// 		_transport.Proxy = http.ProxyURL(proxyUrl)
// 		log.Println("Proxy scheme: %s\nProxy address: %s\n", s, h)
// 	}
// }

func extractDomain(url string) string {
	withReplaces := utils.InsensitiveReplace(url, "http://", "")
	withReplaces = utils.InsensitiveReplace(withReplaces, "https://", "")
	withReplaces = utils.InsensitiveReplace(withReplaces, "www.", "")

	return withReplaces
}

func SendRequest(wg *sync.WaitGroup, site *checklist.Website, sites []checklist.Website) {
	defer wg.Done()

	site.LastResponseCode = 0

	_request, err := http.NewRequest("GET", site.Address, nil)
	if err != nil {
		log.Printf("Problem with a request: %v\n", err)
		return
	}

	switch strategy.Protocol {
	case "UDP":
		_transportH3.Dial = func(ctx context.Context, addr string, tlsConf *tls.Config, quicConf *quic.Config) (quic.EarlyConnection, error) {
			if strategy.Proxy != "noproxy" {
				return quic.DialAddrEarly(ctx, addr, tlsConf, quicConf)
			}
			a := ""
			for i := 0; i < len(sites); i++ {
				if extractDomain(addr) == extractDomain(sites[i].Address)+":443" {
					if strategy.IPV == 6 {
						a = "[" + sites[i].IP + "]:443"
					} else {
						a = sites[i].IP + ":443"
					}
					break
				}
			}
			if a == "" {
				log.Panicf("Panic: can't assign IP to '%s'\n", addr)
			}
			return quic.DialAddrEarly(ctx, a, tlsConf, quicConf)
		}
		//defer _transportH3.Close()
		_client.Transport = _transportH3
	case "TCP":
		_transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			if strategy.Proxy != "noproxy" {
				return _dialer.DialContext(ctx, strategy.ProtoFull, addr)
			}
			a := ""
			for i := 0; i < len(sites); i++ {
				if extractDomain(addr) == extractDomain(sites[i].Address)+":443" {
					if strategy.IPV == 6 {
						a = "[" + sites[i].IP + "]:443"
					} else {
						a = sites[i].IP + ":443"
					}
					break
				}
			}
			if a == "" {
				log.Panicf("Panic: can't assign IP to '%s'\n", addr)
			}
			return _dialer.DialContext(ctx, strategy.ProtoFull, a)
		}
		_client.Transport = _transport
	}

	r := rand.IntN(options.MyOptions.InternalTimeoutMs.Value)
	time.Sleep(time.Duration(r) * time.Millisecond)
	_response, err := _client.Do(_request)
	if err != nil && utils.UnwrapErrCompletely(err).Error() == "invalid header field name: \"connection\"" {
		site.LastResponseCode = 418
		site.HasSuccesses = true
		return
	}
	if err != nil {
		site.LastResponseCode = 0
		//log.Println("err:", err.Error())
		return
	}
	defer _response.Body.Close()

	site.LastResponseCode = _response.StatusCode
	site.HasSuccesses = true
}
