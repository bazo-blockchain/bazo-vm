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
* [Bazo VM - Bachelor Thesis 2018.pdf](https://github.com/bazo-blockchain/bazo-vm/releases/download/v1.0.0/BachelorThesis-VM-HSR-2018.pdf) 


## Development

###  Dependency Management

Packages are managed by [Go Modules](https://github.com/golang/go/wiki/Modules). 

Set the environment variable `GO111MODULE=on` and run `go mod vendor` 
to install all the dependencies into the local vendor directory.


