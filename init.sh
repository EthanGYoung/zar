#!/bin/bash

#https://golang.org/dl/
wget https://dl.google.com/go/go1.12.9.linux-amd64.tar.gz
sudo tar -xvf go1.12.9.linux-amd64.tar.gz
sudo mv go /usr/local


sudo add-apt-repository ppa:jonathonf/python-3.6
sudo apt-get update
sudo apt-get install python3.6
sudo apt-get install python3-pip
python3 -m pip install docker



# Set up environment
export GOROOT=/usr/local/go
export GOPATH=$HOME/Zar

export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
