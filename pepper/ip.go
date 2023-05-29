package main

import (
	"fmt"
	"net"
	"strings"
)

func GetAvailableIP(baseIPHostName string, baseIP string) string {
	ip := baseIP
	hostName := baseIPHostName

	for {
		if val, ok := usedIps[hostName]; ok {
			ip, _ = GetNextIP(val)
			hostName = strings.ReplaceAll(ip, ".", "")
		} else {
			break
		}
	}

	return ip
}

func ipv4ToHex(ipString string) string {
	ip := net.ParseIP(ipString)
	ip = ip.To4()

	hexIP := make([]string, 4)
	for i, octet := range ip {
		hexIP[i] = fmt.Sprintf("%02x", octet)
	}

	return strings.Join(hexIP, ":")
}

func GetNextIP(ipString string) (string, error) {
	ip := net.ParseIP(ipString)
	if ip == nil {
		return "", fmt.Errorf("invalid IP address: %s", ipString)
	}

	ip = ip.To4()
	if ip == nil {
		return "", fmt.Errorf("IPv6 address is not supported")
	}

	nextIP := make(net.IP, len(ip))
	copy(nextIP, ip)

	for i := len(ip) - 1; i >= 0; i-- {
		nextIP[i]++
		if nextIP[i] > 0 {
			break
		}
	}

	return nextIP.String(), nil
}
