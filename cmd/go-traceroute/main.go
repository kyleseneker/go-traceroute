package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"runtime"

	"github.com/urfave/cli/v2"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func main() {
	// Initialize and run the CLI application
	app := newApp()
	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// newApp creates a new CLI application with the traceroute configuration
func newApp() *cli.App {
	return &cli.App{
		Name:  "go-traceroute",
		Usage: "print the route packets take to network host",
		Flags: getFlags(),
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return cli.ShowAppHelp(c)
			}

			// Get traceroute configuration from CLI context
			config := getConfig(c)
			err := runTraceroute(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}

			return nil
		},
	}
}

// getFlags returns the list of CLI flags
func getFlags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name:    "packet_size",
			Value:   40,
			Usage:   "specify the size of the packets.",
			Aliases: []string{"s"},
		},
		&cli.IntFlag{
			Name:    "first_ttl",
			Value:   1,
			Usage:   "Set the initial time-to-live value used in outgoing probe packets.",
			Aliases: []string{"f"},
		},
		&cli.IntFlag{
			Name:    "max_ttl",
			Value:   64,
			Usage:   "Set the max time-to-live (max number of hops) used in outgoing probe packets.",
			Aliases: []string{"m"},
		},
		&cli.IntFlag{
			Name:    "port",
			Value:   33434,
			Usage:   "Sets the base port number used in probes.",
			Aliases: []string{"p"},
		},
		&cli.IntFlag{
			Name:    "wait",
			Value:   5,
			Usage:   "Set the time (in seconds) to wait for a response to a probe.",
			Aliases: []string{"w"},
		},
		&cli.IntFlag{
			Name:    "nqueries",
			Value:   3,
			Usage:   "Set the number of probes per 'ttl' to nqueries.",
			Aliases: []string{"q"},
		},
	}
}

// TracerouteConfig holds the configuration for the traceroute operation
type TracerouteConfig struct {
	hostname     string
	packetSize   int
	firstTTL     int
	maxTTL       int
	basePort     int
	waitTime     int
	numProbes    int
	destIP       string
	destHostname string
}

// getConfig reads the CLI context and returns a TracerouteConfig struct
func getConfig(c *cli.Context) *TracerouteConfig {
	hostname := c.Args().Get(0)
	packetSize := c.Int("packet_size")
	firstTTL := c.Int("first_ttl")
	maxTTL := c.Int("max_ttl")
	basePort := c.Int("port")
	waitTime := c.Int("wait")
	numProbes := c.Int("nqueries")

	// Resolve the IP address of the hostname
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not resolve hostname: %v\n", err)
		os.Exit(1)
	}
	destIP := addrs[0]

	// Check if there are multiple addresses
	if len(addrs) > 1 {
		fmt.Printf("traceroute: Warning: %s has multiple addresses; using %s\n", hostname, destIP)
	}

	return &TracerouteConfig{
		hostname:     hostname,
		packetSize:   packetSize,
		firstTTL:     firstTTL,
		maxTTL:       maxTTL,
		basePort:     basePort,
		waitTime:     waitTime,
		numProbes:    numProbes,
		destIP:       destIP,
		destHostname: hostname,
	}
}

// runTraceroute runs the traceroute operation based on the given configuration
func runTraceroute(config *TracerouteConfig) error {
	fmt.Printf("traceroute to %s (%s), %d hops max, %d byte packets\n", config.destHostname, config.destIP, config.maxTTL, config.packetSize)

	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		return fmt.Errorf("insufficient privileges: try running with sudo")
	}

	// Create ICMP packet listener
	recvConn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return fmt.Errorf("could not create receive socket: %v", err)
	}
	defer recvConn.Close()

	destinationReached := false

	// Iterate through TTL values
	for ttl := config.firstTTL; ttl < config.firstTTL+config.maxTTL && !destinationReached; ttl++ {
		respondingIP, results, allFailed, err := sendProbes(ttl, config, recvConn)
		if err != nil {
			return err
		}

		if !allFailed {
			// Determine DNS and IP address to print at the beginning of the line
			output := getOutput(respondingIP)
			// Print results
			printResults(ttl, output, respondingIP, results)
		}

		// Break if we have reached the destination after all probes for this TTL
		if respondingIP == config.destIP {
			destinationReached = true
		}
	}

	return nil
}

