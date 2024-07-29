 # go-traceroute

`go-traceroute` is a Golang program that mimics the functionality of the UNIX [traceroute](https://linux.die.net/man/8/traceroute) command-line utility. It traces the route packets take to a network host, showing the round-trip time for each hop.

## Features

* Trace the route to a network host.
* Customize packet size, initial TTL, maximum TTL, base port number, wait time for a response, and the number of probes per TTL.
* Handle multiple IP addresses for a given hostname.
* Provide detailed timing information for each hop.

## Installation

To install the `go-traceroute` tool, you need to have Golang installed on your system. Follow these steps:

1. Clone the repository:

    ```sh
    git clone https://github.com/kyleseneker/go-traceroute.git
    ```

1. Navigate to the project directory:

    ```sh
    cd go-traceroute
    ```

1. Build the program:

    ```sh
    make build
    ```

1. Move the executable to a directory in your PATH:

    ```sh
    sudo mv go-traceroute /usr/local/bin
    ```

## Usage

The `go-traceroute` command can be used with various options to specify packet size, initial TTL, and maximum TTL.

### Basic Usage

To trace the route to a host:

```sh
go-traceroute dns.google.com
```

### Options

* `-s`, `--packet-size`: Specify the size of the packets (default: 40 bytes).
* `-f`, `--first_ttl`: Specify the initial time-to-live (TTL) (default: 1).
* `-m`, `--max_ttl`: Specify the maximum time-to-live (TTL) (default: 64).
* `-p`, `--port`: Specify the base port number used in probes (default: 33434).
* `-w`, `--wait`: Specify the time (in seconds) to wait for a response to a probe (default: 5 seconds).
* `-q`, `--nqueries`: Specify the number of probes per TTL (default: 3).

You can combine these options to customize the trace:

```sh
go-traceroute -s 72 -f 2 -m 30 -p 33435 -w 3 -q 4 dns.google.com
```

### Examples

Trace the route to `dns.google.com` with default settings:

```sh
$ go-traceroute dns.google.com
traceroute to dns.google.com (8.8.8.8), 64 hops max, 60 byte packets
1  192.168.0.1 (192.168.0.1)  10.047 ms 7.761 ms 6.676 ms 
2  192.168.1.254 (192.168.1.254)  9.337 ms 7.988 ms 9.185 ms 
3  119.208.99.1 (119.208.99.1)  15.312 ms 8.750 ms 9.176 ms 
4  79.150.19.174 (79.150.19.174)  14.048 ms 12.353 ms 10.380 ms 
5  * * * 
6  * * * 
7  30.130.19.87 (30.130.19.87)  23.189 ms 21.461 ms 19.394 ms 
8  * * * 
9  dns.google (8.8.8.8)  26.375 ms 20.127 ms 28.009 ms
```

## Contributing

Contributions are welcome! If you find a bug or want to add a new feature, please open an issue or submit a pull request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.