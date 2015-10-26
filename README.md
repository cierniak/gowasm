# gowasm

## Prerequisites
* [Install Go](https://golang.org/doc/install).
* Install [WASM prototype](https://github.com/WebAssembly/spec) if you want to execute the code produced by gowasm.
 
## Setup
If you're familiar with Go, you probably don't need any of these instructions but if you're new to Go, these copy and paste instructions may be a good start.

Create a new directory and assign it to a variable $GOWASM:
```
export GOWASM=[directory of your choice]
mkdir -p $GOWASM
```
Set up `$GOPATH` and fetch the gowasm project from github:
```
export GOPATH=$GOWASM
cd $GOWASM
mkdir src
cd src
git clone https://github.com/cierniak/gowasm.git
```
Compile and run gowasm:
```
cd $GOWASM
go install gowasm
bin/gowasm src/gowasm/tests/fac/fac.go
```
To see the list of available command line options, run:
```
bin/gowasm --help
```
If you installed the interpreted from the spec repo (see prerequisites), you can run the generated code with:
```
wasm -t out.wast
```
