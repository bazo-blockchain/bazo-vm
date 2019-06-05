# Bazo Virtual Machine for Smart Contract

[![Build Status](https://travis-ci.org/bazo-blockchain/bazo-vm.svg?branch=master)](https://travis-ci.org/bazo-blockchain/bazo-vm)
[![Go Report Card](https://goreportcard.com/badge/github.com/bazo-blockchain/bazo-vm)](https://goreportcard.com/report/github.com/bazo-blockchain/bazo-vm)
[![GoDoc](https://godoc.org/github.com/bazo-blockchain/bazo-vm?status.svg)](https://godoc.org/github.com/bazo-blockchain/bazo-vm)

Bazo VM is a stack-based virtual machine to execute smart contract programs on the Bazo blockchain.

Smart contracts can be written in [Lazo](https://github.com/bazo-blockchain/lazo) (a high-level smart contract language) 
and then compiled to Bazo bytecode (an intermediate language).
[Bazo-miner](https://github.com/bazo-blockchain/bazo-miner) executes the bytecode on Bazo VM 
and persist changes on the blockchain.

## Background 

The Bazo Blockchain is a blockchain to test diverse mechanisms and algorithms.
In the current version mechanisms to run it on mobile devices
and Proof of Stake are integrated. It was only possible to transfer Bazo
coins before this thesis. The idea of this work was to enhance the Bazo
Blockchain with smart contracts.

**Documents**
* [Bazo VM - Bachelor Thesis 2018.pdf](https://eprints.hsr.ch/682/1/FS%202018-BA-EP-Steiner-Meier-Integrating%20Smart%20Contracts%20into%20the%20Bazo%20Blockchain.pdf) 

## Development

Run `./scripts/set-hooks.sh` to setup git hooks.

###  Dependency Management

Packages are managed by [Go Modules](https://github.com/golang/go/wiki/Modules). 

Set the environment variable `GO111MODULE=on` and run `go mod vendor` 
to install all the dependencies into the local vendor directory.

### Run Unit Tests

    go test ./... 

It will run all tests in the current directory and all of its subdirectories.

To see the test coverage, run `./scripts/test.sh` and then open the **coverage.html** file.

### Run Lints

    ./scripts/lint.sh
    
It will run golint on all packages except the vendor directory.

## Using Bazo VM with Lazo

It is difficult to write Bazo bytecode manually. Therefore, it is recommended to use [Lazo](https://github.com/bazo-blockchain/lazo)
language to generate bytecode automatically. To use VM with a Lazo program, run the following command:

    lazo run program.lazo

It will generate Bazo bytecode from source code and directly execute it on the VM.