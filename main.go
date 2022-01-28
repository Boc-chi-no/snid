package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"src.agwa.name/go-listener"
)

func main() {
	var flags struct {
		listen          []string
		defaultHostname string
		backendType     string
		proxyProto      bool
		unixDirectory   string
		backendCidr     []*net.IPNet
		backendPort     int
		nat46Prefix     net.IP
	}
	flag.Func("listen", "Socket to listen on (repeatable)", func(arg string) error {
		flags.listen = append(flags.listen, arg)
		return nil
	})
	flag.StringVar(&flags.defaultHostname, "default-hostname", "", "Default hostname if client does not provide SNI")
	flag.StringVar(&flags.backendType, "backend-type", "", "unix, tcp, or nat46")
	flag.BoolVar(&flags.proxyProto, "proxy-proto", false, "Use PROXY protocol when talking to backend")
	flag.StringVar(&flags.unixDirectory, "unix-directory", "", "Path to directory containing backend UNIX sockets (unix backends)")
	flag.Func("backend-cidr", "CIDR of allowed backends (repeatable) (tcp, nat46 backends)", func(arg string) error {
		_, ipnet, err := net.ParseCIDR(arg)
		if err != nil {
			return err
		}
		flags.backendCidr = append(flags.backendCidr, ipnet)
		return nil
	})
	flag.IntVar(&flags.backendPort, "backend-port", 0, "Port number of backend (tcp, nat46 backends)")
	flag.Func("nat46-prefix", "IPv6 prefix for NAT64 source address (nat46 backends)", func(arg string) error {
		flags.nat46Prefix = net.ParseIP(arg)
		if flags.nat46Prefix == nil {
			return fmt.Errorf("not a valid IP address")
		}
		if flags.nat46Prefix.To4() != nil {
			return fmt.Errorf("not an IPv6 address")
		}
		return nil
	})
	flag.Parse()

	server := &Server{
		ProxyProtocol:   flags.proxyProto,
		DefaultHostname: flags.defaultHostname,
	}

	switch flags.backendType {
	case "unix":
		if flags.unixDirectory == "" {
			log.Fatal("-unix-directory must be specified when you use -backend unix")
		}
		server.Backend = &UnixDialer{Directory: flags.unixDirectory}
	case "tcp":
		if len(flags.backendCidr) == 0 {
			log.Fatal("At least one -backend-cidr flag must be specified when you use -backend tcp")
		}
		server.Backend = &TCPDialer{Port: flags.backendPort, Allowed: flags.backendCidr}
	case "nat46":
		if len(flags.backendCidr) == 0 {
			log.Fatal("At least one -backend-cidr flag must be specified when you use -backend nat46")
		}
		if flags.nat46Prefix == nil {
			log.Fatal("-nat46-prefix must be specified when you use -backend nat46")
		}
		server.Backend = &TCPDialer{Port: flags.backendPort, Allowed: flags.backendCidr, IPv6SourcePrefix: flags.nat46Prefix}
	default:
		log.Fatal("-backend-type must be unix, tcp, or nat46")
	}

	if len(flags.listen) == 0 {
		log.Fatal("At least one -listen flag must be specified")
	}

	listeners, err := listener.OpenAll(flags.listen)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.CloseAll(listeners)

	for _, l := range listeners {
		go serve(l, server)
	}

	select {}
}

func serve(listener net.Listener, server *Server) {
	log.Fatal(server.Serve(listener))
}
