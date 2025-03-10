package main

import (
	"context"
	"fmt"
	"libcore"
	"libcore/device"
	"neko/pkg/neko_common"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	v2rayNet "github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/features/dns"
	"github.com/v2fly/v2ray-core/v5/features/dns/localdns"
)

var instance *libcore.V2RayInstance
var getNekorayTunIndex = func() int { return 0 } // Windows only
var underlyingNetDialer *net.Dialer              // NKR_VPN_LEGACY_DNS only

func setupCore() {
	// TODO del
	device.IsNekoray = true
	libcore.SetConfig("", false, true)
	libcore.InitCore("", "", "", nil, ".", "moe.nekoray.pc:bg", true, 50)
	// localdns setup
	resolver_def := &net.Resolver{PreferGo: false}
	resolver_go := &net.Resolver{PreferGo: true}
	if underlyingNetDialer != nil && os.Getenv("NKR_VPN_LEGACY_DNS") == "1" {
		resolver_def.Dial = underlyingNetDialer.DialContext
		resolver_go.Dial = underlyingNetDialer.DialContext
		logrus.Println("using NKR_VPN_LEGACY_DNS")
	}
	localdns.SetLookupFunc(func(network string, host string) (ips []net.IP, err error) {
		// fix old sekai
		defer func() {
			if len(ips) == 0 {
				logrus.Println("LookupIP error:", network, host, err)
				err = dns.ErrEmptyResponse
			}
		}()
		// Normal mode use system resolver (go bug)
		if getNekorayTunIndex() == 0 {
			return resolver_def.LookupIP(context.Background(), network, host)
		}
		// Windows VPN mode use Go resolver
		return resolver_go.LookupIP(context.Background(), network, host)
	})
	//
	neko_common.GetProxyHttpClient = func() *http.Client {
		return getProxyHttpClient(instance)
	}
}

// PROXY

func getProxyHttpClient(_instance *libcore.V2RayInstance) *http.Client {
	dailContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		dest, err := v2rayNet.ParseDestination(fmt.Sprintf("%s:%s", network, addr))
		if err != nil {
			return nil, err
		}
		return _instance.DialContext(ctx, dest)
	}

	transport := &http.Transport{
		TLSHandshakeTimeout:   time.Second * 3,
		ResponseHeaderTimeout: time.Second * 3,
	}

	if _instance != nil {
		transport.DialContext = dailContext
	}

	client := &http.Client{
		Transport: transport,
	}

	return client
}
