package virtualbox

import (
	"mikrodock-cli/parsers"
	"strings"
)

// GetPrivateAddress uses a little trick for the virtualbox provider : FE012 is the beginning of the MAC
// Returns a pointer (the main struct is allocated in the caller so it's ok) to allow nil value in case of no interfaces
func GetPrivateAddress(ifaces []parsers.NetInterface) *string {
	for _, iface := range ifaces {
		if strings.HasPrefix(iface.LinkAddress.Address, "fe:01:2") {
			// According to iproute2, the ipv4 address is always listed first
			return &iface.Addresses[0].Address
		}
	}
	return nil
}
