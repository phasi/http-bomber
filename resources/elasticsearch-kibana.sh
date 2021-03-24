#!/bin/bash

## Deploy elasticsearch and kibana locally to visualize http test results.
## Run './elasticsearch-kibana.sh deploy' to deploy
## Run './elasticsearch-kibana.sh destroy' to destroy

## Settings
EK_NETWORK=elk
CONT_ELASTICSEARCH=elasticsearch
CONT_KIBANA=kibana


## Deploy resources
function deploy {

# Create network for elasticsearch and kibana
docker network create $EK_NETWORK

# deploy elasticsearch
docker run -d --name $CONT_ELASTICSEARCH --net $EK_NETWORK -p 9200:9200 -p 9300:9300 -e "discovery.type=single-node" elasticsearch:7.11.2

# deploy kibana
docker run -d --name $CONT_KIBANA --net $EK_NETWORK -p 5601:5601 -e ELASTICSEARCH_HOSTS=http://${CONT_ELASTICSEARCH}:9200 kibana:7.11.2
}

## Destroy resources
function destroy {
    echo "Destroying resources..."
    docker stop $CONT_ELASTICSEARCH $CONT_KIBANA
    docker rm $CONT_ELASTICSEARCH $CONT_KIBANA
    docker network rm $EK_NETWORK
    echo "Done"
}

# Subcommand
"$@"
