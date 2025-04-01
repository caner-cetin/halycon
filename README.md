# Halycon

[![Go Report Card](https://goreportcard.com/badge/github.com/caner-cetin/halycon)](https://goreportcard.com/report/github.com/caner-cetin/halycon)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

**Halycon is a command-line interface (CLI) tool designed to interact with the Amazon Selling Partner API (SP-API), automating various tasks related to catalog management, inventory, listings, and fulfillment.**

It provides utilities to streamline common workflows, such as converting product identifiers (UPC/ASIN/SKU), creating FBA shipment plans, managing product listings, retrieving product definitions, submitting feeds, and more.

- [Halycon](#halycon)
  - [Features](#features)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
  - [Configuration](#configuration)
  - [Usage](#usage)
    - [Common Flags](#common-flags)
    - [Commands](#commands)
      - [`upc-to-asin`](#upc-to-asin)
      - [`asin-to-sku`](#asin-to-sku)
      - [`shipment create`](#shipment-create)
      - [`shipment operation status`](#shipment-operation-status)
      - [`definition search`](#definition-search)
      - [`definition get`](#definition-get)
      - [`listings create`](#listings-create)
      - [`listings get`](#listings-get)
      - [`listings patch`](#listings-patch)
      - [`listings delete`](#listings-delete)
      - [`catalog get`](#catalog-get)
      - [`feeds upload` / `get` / `report`](#feeds-upload--get--report)
      - [`generate`](#generate)
      - [`version`](#version)
    - [Variations (Parent/Child Listings)](#variations-parentchild-listings)
  - [Development](#development)
  - [Contributing](#contributing)
  - [License](#license)

---

## Features

*   **Identifier Conversion:**
    *   Convert UPCs to ASINs (`upc-to-asin`).
    *   Convert ASINs to SKUs, preparing data for shipment plans (`asin-to-sku`).
*   **FBA Shipment Management:**
    *   Create FBA inbound shipment plans from SKU/quantity data (`shipment create`).
    *   Check the status of shipment plan operations (`shipment operation status`).
    *   Handles prep/label owner requirements with caching.
*   **Product Definitions:**
    *   Search for Amazon product type definitions using keywords or item names (`definition search`).
    *   Retrieve detailed product type definitions and schemas (`definition get`).
*   **Listing Management:**
    *   Create new product listings with attribute data (`listings create`).
    *   Retrieve existing listing details, including attributes and relationships (`listings get`).
    *   Update listings using JSON Patch operations (`listings patch`).
    *   Delete listings (`listings delete`).
    *   Support for creating Parent/Child variation relationships.
*   **Catalog Information:**
    *   Get detailed information about a catalog item by ASIN (`catalog get`).
*   **Feeds API:**
    *   Upload feed documents (`feeds upload`).
    *   Get feed processing status (`feeds get`).
    *   Download and display feed processing reports (`feeds report`).
*   **SP-API Client Generation:** Includes a script (`generate_swagger_client.sh`) using `oapi-codegen` to generate Go client code from SP-API OpenAPI specifications.
*   **Authentication & Rate Limiting:** Handles SP-API authentication (LWA token refresh) and implements rate limiting for API calls.
*   **Configuration:** Uses a YAML file for easy configuration of credentials and settings.
*   **(Experimental) AI Text Generation:** Includes a supplementary utility to interact with the Groq API for generating text based on prompts and images (`generate`).

## Prerequisites

*   **Go:** Version 1.23 or higher (see `go.mod`).
*   **Just:** A command runner, recommended for development and building (`https://github.com/casey/just`).
*   **Amazon SP-API Credentials:**
    *   A registered SP-API application (Client ID & Secret).
    *   A Refresh Token obtained by self-authorizing your application for your Seller account.
    *   Your Seller ID (Seller Token).
*   **(Optional) Groq API Key:** Required only for the `halycon generate` command. Store in the configuration file.

## Installation

1.  **Clone the Repository:**
    ```bash
    git clone https://github.com/caner-cetin/halycon.git
    cd halycon
    ```
2.  **Build using `just`:** (Requires `just` to be installed)
    *   Build for your current OS/Architecture:
        ```bash
        just build-current
        # Output: dist/halycon
        ```
    *   Build for multiple platforms:
        ```bash
        just build
        # Output: dist/halycon-<os>-<arch>[.exe]
        ```
    *   Build and create compressed packages for distribution:
        ```bash
        just package
        # Output: dist/halycon-<os>-<arch>.(tar.gz|zip)
        ```
3.  **(Alternative) Go Install:**
    ```bash
    # Ensure GOPATH/bin is in your PATH
    go install github.com/caner-cetin/halycon@latest
    # Or if proxies cause issues:
    # GOPROXY=direct go install github.com/caner-cetin/halycon@latest
    ```
4.  **(Optional) Pre-built Binaries:** Check the [Releases](https://github.com/caner-cetin/halycon/releases) page for pre-built binaries corresponding to the `just package` command output.

## Configuration

Halycon uses a YAML configuration file named `.halycon.yaml`.

1.  **Create the File:** Copy the provided `.halycon.dummy.yaml` file.
2.  **Rename:** Rename it to `.halycon.yaml`.
3.  **Location:** Place the file in your user home directory (`$HOME/.halycon.yaml`). Alternatively, specify a path using the `--config` flag.
4.  **Edit:** **Crucially, edit the file and fill in the required values.** Halycon will prompt you to select defaults if multiple clients/merchants/addresses are defined and none are marked as `default: true`.

    ```yaml
    # .halycon.yaml
    amazon:
      auth:
        # Define one or more SP-API applications
        clients:
          - id: YOUR_CLIENT_ID # REQUIRED
            secret: YOUR_CLIENT_SECRET # REQUIRED
            name: MyPrimaryApp # Optional: Reference name
            auth_endpoint: https://api.amazon.com/auth/o2/token # Optional: Defaults provided
            api_endpoint: sellingpartnerapi-na.amazon.com # Optional: Defaults provided (e.g., sellingpartnerapi-eu.amazon.com)
            default: true # REQUIRED if multiple clients defined
          # - id: ... (another client if needed)
        # Define one or more Seller accounts you've authorized
        merchants:
          - refresh_token: YOUR_REFRESH_TOKEN # REQUIRED
            seller_token: YOUR_SELLER_ID # REQUIRED
            marketplace_id: # REQUIRED: At least one marketplace ID
              - ATVPDKIKX0DER # Example: US
              # - A2EUQ1WTGCTBG2 # Example: CA
            name: MyUSAccount # Optional: Reference name
            default: true # REQUIRED if multiple merchants defined
          # - refresh_token: ... (another merchant)
      fba:
        # Set to true if you use FBA and need shipment commands
        enable: true
        # Define one or more ship-from addresses for FBA shipments
        ship_from:
          - address_line_1: 123 Main St # REQUIRED
            address_line_2: Suite 100 # Optional
            city: Anytown # REQUIRED
            company_name: My Company # Optional
            country_code: US # REQUIRED (Defaults to US)
            email: contact@example.com # Optional
            name: John Doe # REQUIRED (Contact Name)
            phone_number: 555-123-4567 # REQUIRED
            postal_code: "12345" # REQUIRED
            state_or_province_code: CA # REQUIRED (State/Province Code)
            default: true # REQUIRED if multiple addresses defined
          # - address_line_1: ... (another address)
      # Default language tag for operations requiring it (e.g., listings)
      default_language_tag: en_US # Optional: Defaults to en_US

    # Required only for the 'halycon generate' command
    groq:
      token: YOUR_GROQ_API_KEY
    ```

## Usage

```bash
halycon [command] [subcommand] [flags]
```

### Common Flags

*   `--config <path>`: Specify a configuration file path.
*   `-v`, `-vv`, `-vvv`: Increase output verbosity (Warn -> Info -> Debug -> Trace).

### Commands

#### `upc-to-asin`

Converts UPC(s) to ASIN(s). Useful for preparing product lists for other operations.

*   **Single UPC:**
    ```bash
    halycon upc-to-asin --single -i 754603373000
    ```
*   **List of UPCs (from file):**
    ```bash
    # Input file (e.g., upcs.txt) should have one UPC per line
    halycon upc-to-asin -i upcs.txt -o asins.txt
    # Output file (asins.txt) will have one ASIN per line
    ```

#### `asin-to-sku`

Looks up SKUs (and product names) for given ASIN(s). Creates a CSV file ready for shipment planning.

*   **Single ASIN:**
    ```bash
    halycon asin-to-sku --single -i B07H2WGKVB
    ```
*   **List of ASINs (from file):**
    ```bash
    # Input file (e.g., asins.txt) should have one ASIN per line
    halycon asin-to-sku -i asins.txt -o skus_for_shipment.csv
    ```
*   **Output CSV Format (`skus_for_shipment.csv`):**
    ```csv
    ASIN,SKU,Product Name,Quantity
    B07H2WGKVB,YOUR_SKU_1,"Example Product Title 1",
    B08EXAMPLE,YOUR_SKU_2,"Example Product Title 2",
    ...
    ```
    *Fill in the `Quantity` column before using this file with `shipment create`.*

#### `shipment create`

Creates an FBA Inbound Shipment Plan.

*   **Usage:**
    ```bash
    # Input CSV file requires columns: ASIN,SKU,Product Name,Quantity (Quantity must be filled)
    halycon shipment create -i skus_for_shipment.csv -v
    ```
    *   Outputs the `inbound_plan_id` and `operation_id`.
    *   Prompts to open the plan in Seller Central.
    *   Handles prep/label owner requirements automatically, caching choices in `halycon_item_requirements.json` in the system's temp directory.

#### `shipment operation status`

Checks the status of an FBA inbound operation (like plan creation).

*   **Usage:**
    ```bash
    halycon shipment operation status -i <operation_id> -v
    ```
    *   Displays the status (SUCCESS, FAILED, IN_PROGRESS). Shows detailed problems if the operation failed.

#### `definition search`

Searches for Amazon Product Type Definitions. Required for finding the correct `productType` for listing operations.

*   **By Keywords:**
    ```bash
    halycon definition search --keywords "socks, apparel"
    ```
*   **By Item Name (Title):**
    ```bash
    halycon definition search --item "My Awesome Product Title"
    ```
    *   Outputs a table with Display Name and the required Amazon Name (`productType`).

#### `definition get`

Retrieves the detailed schema and requirements for a specific Product Type Definition. Essential for constructing the `attributes` JSON for listing operations.

*   **Usage:**
    ```bash
    halycon definition get --type SOCKS -v > socks_definition.txt
    # Use --detailed for full property info including constraints
    halycon definition get --type SOCKS --detailed -v > socks_definition_detailed.json
    ```
    *   Outputs a summary and the schema location.
    *   With `--detailed`, prints a structured representation of the schema properties, constraints, and requirements, useful for building the attributes JSON. The output includes the official schema URL (e.g., `https://selling-partner-definitions-prod-iad.s3.amazonaws.com/schema/...`), which can be used with editors supporting JSON Schema for validation and autocompletion (though access might be restricted).

#### `listings create`

Creates a new product listing or a variation relationship.

*   **Usage:**
    ```bash
    halycon listings create --sku YOUR_NEW_SKU \
                           --type PRODUCT_TYPE_NAME \
                           --requirements LISTING \
                           --input attributes.json \
                           [--fill-marketplace-id] \
                           [--fill-language-tag] \
                           -v
    ```
    *   `--sku`: The Seller SKU for the new listing.
    *   `--type`: The Amazon Product Type Name (from `definition search`).
    *   `--requirements`: Usually `LISTING`.
    *   `--input`: Path to a JSON file containing the listing attributes (based on `definition get`).
    *   `--fill-marketplace-id`: Automatically adds the default marketplace ID to attribute objects where missing.
    *   `--fill-language-tag`: Automatically adds the default language tag (`en_US` unless overridden in config) to attribute objects where missing.
    *   **Validation:** If the submission fails, Halycon prints the issues reported by Amazon. Correct the `attributes.json` and retry.
    *   **Variations:** See the [Variations](#variations) section below.

#### `listings get`

Retrieves details for an existing listing.

*   **Usage:**
    ```bash
    halycon listings get --sku YOUR_EXISTING_SKU -v [--attributes] [--related]
    ```
    *   `--attributes`: Displays the listing's current attributes in a structured format, including JSON Paths useful for `listings patch`.
    *   `--related`: Also fetches and displays details for related parent/child SKUs.

#### `listings patch`

Updates an existing listing using JSON Patch operations.

*   **Usage:**
    ```bash
    halycon listings patch --sku YOUR_EXISTING_SKU --input patch.json -v
    ```
    *   `--input`: Path to a JSON file containing the patch operations (see RFC 6902 and example below). Requires a `productType` key and a `patches` array.
    *   **Finding Paths:** Use `halycon listings get --attributes` or `halycon definition get --detailed` to find the correct JSON Pointers (`path`) for attributes.
    *   **Example `patch.json`:**
        ```json
        {
          "productType": "PRODUCT_TYPE_NAME",
          "patches": [
            {
              "op": "replace",
              "path": "/attributes/item_name/0/value",
              "value": "New Updated Product Title"
            },
            {
              "op": "add",
              "path": "/attributes/bullet_point/-", // Append to bullet points array
              "value": { "value": "New Bullet Point", "language_tag": "en_US" }
            },
            {
               "op": "delete",
               "path": "/attributes/color/0", // Must match existing value for deletion
               "value": {"marketplace_id": "ATVPDKIKX0DER", "value": "Blue"}
            }
          ]
        }
        ```

#### `listings delete`

Deletes a listing.

*   **Usage:**
    ```bash
    halycon listings delete --sku YOUR_SKU_TO_DELETE -v [--related]
    ```
    *   `--related`: Also attempts to delete related parent/child SKUs associated with the target SKU.

#### `catalog get`

Retrieves catalog item details by ASIN.

*   **Usage:**
    ```bash
    halycon catalog get --asin B07H2WGKVB -v
    ```

#### `feeds upload` / `get` / `report`

Submits feeds (e.g., pricing/quantity updates) and checks their status.
**Note:** Many flat file and XML feed types are being deprecated by Amazon in favor of the Listings API or JSON feeds. See the [deprecation notice](https://developer-docs.amazon.com/sp-api/changelog/deprecation-of-feeds-api-support-for-xml-and-flat-file-listings-feeds).

*   **Upload Feed:**
    ```bash
    halycon feeds upload -i FeedFile.csv \
                        --feed-type FEED_TYPE_ENUM \
                        --content-type text/csv \
                        -v
    # Outputs Feed ID
    ```
*   **Get Feed Status:**
    ```bash
    halycon feeds get -i <feed_id> -v
    ```
*   **Get Feed Report:** (Downloads and prints the processing report if available)
    ```bash
    halycon feeds report -i <feed_id> -v
    ```

#### `generate`

**(Experimental)** Uses the Groq API for text generation based on an image and prompt. Requires `groq.token` in config.

*   **Usage:**
    ```bash
    # Image URL and command-line prompt
    halycon generate --input "https://example.com/image.jpg" --prompt "Describe this image."

    # Local image file and prompt file
    halycon generate --input-file ./image.png --prompt-file ./prompt.txt
    ```

#### `version`

Displays the Halycon version.

*   **Usage:**
    ```bash
    halycon version
    ```

### Variations (Parent/Child Listings)

Creating variations involves creating a parent listing and then linking child listings to it.

1.  **Create Parent Listing:** Use `halycon listings create` with attributes defining it as a parent and specifying the variation theme (e.g., `COLOR`, `SIZE`, `COLOR_SIZE`).
    *   **Key Parent Attributes (`attributes.json`):**
        ```json
        {
          "parentage_level": [{"value": "parent"}],
          "variation_theme": [{"name": "COLOR"}]
          // ... other parent-level attributes (brand, item_name, etc.)
        }
        ```
2.  **Create Child Listing(s):** Use `halycon listings create` for each child variation. Link them to the parent SKU.
    *   **Key Child Attributes (`child_attributes.json`):**
        ```json
        {
          "parentage_level": [{"value": "child"}],
          "variation_theme": [{"name": "COLOR"}], // Must match parent's theme
           // Link to the parent SKU created in step 1
          "relationship": [{"type": "VARIATION", "parent_sku": "PARENT_SKU_HERE"}],
          // The specific variation attribute value for this child
          "color": [{"value": "Red", "marketplace_id": "ATVPDKIKX0DER"}],
          // ... other child-specific attributes (UPC/EAN, price, quantity, etc.)
        }
        ```

## Development

*   **Setup:** Clone the repository and ensure Go and Just are installed.
*   **Generate SP-API Clients:** Run `just generate` to download OpenAPI specs and generate Go client code using `oapi-codegen`. This updates files in `internal/amazon/`.
*   **Build:** Use `just build-current` for local builds or `just build` / `just package` for releases.
*   **Linting:** Run `just lint` to execute `golangci-lint` using the `.golangci.yml` configuration.
*   **Pre-commit:** Uses `pre-commit` for automated checks (formatting, linting, etc.). Install with `pip install pre-commit` and run `pre-commit install`.

## Contributing

Contributions are welcome! Whether it's bug reports, feature requests, or code improvements, please feel free to open an issue or submit a pull request.

## License

Halycon is licensed under the **GNU General Public License v3.0**. See the [LICENSE](./LICENSE) file for details.
```
