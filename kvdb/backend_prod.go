//go:build !dev && !js

package kvdb

import "errors"

func GetTestBackend(path, name string) (Backend, func(), error) {
	return nil, func() {}, errors.New(
		"kvdb test backend not available in production builds",
	)
}
