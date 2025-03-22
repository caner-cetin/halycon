#!/usr/bin/env sh
set -e

# =====================================
# Setup working directory
# =====================================
SCRIPT_DIR=$(pwd)
mkdir -p internal/amazon/catalog internal/amazon/fba_inbound internal/amazon/fba_inventory internal/amazon/listings internal/amazon/product_type_definitions internal/amazon/feeds models
# =====================================
# Install required tools
# =====================================
echo "Installing oapi-codegen..."
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
# =====================================
# Main function to process each API model
# =====================================
install_model() {
    JSON=$1
    REMOTE_MODEL_PATH=$2
    PACKAGE_FOLDER=$3
    LOCAL_MODEL_PATH=models/halycon_sp_api_${JSON}
    CONVERTED_MODEL_PATH=models/halycon_sp_api_${JSON%.json}.yaml

    echo "=== Processing ${JSON} ==="

    # Download the OpenAPI spec
    echo "Downloading specification..."
    curl -s -o ${LOCAL_MODEL_PATH} ${REMOTE_MODEL_REPO}/${REMOTE_MODEL_PATH}

    # Use Swagger converter API
    echo "Converting to OpenAPI 3.0 YAML using Swagger API..."
    SPEC_URL="${REMOTE_MODEL_REPO}/${REMOTE_MODEL_PATH}"
    curl -s -X GET "https://converter.swagger.io/api/convert?url=${SPEC_URL}" \
         -H "Accept: application/yaml" \
         -o ${CONVERTED_MODEL_PATH}

    # Check if conversion was successful
    if [ ! -s ${CONVERTED_MODEL_PATH} ]; then
        echo "Error: Conversion failed or produced empty file. Using local file instead..."
        curl -s -X POST "https://converter.swagger.io/api/convert" \
             -H "Accept: application/yaml" \
             -H "Content-Type: application/json" \
             --data-binary @${LOCAL_MODEL_PATH} \
             -o ${CONVERTED_MODEL_PATH}
    fi

    # Extract package name from folder path
    PACKAGE_NAME=$(basename ${PACKAGE_FOLDER})

    # Generate client code with oapi-codegen using the converted YAML
    echo "Generating client code..."
    oapi-codegen -package ${PACKAGE_NAME} \
        -generate types,client,spec \
        -o ${PACKAGE_FOLDER}/client.go \
        -response-type-suffix Resp \
        ${CONVERTED_MODEL_PATH}

    # Check if generation was successful
    if [ $? -ne 0 ]; then
        echo "Error generating client for ${PACKAGE_NAME}. Falling back to direct generation from JSON..."
        oapi-codegen -package ${PACKAGE_NAME} \
            -generate types,client,spec \
            -o ${PACKAGE_FOLDER}/client.go \
            ${LOCAL_MODEL_PATH}

        if [ $? -ne 0 ]; then
            echo "Error: Could not generate client for ${PACKAGE_NAME} even from original JSON."
            exit 1
        fi
    else
        echo "Successfully generated client for ${PACKAGE_NAME}"
    fi

    echo "=== Done with ${JSON} ==="
    echo ""
}

# =====================================
# Process each API model
# =====================================
REMOTE_MODEL_REPO="https://raw.githubusercontent.com/amzn/selling-partner-api-models/refs/heads/main/models"

install_model fulfillmentInbound_2024-03-20.json        fulfillment-inbound-api-model/fulfillmentInbound_2024-03-20.json            internal/amazon/fba_inbound
install_model fbaInventory.json                         fba-inventory-api-model/fbaInventory.json                                   internal/amazon/fba_inventory
install_model catalogItems_2022-04-01.json              catalog-items-api-model/catalogItems_2022-04-01.json                        internal/amazon/catalog
install_model listingsItems_2021-08-01.json             listings-items-api-model/listingsItems_2021-08-01.json                      internal/amazon/listings
install_model definitionsProductTypes_2020-09-01.json   product-type-definitions-api-model/definitionsProductTypes_2020-09-01.json  internal/amazon/product_type_definitions
install_model feeds_2021-06-30.json                     feeds-api-model/feeds_2021-06-30.json                                       internal/amazon/feeds

# =====================================
# Clean up dependencies
# =====================================
echo "Cleaning up dependencies..."
cd "${SCRIPT_DIR}"
go mod tidy

echo "Process completed! All clients have been generated successfully."
