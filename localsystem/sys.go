package localsystem

import (
	"syscall"

	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/nerr"
)

// Reboot reboots the device.
func Reboot() *nerr.E {
	log.L.Infof("*!!* REBOOTING DEVICE NOW *!!*")

	err := syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
	if err != nil {
		return nerr.Translate(err).Addf("failed to reboot device")
	}

	return nil
}