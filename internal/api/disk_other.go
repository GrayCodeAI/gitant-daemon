//go:build !linux && !darwin

package api

import "errors"

func getDiskUsage(path string) (float64, error) {
	return 0, errors.New("disk usage not supported on this platform")
}
