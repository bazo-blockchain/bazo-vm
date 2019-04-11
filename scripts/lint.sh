#!/usr/bin/env bash
golint -set_exit_status $(go list ./... | grep -v /vendor/)