# bootc tldr

prep

```
docker network create \
  --driver bridge \                                                                                                                       --subnet 172.30.0.0/24 \
  mybr

KIND_EXPERIMENTAL_DOCKER_NETWORK=mybr kind create cluster

docker run -d --name mybr-dhcp \
  --network mybr \
  --cap-add NET_ADMIN \
  --restart unless-stopped \
  andyshinn/dnsmasq \
  --dhcp-range=172.30.0.100,172.30.0.150,12h
```

find mybr link name

```
$ ip link | grep br-$(docker network inspect mybr | jq ".[0].Id" | tr -d '"' | cut -c1-7)
16: br-ce6671d13ec5: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP mode DEFAULT group default
17: veth91c57e1@if2: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue master br-ce6671d13ec5 state UP mode DEFAULT group default
19: vethda889af@if2: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue master br-ce6671d13ec5 state UP mode DEFAULT group default
21: vnet3: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue master br-ce6671d13ec5 state UNKNOWN mode DEFAULT group default qlen 1000
```

or using `brctl`:

```
brctl show
bridge name     bridge id               STP enabled     interfaces
br-ce6671d13ec5         8000.da3ea3facb5b       no              veth91c57e1
                                                        vethda889af
                                                        vnet3
docker0         8000.e2b85f223fef       no
virbr0          8000.525400765ff9       yes
virbr1          8000.5254003139d0       yes
```

so `mybr` is `br-ce6671d13ec5`

and use it as bridge name for vm
