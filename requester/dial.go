package requester

import (
	"context"
	"crypto/tls"
	"net"
	"time"
)

var (
	dialer = &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}
)

func dialContext(ctx context.Context, network, address string) (conn net.Conn, err error) {
	switch network {
	case "tcp", "tcp4", "tcp6":
		// 检测缓存
		if TCPAddrCache.Existed(address) {
			return net.DialTCP(network, nil, TCPAddrCache.Get(address))
		}

		// Resolve TCP address
		ta, err := net.ResolveTCPAddr(network, address)
		if err != nil {
			return nil, err
		}

		// 加入缓存
		TCPAddrCache.Set(address, ta)
		return net.DialTCP(network, nil, ta)
	}

	// 非 tcp 请求
	conn, err = dialer.DialContext(ctx, network, address)
	return
}

func dial(network, address string) (conn net.Conn, err error) {
	return dialContext(context.Background(), network, address)
}

func dialTLS(network, address string) (tlsConn net.Conn, err error) {
	conn, err := dial(network, address)
	if err != nil {
		return nil, err
	}

	return tls.Client(conn, TLSConfig), nil
}
