package testconn

import (
	"net"
	"time"

	"github.com/lightningnetwork/lnd/tor"
)

func WrapDial(dial tor.DialFunc, brokenNetwork func() bool) tor.DialFunc {
	return func(net, addr string, timeout time.Duration) (net.Conn, error) {
		if brokenNetwork() {
			return nil, ErrBrokenNetwork
		}

		conn, err := dial(net, addr, timeout)
		if err != nil {
			return nil, err
		}

		return New(conn, brokenNetwork), nil
	}
}

func PostAccept(conn net.Conn, err error,
	brokenNetwork func() bool) (net.Conn, error) {

	if err != nil {
		return nil, err
	}

	if brokenNetwork() {
		err = conn.Close()
		if err == nil {
			err = ErrBrokenNetwork
		}
		return nil, err
	}

	return New(conn, brokenNetwork), nil
}
