# ros2-node-shim

## Description

This repository contains the code to the ros2 node shims. The ros2 node shims are supposed to deploy and handle ros nodes and keep a connection with the controller. Check out the controller repository to get more information about the controller (https://gitlab.stud.idi.ntnu.no/master-thesis-ros2-k3s/ros-2-custom-crd.git). If you want easy deployment of the ros2 node shims in your cluster look at https://gitlab.stud.idi.ntnu.no/master-thesis-ros2-k3s/ansible-ros2-shim for automatic deployment using ansible.

## Requirements
- go version 1.22.2
- Python 3.11.2 (optional)

## Configuration

Create config folders, ``config_shim_#`` for every ros node shim you want to use in your cluster. 

```bash
mkdir config_shim_#
```
In every ``config_shim_#`` folder your create you need to create a ``nodeShimConfig.txt`` file.

```
cd config_shim_#
nano nodeShimConfig.txt
```

In this file you will need to set configuration options to connect you shim to your controller and vice-versa. The imidiate change is replacing ``HOMEPATH`` by your home path on shim host. 

This is the txt file, if you changed the name of the certificates in ``auto-cert``, you need to check the configuration you have set in the repository https://gitlab.stud.idi.ntnu.no/master-thesis-ros2-k3s/auto-cert. The root path and the log path are 
the paths where your ros projects will get cloned to. You can change it if you desire. If you did not configure your ros
installation in a special way, you can leaver the ros2 path and the log path by default. 

The node port dns, is the dns address for the host running the controller (master node) and the port is the port given by ``kubectl get services``, and you should look for ``controller-service`` loadbalancer you deployed in your controller (https://gitlab.stud.idi.ntnu.no/master-thesis-ros2-k3s/ros-2-custom-crd). The port that you would enter is ``9101:PORT/TCP`` if you set it on default. To get a local dns see [here](README.md#how-to-set-up-dns-local).


The dds bridge is an option to translate incoming dds messages from ros and forward them to JSON to a specific
webserver. Here it is used to send images from ros publisher to a webserver, to see the final images. Here is more information
on the webserver https://gitlab.stud.idi.ntnu.no/master-thesis-ros2-k3s/imageflaskwebserver. 

You can more information on the controller/shim authentication [here](/docs/authentication.excalidraw.png). You need to create 
a certificate server and key for the server grpc in the controller wit Common name = DNS/IP of the controller host. This is required to authenticate all the ros 2 shim and register them as legitimate nodes. And at the same time on the shim side you 
need a client certificate with your key alongside the ca certificate to authenticate itself to the grpc server in the controller. It sends a token that terminates the authentication flow. If the token is legitimate the shim receives a port from the controller so it can initate its own grpc server.

In the second stage the shim disconnects from the controller and starts a new grpc server using its generated server certificate and key with Common name = DNS/IP of the shim and on the port given by the controller. The controller then queries and attempts to authenticate itself using the port and on the same IP/DNS address it received during the controller authentication. It uses  its own client certificate, key and ca certificate. The shim authenticates itself to the controller. The controller subsquently sends a register request alongside a token. This token is then validated on the shim, and if validated, it enables the controller to use all the grpc functions on the shim. 

If you want to assign a shim node as powerfull, you can tag the shim using the ``POWER`` variable, to either set it TRUE or FALSE.

```txt
ROS2_PATH = HOMEPATH/ros2_humble
ROOT_PATH = HOMEPATH/projects
LOG_PATH = HOMEPATH/projects/log
ROS_LOG_PATH = HOMEPATH/.ros/log
CA_PEM = ./ca/ca_cert.txt
CA_CLIENT_CERT = ./ca/client_cert_shim_2.txt
CA_CLIENT_KEY = ./ca/client_key_shim_2.txt
CA_SERVER_CERT = ./ca/server_cert_shim_3.txt
CA_SERVER_KEY = ./ca/server_key_shim_3.txt
NODE_PORT_DNS = DNS:PORT
SECRET = ./ca/token_shim.txt
SECRET_KEY = ./ca/token_key_ctrl.txt
POWER = BOOL
DDS_BRIDGE = BOOL
TOPIC = /processed_image_topic
ENDPOINT = http://IP:3002/api/upload_image
```

## How to Run

```bash
mkdir ca
```

```bash
go mod tidy
go run main.go
```
or

```bash
go build -o bin/ros2-node-shim.bin
cd /bin
./ros2-node-shim.bin
```

## How to set up dns .local

```bash
sudo apt install avahi-daemon
sudo systemctl start avahi-daemon 
sudo systemctl enable avahi-daemon
```

Check that the config is right 


```bash
sudo nano /etc/avahi/avahi-daemon.conf
``` 

You should see

```conf
[server] 
use-ipv4=yes 
use-ipv6=no 
allow-interfaces=eth0
```
Configure your hostname

```bash
sudo hostnamectl set-hostname x
```

Check if the configuration is correct

```bash
sudo nano /etc/hosts
```

Now you can use your hostname.local as your dns entry.


