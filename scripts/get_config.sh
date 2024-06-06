
#!/bin/bash

# Check if two arguments are passed
if [ "$#" -ne 3 ]; then
    echo "Usage: $0 <DNS_NAME> <REPO_URL> <TOKEN>"
    exit 1
fi

DNS_NAME=$1
REPO_URL=$2
TOKEN=$3

# Download the file
HTTP_RESPONSE=$(curl -s -w "%{http_code}" --header "PRIVATE-TOKEN: $TOKEN" -o .sparrow.yaml "$REPO_URL")

# Check if the download was successful
if [ "$HTTP_RESPONSE" -ne 200 ]; then
    echo "Failed to download .sparrow.yaml, HTTP response code: $HTTP_RESPONSE"
    exit 1
fi

# Replace <TOKEN> and <DNS_NAME> with the provided arguments
sed -i "s/<DNS_NAME>/$DNS_NAME/g" .sparrow.yaml
sed -i "s/<TOKEN>/$TOKEN/g" .sparrow.yaml

echo "Downloaded and modified .sparrow.yaml configuration file"

# Move the modified cfg.yaml to the home directory as .sparrow.yaml
mv .sparrow.yaml $HOME/.sparrow.yaml

echo "Moved .sparrow.yaml to HOME: $HOME/.sparrow.yaml"