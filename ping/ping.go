package ping

import (
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

//go:generate mockgen -package mocks -destination mocks/user.go github.com/pivotal-cf/pcfdev-cli/ping User
type User interface {
	IsPrivileged() (bool, error)
}

type Pinger struct {
	User User
}

func (p *Pinger) ICMPProtocol() (protocol string, err error) {
	privilegedUser, err := p.User.IsPrivileged()
	if err != nil {
		return "", fmt.Errorf("failed to determine user privileges: %s", err)
	}

	if privilegedUser {
		return "ip4:1", nil
	}

	return "udp4", nil
}

func (p *Pinger) ICMPAddr(ip string) (addr net.Addr, err error) {
	privilegedUser, err := p.User.IsPrivileged()
	if err != nil {
		return nil, fmt.Errorf("failed to determine user privileges: %s", err)
	}

	if privilegedUser {
		ipAddr := &net.IPAddr{
			IP: net.ParseIP(ip),
		}
		return ipAddr, nil
	}

	udpAddr := &net.UDPAddr{
		IP: net.ParseIP(ip),
	}
	return udpAddr, nil
}

func (p *Pinger) TryIP(ip string) (bool, error) {
	icmpProtocol, err := p.ICMPProtocol()
	if err != nil {
		return false, err
	}

	icmpAddr, err := p.ICMPAddr(ip)
	if err != nil {
		return false, err
	}

	pingConn, err := icmp.ListenPacket(icmpProtocol, "0.0.0.0")
	if err != nil {
		return false, fmt.Errorf("failed to open connection: %s", err)
	}

	defer pingConn.Close()

	message := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
		},
	}
	messageData, err := message.Marshal(nil)
	if err != nil {
		return false, fmt.Errorf("failed to marshal icmp message: %s", err)
	}

	_, err = pingConn.WriteTo(messageData, icmpAddr)
	if err != nil {
		return false, fmt.Errorf("failed to send icmp message: %s", err)
	}
	responseData := make([]byte, 1500)
	pingConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	len, _, err := pingConn.ReadFrom(responseData)
	if err != nil {
		return false, nil
	}

	response, err := icmp.ParseMessage(1, responseData[:len])
	if err != nil {
		return false, fmt.Errorf("badly formatted response: %s", err)
	}

	switch response.Type {
	case ipv4.ICMPTypeEchoReply:
		return true, nil
	default:
		return false, errors.New("ping response did not have type 'echo reply'")
	}
}