// sendProbes sends probes for a given TTL and returns the responding IP and results
func sendProbes(ttl int, config *TracerouteConfig, recvConn *icmp.PacketConn) (string, []string, bool, error) {
	var respondingIP string
	results := make([]string, config.numProbes)
	ttlPrinted := false
	allFailed := true

	for probe := 0; probe < config.numProbes; probe++ {
		dstAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", config.destIP, config.basePort+ttl))
		if err != nil {
			results[probe] = "*"
			continue
		}

		sendConn, err := net.DialUDP("udp4", nil, dstAddr)
		if err != nil {
			results[probe] = "*"
			continue
		}

		p := ipv4.NewPacketConn(sendConn)
		if err := p.SetTTL(ttl); err != nil {
			sendConn.Close()
			results[probe] = "*"
			continue
		}

		start := time.Now()

		_, err = sendConn.Write(make([]byte, config.packetSize))
		sendConn.Close()
		if err != nil {
			results[probe] = "*"
			continue
		}

		reply := make([]byte, 1500)
		err = recvConn.SetReadDeadline(time.Now().Add(time.Duration(config.waitTime) * time.Second))
		if err != nil {
			results[probe] = "*"
			continue
		}

		n, peer, err := recvConn.ReadFrom(reply)
		rtt := time.Since(start)
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				results[probe] = "*"
				if !ttlPrinted {
					fmt.Printf("%d  * ", ttl)
					ttlPrinted = true
				} else {
					fmt.Printf("* ")
				}
				continue
			}
			return "", nil, false, fmt.Errorf("could not read ICMP message: %v", err)
		}

		icmpMessage, err := icmp.ParseMessage(1, reply[:n])
		if err != nil {
			results[probe] = "*"
			if !ttlPrinted {
				fmt.Printf("%d  * ", ttl)
				ttlPrinted = true
			} else {
				fmt.Printf("* ")
			}
			continue
		}

		peerIP := peer.(*net.IPAddr).IP.String()

		switch icmpMessage.Type {
		case ipv4.ICMPTypeTimeExceeded, ipv4.ICMPTypeDestinationUnreachable:
			results[probe] = fmt.Sprintf("%.3f ms", rtt.Seconds()*1000)
			respondingIP = peerIP
			allFailed = false
		case ipv4.ICMPTypeEchoReply:
			if peerIP == config.destIP {
				results[probe] = fmt.Sprintf("%.3f ms", rtt.Seconds()*1000)
				respondingIP = peerIP
				allFailed = false
			} else {
				results[probe] = "*"
				if !ttlPrinted {
					fmt.Printf("%d  * ", ttl)
					ttlPrinted = true
				} else {
					fmt.Printf("* ")
				}
			}
		default:
			results[probe] = fmt.Sprintf("%.3f ms", rtt.Seconds()*1000)
			respondingIP = peerIP
			allFailed = false
		}
	}

	// Ensure a newline is printed after all probes for this TTL
	if allFailed {
		fmt.Println()
	}

	return respondingIP, results, allFailed, nil
}

// getOutput resolves the DNS name and filters out .localdomain
func getOutput(respondingIP string) string {
	if respondingIP == "" {
		return ""
	}

	hosts, err := net.LookupAddr(respondingIP)
	if err != nil || len(hosts) == 0 {
		return respondingIP
	}

	return strings.TrimSuffix(hosts[0], ".")
}

// printResults prints the traceroute results for a given TTL
func printResults(ttl int, output, respondingIP string, results []string) {
	if respondingIP != "" {
		fmt.Printf("%d  %s (%s)  ", ttl, output, respondingIP)
	} else {
		fmt.Printf("%d  ", ttl)
	}

	for _, result := range results {
		fmt.Printf("%s ", result)
	}
	fmt.Println()
}
