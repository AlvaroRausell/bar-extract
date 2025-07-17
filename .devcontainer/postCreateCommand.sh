#!/bin/bash
# Install ACT for GitHub Actions local testing
curl --proto '=https' --tlsv1.2 -sSf https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash
sudo mv ./bin/act /usr/local/bin