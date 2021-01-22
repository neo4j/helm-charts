# YAML Prototypes

This folder is a place to develop yaml files that do the behaviours we want.
They should later be converted into helm chart templates, 
but we develop them here so we can think about what we want to do and how they should work.

These should *not* be committed to any public facing repositories. 

## Helper scripts

### start-kind-cluster.sh
starts a 2 "node" cluster on `localhost` (one node is the control plane, so it might only be 1 node).

This uses the configuration in `kindconfig.yaml`.

### start-gke-cluster.sh
starts a 3 node cluster on GKE. You must remember to kill the cluster afterwards!

```shell script
# to kill a gke cluster:
gcloud container --project "${PROJECT_ID}" clusters delete ${CLUSTER_NAME} --zone "${ZONE}" --quiet
```

https://console.cloud.google.com/kubernetes/

## Installing Tools

### gcloud
```shell script
curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
echo "deb https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
sudo apt-get update
sudo apt-get install google-cloud-sdk
gcloud init
```

### kind 

```shell script
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.8.1/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/bin/
kind --version
```

### kubectl

```shell script
curl -LO "https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x ./kubectl
sudo mv ./kubectl /usr/local/bin/kubectl
kubectl version --client
```

## Deploying Neo4j
```shell script
kubectl create namespace neo4j
kubectl apply -f neo4j-gce-storageclass.yaml
gcloud compute disks create --size 10GB --type pd-ssd pd-ssd-disk-1 --zone=europe-west1-b
kubectl apply -f neo4j-pvc.yaml
kubectl apply -f neo4j-config.yaml
kubectl apply -f neo4j-deployment.yaml
kubectl apply -f neo4j-service.yaml
````
If you want to use Neo4j browser run
 ```shell script
kubectl port-forward deployment/neo4j-db -n neo4j 7474:7474 7687:7687
```

Open localhost:7474 in your favourite browser.

Enjoy :)