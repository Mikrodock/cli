package iptables

import "strconv"

type IPTableAction int

const (
	APPEND IPTableAction = iota
	DELETE
)

// iptables -t nat -A POSTROUTING -s 172.18.0.5/32 -d 172.18.0.5/32 -p tcp -m tcp --dport 8080 -j MASQUERADE
// iptables -t nat -A DOCKER -p tcp -m tcp --dport 8080 -j DNAT --to-destination 172.18.0.5:8080
// iptables -A DOCKER -d 172.18.0.5/32 ! -i docker_gwbridge -o docker_gwbridge -p tcp -m tcp --dport 8080 -j ACCEPT

// TODO: Differ internal port and external port
func NewLinkPort(proxyIP string, destinationPort int) []string {
	rules := make([]string, 3)
	rules[0] = createRule("nat", "POSTROUTING", "", APPEND, "tcp", proxyIP+"/32", proxyIP+"/32", 8080, "", "MASQUERADE")
	rules[1] = createRule("nat", "DOCKER", "", APPEND, "tcp", "", proxyIP+":"+strconv.Itoa(destinationPort), 8080, "", "DNAT")
	rules[2] = createRule("", "DOCKER", "docker_gwbridge", APPEND, "tcp", "", proxyIP+"/32", 8080, "", "ACCEPT")
}

func createRule(tableName, chainName, mustOutputInterface string, action IPTableAction, protocol, source, destination string, destinationPort int, natDestination, jump string) string {
	rule := "iptables "
	if tableName != "" {
		rule += "-t " + tableName + " "
	}
	switch action {
	case APPEND:
		rule += "-A "
		break
	case DELETE:
		rule += "-D "
		break
	}

	if mustOutputInterface != "" {
		rule += "! -i " + mustOutputInterface + " -o " + mustOutputInterface + " "
	}

	rule += chainName + " -m " + protocol + " -p " + protocol + " "

	if source != "" {
		rule += "-s " + source + " "
	}

	if destination != "" {
		rule += "-d " + destination + " "
	}

	rule += "--dport " + strconv.Itoa(destinationPort) + " "

	rule += "-j " + jump + " "

	if jump == "DNAT" {
		rule += "--to-destination " + natDestination
	}

	return rule
}
