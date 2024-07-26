# How to test the traceroute check

## What is this
Kathara is a container-based network emulation tool. The files in this folder configure a small test network using kathara.
In this case we use kathara to locally simulate a network with a webserver, a client and multiple network hops between them.
## Requirements
- install [ kathara ](https://github.com/KatharaFramework/Kathara)
- install wireshark (optional)

## How to

1. Start kathara network
In this folder run:
```bash
kathara lstart
```

This starts the test-lab ([ topology ](https://github.com/KatharaFramework/Kathara-Labs/blob/main/main-labs/basic-topics/static-routing/004-kathara-lab_static-routing.pdf))

2. Connect to the client system
In a separate terminal run:
```bash
kathara connect pc1
```


3. (optional) Explore the network
Aside from you, there are two routers and a webserver in this lab.
Tracerouting to the webserver shows us, that we need to go through the two routers to reach the webserver:
```bash
export WEBSERVER=200.1.1.7
root@pc1:/# traceroute $WEBSERVER
traceroute to 200.1.1.7 (200.1.1.7), 30 hops max, 60 byte packets
 1  195.11.14.1 (195.11.14.1)  0.972 ms  1.093 ms  1.095 ms
 2  100.0.0.10 (100.0.0.10)  1.543 ms  1.712 ms  1.838 ms
 3  200.1.1.7 (200.1.1.7)  2.232 ms  2.310 ms  2.394 ms
```

We can also look at the server website:
```bash
root@pc1:/# curl $WEBSERVER
```
This should return the default apache website.

4. Run sparrow

To run sparrow we first need to build and move the sparrow binary into the container. Luckily, kathara mounts a shared folder to all systems in the lab. 
We can use this folder to run sparrow in the containers without having to build our own image! 

```bash
go build -o sparrow . && mv sparrow pkg/checks/traceroute/test-lab/shared/
```

Back in the client container:
```bash
root@pc1:/# cd /shared
root@pc1:/shared# ./sparrow -h
Sparrow is an infrastructure monitoring agent that is able to perform different checks.
The check results are exposed via an API.

Usage:
  sparrow [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  run         Run sparrow

Flags:
      --config string   config file (default is $HOME/.sparrow.yaml)
  -h, --help            help for sparrow

Use "sparrow [command] --help" for more information about a command.
```
Now we just have to create a config for sparrow to use and we're ready to develop. For testing traceroute I used this config:
```yaml
root@pc1:/shared# cat config.yaml
name: sparrow.dev
loader:
  type: file
  interval: 30s
  file:
    path: ./config.yaml
traceroute:
  interval: 5s
  timeout: 3s
  retries: 3
  maxHops: 8
  targets:
    - addr: 200.1.1.7
      port: 80
```


Now just run sparrow in the shared folder:

```bash
root@pc1:/shared# ./sparrow run --config config.yaml
```

5. Other tools
The container image has a bunch of utilities for debugging network issues. If you're debugging low level issues, where you need to inspect 
specific network packets you can use tcpdump directly, which is preinstalled:
```bash
root@pc1:/# tcpdump
tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on eth0, link-type EN10MB (Ethernet), snapshot length 262144 bytes
14:30:40.022157 IP 195.11.14.5 > 200.1.1.7: ICMP echo request, id 66, seq 1, length 64
14:30:40.023392 IP 200.1.1.7 > 195.11.14.5: ICMP echo reply, id 66, seq 1, length 64
^C
2 packets captured
2 packets received by filter
0 packets dropped by kernel
```

Or dump to a file which you can the inspect with wireshark:

```bash
# capture in kathara client to shared folder
root@pc1:/# tcpdump -w /shared/dump.pcap
# open dumped capture in wireshark on your host system
wireshark -r dump.pcap
```


Happy Debugging!
