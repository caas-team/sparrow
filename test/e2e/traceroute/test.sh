#!/bin/bash

EXIT_CODE=0

function cleanup()
{
    kathara lclean
    yes | rm ./shared/api.json ./shared/prometheus.txt ./shared/sparrow
    exit $EXIT_CODE
}

function error() {
    echo "[ ERROR ]: $@"
    EXIT_CODE=1
}

function info() {
    echo "[ INFO ]: $@"
}

function success() {
    echo "[ SUCCESS ]: $@"
}

function check_prometheus_output() {
  if grep -q 'sparrow_traceroute_minimum_hops{target="200.1.1.7"} 3' ./shared/prometheus.txt; then
    success "The specific Prometheus output is present."
  else
    error "The specific Prometheus output is not present."
  fi
}

function check_api_output() {
    if jq -e '
        .data["200.1.1.7"].hops | 
        .["1"][0].addr.ip == "195.11.14.1" and 
        .["1"][0].reached == false and
        .["1"][0].ttl == 1 and
        .["2"][0].addr.ip == "100.0.0.10" and 
        .["2"][0].reached == false and
        .["2"][0].ttl == 2 and
        .["3"][0].addr.ip == "200.1.1.7" and 
        .["3"][0].reached == true and
        .["3"][0].ttl == 3 and
        .["3"][0].addr.port == 80' ./shared/api.json > /dev/null; then
        success "The API output matches the expected hops and conditions."
    else
        error "The API output does not match the expected hops and conditions."
        cat ./shared/api.json
    fi
}

trap cleanup EXIT

# Start the Kathará lab
kathara lstart


# Copy the binary into the shared folder
info "Using $SPARROW_BIN"
cp $SPARROW_BIN ./shared/sparrow

# Start Sparrow on pc1
kathara exec pc1 "/shared/sparrow run --config /shared/config.yaml" &

# Wait for 10 seconds to ensure Sparrow is up and running
sleep 10

# Curl the API of Sparrow
kathara exec pc1 "bash /shared/get_api.sh"

check_prometheus_output

check_api_output
