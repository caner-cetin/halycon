# halycon

utilities for Amazon SP API, mostly for finding cure for my annoyances

- [halycon](#halycon)
  - [usage](#usage)
    - [pre-built binaries](#pre-built-binaries)
    - [build yourself](#build-yourself)
    - [go](#go)
  - [UPC to ASIN](#upc-to-asin)
    - [what](#what)
    - [why](#why)
    - [how](#how)
  - [Shipments from ASIN List](#shipments-from-asin-list)

## usage

fill the `.halycon.dummy.yaml`, rename to `.halycon.yaml`, move the config to home directory, then

### pre-built binaries

i will upload binaries to releases soon tm

### build yourself

```bash
just build-current
mv ./dist/halycon /usr/local/bin/halycon
halycon --help
```

### go

```bash
GOPROXY=direct go install github.com/caner-cetin/halycon@latest
halycon --help
```

## UPC to ASIN

### what

converts list of UPCs or a single UPC to ASIN

### why

On `Send To Amazon` page, while creating shipment plans, you can search by SKU, Title, ASIN and FNSKU for the products you want to ship.

![alt text](./static/upc-to-asin-1.jpg)

Guessed what is missing? UPC!!!!!!!!!!!!

So whenever my brother sends a three page invoice to me like "create shipments for this", I have to do this 
```
switch tab -> fba inventory -> search with upc -> copy asin -> switch tab -> send to amazon -> search with asin -> enter quantity
```
for every single product.

`upc-to-asin` simplifies this process just to
```
send to amazon -> search with asin -> enter quantity
```
(dw, there is another command for creating shipment plans and skipping this process too)

### how

single upc, for debugging purposes
```bash
halycon upc-to-asin --single --upc 754603373107 -vvv
```
for list of upcs
```bash
halycon upc-to-asin --upc list.txt --output out.txt
```
where list is newline delimited (one per line) text file
```bash
754603337918
754603337840
...
```
output will be in same format.
```bash
B00M553N8E
B0182K0QJO
...
```

## Shipments from ASIN List

Too lazy for this? Not simplified enough?
```
send to amazon -> search with asin -> enter quantity
```
I got you!

*uhhh, work in progress*