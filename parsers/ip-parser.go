package parsers

import (
	"regexp"
	"strconv"
	"strings"
)

type AddressType int

const (
	V4 AddressType = iota
	V6
)

type LinkAddress struct {
	Address   string
	Broadcast string
}

type InterfaceAddress struct {
	AddressType AddressType
	Address     string
	CIDR        int
	Broadcast   string
}

type LinkInfo struct {
	Key   string
	Value string
}

type NetInterface struct {
	InterfaceIndex int
	InterfaceName  string
	LinkFlags      []string
	LinkInfos      []LinkInfo
	LinkType       string
	LinkAddress    LinkAddress
	Addresses      []InterfaceAddress
}

func ParseIPAddrShow(out string) []NetInterface {
	splitPerInterface := regexp.MustCompile(`(?m)^\S`)
	res := splitPerInterface.FindAllStringIndex(out, -1)

	startIndex := make([]int, len(res))
	for i, v := range res {
		startIndex[i] = v[0]
	}

	ifaces := make([]NetInterface, len(startIndex)-1)

	for s := 1; s < len(startIndex); s++ {
		start := startIndex[s-1]
		end := startIndex[s]
		slice := out[start : end-1] // -1 to remove line break
		netif := parseInterface(slice)
		ifaces[s-1] = netif
	}
	return ifaces

}

func parseInterface(ifDescr string) NetInterface {

	regexHeader := regexp.MustCompile(`^(\d)+: ((?:[\w@])+): <((?:[\w-]+,?)+)> ((?:\w+\s\w+\s?)+)`)
	regexLink := regexp.MustCompile(`\s{4}link/(\w+) ((?:[[:xdigit:]]{2}:?){6}) brd ((?:[[:xdigit:]]{2}:?){6})`)
	regexIPV4 := regexp.MustCompile(`\s{4}inet ((?:\d+\.?){4})/(\d+) (?:brd ((?:\d+\.?){4}))?`)
	regexIPV6 := regexp.MustCompile(`\s{4}inet6 ([0-9a-f:]+)/(\d+)`)

	lines := strings.Split(ifDescr, "\n")
	header := lines[0]
	details := lines[1:]
	headerParsed := regexHeader.FindStringSubmatch(header)
	interfaceNumber := headerParsed[1]
	interfaceName := headerParsed[2]
	flags := strings.Split(headerParsed[3], ",")
	infos := strings.Split(headerParsed[4], " ")
	infosTyped := make([]LinkInfo, len(infos)/2)
	for i := 0; i < len(infos)-1; i += 2 {
		info := LinkInfo{infos[i], infos[i+1]}
		infosTyped[i/2] = info
	}
	ifn, _ := strconv.Atoi(interfaceNumber)
	netif := NetInterface{InterfaceIndex: ifn, InterfaceName: interfaceName, LinkFlags: flags, LinkInfos: infosTyped}
	addresses := make([]InterfaceAddress, 0)
	for _, line := range details {
		switch {
		case strings.HasPrefix(line, "    link/"):
			link := regexLink.FindStringSubmatch(line)
			netif.LinkType = link[1]
			netif.LinkAddress.Address = link[2]
			netif.LinkAddress.Broadcast = link[3]
		case strings.HasPrefix(line, "    inet "):
			ipv4 := regexIPV4.FindStringSubmatch(line)
			cidr, _ := strconv.Atoi(ipv4[2])
			add := InterfaceAddress{V4, ipv4[1], cidr, ipv4[3]}
			addresses = append(addresses, add)
		case strings.HasPrefix(line, "    inet6 "):
			ipv6 := regexIPV6.FindStringSubmatch(line)
			cidr, _ := strconv.Atoi(ipv6[2])
			add := InterfaceAddress{V6, ipv6[1], cidr, ""}
			addresses = append(addresses, add)
		}
	}
	netif.Addresses = addresses
	return netif
}
