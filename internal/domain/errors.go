package domain

import "errors"

func IsNotFound(err error) bool {
	return errors.Is(err, ErrFileNotFound)
}

func IsDependencyMissing(err error) bool {
	return errors.Is(err, ErrDependencyMissing)
}

func IsDeviceUnavailable(err error) bool {
	return errors.Is(err, ErrNoDevice) || errors.Is(err, ErrDeviceLocked)
}
