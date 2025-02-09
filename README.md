# free5gc-k8s


# Prérequis
- Ubuntu 20.04 LTS pour avoir le kernel 5.4 afin d'installer le module gtp5g
- Docker 
- Kind
- Kubectl
- Helm

## Installer le module gtp5g
```bash
sudo apt-get install linux-headers-$(uname -r) -y
git clone https://github.com/free5gc/gtp5g.git
cd gtp5g
make
sudo make install
sudo modprobe gtp
```

# Test des charts Helm
Partie basée sur le dépôt Orange-OpenSource/towards5gs-helm et plus particulièrement en utilisant la documentation suivante :
https://github.com/Orange-OpenSource/towards5gs-helm/blob/main/docs/demo/Setup-free5gc-on-KinD-cluster-and-test-with-UERANSIM.md

## Création du cluster Kind
```bash
kind create cluster --config cluster-kind.yaml
```

## Installer les plugins CNI dans les nodes du cluster
```bash
wget https://github.com/containernetworking/plugins/releases/download/v1.6.2/cni-plugins-linux-amd64-v1.6.2.tgz
# Dernière version des plugins CNI lors de la rédaction de ce document
tar -xvf cni-plugins-linux-amd64-v1.6.2.tgz
docker ps
# Récupérer les ids des containers des nodes
docker cp  . <container_id>:/opt/cni/bin/
# Copier le contenu du dossier cni-plugins-linux-amd64-v1.6.2 dans le container
# Faire la même chose pour les autres nodes
```

## Installer Multus CNI
```bash
git clone https://github.com/k8snetworkplumbingwg/multus-cni
cd multus-cni
cat ./deployments/multus-daemonset-thick.yml | sudo kubectl apply -f -
```

## Cloner le dépot Orange-OpenSource/towards5gs-helm
```bash
git clone https://github.com/Orange-OpenSource/towards5gs-helm.git
```

## Ajuster les configuration réseau de Free5GC pour le cluster Kind
### Exécuter les commandes suivantes dans le container executant le worker node
```
root@kind-worker:/# ip a
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
        valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host 
        valid_lft forever preferred_lft forever
10: eth0@if11: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP group default 
    link/ether 02:42:ac:12:00:03 brd ff:ff:ff:ff:ff:ff link-netnsid 0
    inet 172.18.0.3/16 brd 172.18.255.255 scope global eth0
        valid_lft forever preferred_lft forever
    inet6 fc00:f853:ccd:e793::3/64 scope global nodad 
        valid_lft forever preferred_lft forever
    inet6 fe80::42:acff:fe12:3/64 scope link 
        valid_lft forever preferred_lft forever

root@kind-worker:/# ip r
default via 172.18.0.1 dev eth0 
10.244.0.0/24 via 172.18.0.2 dev eth0 
172.18.0.0/16 dev eth0 proto kernel scope link src 172.18.0.3
```

### Éditer les fichiers de configuration de Free5GC
charts/free5gc/values.yaml
```yaml
n6network:
  enabled: true
  name: n6network
  type: ipvlan
  masterIf: eth0                # Interface maître du worker node
  mode: l3                      # Mode L3 pour le routage
  subnetIP: 172.18.0.0          # Subnet du worker node
  cidr: 16                      # Masque /16
  gatewayIP: 172.18.0.1         # Gateway corrigée (anciennement 172.18.0.0)
  excludeIP: 172.18.0.0         # IP à exclure
  ipam:
    type: static
    addresses:
      - address: "172.18.0.22/16"  # IP statique de l'UPF
```

charts/free5gc/charts/free5gc-upf/values.yaml
```yaml
n6if:
  ipAddress: 172.18.0.22   # Doit correspondre à l'IP définie dans n6network
  gatewayIP: 172.18.0.1    # Gateway cohérente avec le subnet
```

free5gc/charts/free5gc-upf/values.yaml
```yaml
n6if:  # DN
  ipAddress: 172.18.0.22
```

### Creation du volume persistant pour le UPF
- Création d'un dossier dans le worker node
```
docker exec -it <id du worker node> /bin/bash
mkdir -p /mnt/upf
```
- Appliquer la configuration pour créer le volume persistant
```
kubectl apply -f persistent.yaml
```

## Deployer les charts Helm
### Créer le namespace free5gc
```bash
kubectl create namespace free5gc
```

### Deployer les charts Helm
```bash
sudo helm -n free5gc install free5gc-premier ./free5gc/
```
