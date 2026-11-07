//go:build !windows && !js

package main

import "errors"

func installUpdate(src, dst, work string) error {
	return errors.New("updating in place is only available on Windows")
}
