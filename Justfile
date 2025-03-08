default: build

name := "halycon"
version := "0.1.0"

build_dir := "dist"
build_flags := "-trimpath -ldflags='-s -w'"

list:
    @just --list

clean:
    rm -rf {{build_dir}}

setup:
    mkdir -p {{build_dir}}

tidy:
    go mod tidy

generate:
    #!/usr/bin/env sh
    go install github.com/go-swagger/go-swagger/cmd/swagger@latest
    REMOTE_MODEL_REPO="https://raw.githubusercontent.com/amzn/selling-partner-api-models/refs/heads/main/models"
    mkdir -p internal/amazon/catalog internal/amazon/fba_inbound internal/amazon/fba_inventory internal/amazon/listings models
    install_model() {
        JSON=$1
        REMOTE_MODEL_PATH=$2
        PACKAGE_FOLDER=$3
        LOCAL_MODEL_PATH=models/halycon_sp_api_${JSON}
        curl -o ${LOCAL_MODEL_PATH} ${REMOTE_MODEL_REPO}/${REMOTE_MODEL_PATH}
        swagger generate client -f ${LOCAL_MODEL_PATH} -t ${PACKAGE_FOLDER}
    }
    install_model fulfillmentInbound_2024-03-20.json fulfillment-inbound-api-model/fulfillmentInbound_2024-03-20.json internal/amazon/fba_inbound
    install_model fbaInventory.json                  fba-inventory-api-model/fbaInventory.json                        internal/amazon/fba_inventory
    install_model catalogItems_2022-04-01.json       catalog-items-api-model/catalogItems_2022-04-01.json             internal/amazon/catalog
    install_model listingsItems_2021-08-01.json      listings-items-api-model/listingsItems_2021-08-01.json           internal/amazon/listings

    go mod tidy

build: clean setup tidy
    #!/usr/bin/env sh
    GOOS=linux GOARCH=amd64 go build {{build_flags}} -o {{build_dir}}/{{name}}-linux-amd64
    GOOS=linux GOARCH=arm64 go build {{build_flags}} -o {{build_dir}}/{{name}}-linux-arm64

    GOOS=darwin GOARCH=amd64 go build {{build_flags}} -o {{build_dir}}/{{name}}-darwin-amd64
    GOOS=darwin GOARCH=arm64 go build {{build_flags}} -o {{build_dir}}/{{name}}-darwin-arm64

    GOOS=windows GOARCH=amd64 go build {{build_flags}} -o {{build_dir}}/{{name}}-windows-amd64.exe
    GOOS=windows GOARCH=arm64 go build {{build_flags}} -o {{build_dir}}/{{name}}-windows-arm64.exe

    chmod +x {{build_dir}}/{{name}}-linux-*
    chmod +x {{build_dir}}/{{name}}-darwin-*

build-current: tidy setup
    go build {{build_flags}} -o {{build_dir}}/{{name}}
    chmod +x {{build_dir}}/{{name}}

package: build
    #!/usr/bin/env sh
    cd {{build_dir}}
    
    tar czf {{name}}-linux-amd64.tar.gz {{name}}-linux-amd64
    tar czf {{name}}-linux-arm64.tar.gz {{name}}-linux-arm64
    
    tar czf {{name}}-darwin-amd64.tar.gz {{name}}-darwin-amd64
    tar czf {{name}}-darwin-arm64.tar.gz {{name}}-darwin-arm64
    
    zip {{name}}-windows-amd64.zip {{name}}-windows-amd64.exe
    zip {{name}}-windows-arm64.zip {{name}}-windows-arm64.exe