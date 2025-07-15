# Halycon

[![Go Report Card](https://goreportcard.com/badge/github.com/caner-cetin/halycon)](https://goreportcard.com/report/github.com/caner-cetin/halycon)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Nightly Build](https://github.com/caner-cetin/halycon/actions/workflows/nightly.yml/badge.svg)](https://github.com/caner-cetin/halycon/actions/workflows/nightly.yml)

**Halycon is a command-line interface (CLI) tool designed to interact with the Amazon Selling Partner API (SP-API), automating various tasks related to catalog management, inventory, listings, and fulfillment.**

It provides utilities to streamline common workflows, such as converting product identifiers (UPC/ASIN/SKU), creating FBA shipment plans, managing product listings, retrieving product definitions, submitting feeds, managing local FBA inventory cache, and more.

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
      - [`inventory build`](#inventory-build)
      - [`inventory count`](#inventory-count)
      - [`config`](#config)
      - [`generate` (Experimental)](#generate-experimental)
      - [`version`](#version)
    - [Variations (Parent/Child Listings)](#variations-parentchild-listings)
  - [Development](#development)
  - [Contributing](#contributing)
  - [License](#license)

---

## Features

*   **Identifier Conversion:**
    *   Convert UPCs to ASINs using the Catalog API (`upc-to-asin`).
    *   Convert ASINs to SKUs (and retrieve product names) using the local FBA inventory cache, preparing data for shipment plans (`asin-to-sku`).
*   **FBA Shipment Management:**
    *   Create FBA inbound shipment plans from SKU/quantity data (`shipment create`).
    *   Check the status of shipment plan operations (`shipment operation status`).
    *   Handles prep/label owner requirements automatically, caching choices locally (`halycon_item_requirements.json` in temp dir).
*   **Product Definitions:**
    *   Search for Amazon product type definitions using keywords or item names (`definition search`).
    *   Retrieve detailed product type definitions and schemas, including property details and constraints (`definition get`).
*   **Listing Management:**
    *   Create new product listings or variation relationships (`listings create`), with options to autofill marketplace ID and language tags.
    *   Retrieve existing listing details, including attributes (with JSON paths), summaries, issues, offers, and relationships (`listings get`). Option to fetch related parent/child listings.
    *   Update listings using JSON Patch operations (RFC 6902) (`listings patch`).
    *   Delete listings (`listings delete`), with an option to delete related parent/child listings.
    *   Support for creating Parent/Child variation relationships.
*   **Catalog Information:**
    *   Get detailed information about a catalog item by ASIN (`catalog get`).
*   **Feeds API:**
    *   Upload feed documents with specified content types (`feeds upload`).
    *   Get feed processing status (`feeds get`).
    *   Download and display feed processing reports, handling decompression (`feeds report`).
*   **FBA Inventory Caching & Search:**
    *   Build and maintain a local SQLite database of your FBA inventory summary with UPC data (`inventory build`).
    *   Interactive inventory management with advanced filtering, sorting, and multiple output formats (`inventory count`). Includes UPC tracking, quantity-based filtering, and preview functionality.
*   **SP-API Client Generation:** Includes a script (`generate_swagger_client.sh`) using `oapi-codegen` to generate Go client code from SP-API OpenAPI specifications.
*   **Authentication & Rate Limiting:** Handles SP-API authentication (LWA token refresh) and implements rate limiting for API calls based on documented SP-API limits.
*   **Configuration:** Uses a YAML file (`.halycon.yaml`) for easy configuration of credentials, endpoints, FBA addresses, and other settings. Handles multiple profiles (clients, merchants, addresses) with default selection. Includes an interactive configuration generator (`config`).
*   **Database Migrations:** Uses `goose` for managing the SQLite database schema migrations.
*   **(Experimental) AI Text Generation:** Includes a supplementary utility to interact with the Groq API for generating text based on prompts and images (`generate`).

## Prerequisites

*   **Go:** Version 1.23 or higher (see `go.mod`).
*   **Just:** A command runner, recommended for development and building (`https://github.com/casey/just`).
*   **Amazon SP-API Credentials:**
    *   A registered SP-API application (Client ID & Secret).
    *   A Refresh Token obtained by self-authorizing your application for your Seller account.
    *   Your Seller ID (Seller Token).
*   **SQLite:** Required for the `inventory` commands (managed internally, but the `fts5` build tag is needed).
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
    *   Build for multiple platforms (Linux, macOS, Windows - amd64/arm64):
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
    # Requires build tools (like C compiler) for sqlite fts5 bindings
    go install github.com/caner-cetin/halycon@latest
    # Or if proxies cause issues:
    # GOPROXY=direct go install github.com/caner-cetin/halycon@latest
    ```
4.  **(Optional) Pre-built Binaries:** Check the [Releases](https://github.com/caner-cetin/halycon/releases) page (including the `nightly` pre-release) for binaries corresponding to the `just package` command output.

## Configuration

Halycon uses a YAML configuration file named `.halycon.yaml`.

1.  **Create the File:** Copy the provided `.halycon.dummy.yaml` file.
2.  **Rename:** Rename it to `.halycon.yaml`.
3.  **Location:** Place the file in your user home directory (`$HOME/.halycon.yaml`). Alternatively, specify a path using the `--config <path>` flag.
4.  **Edit:** **Crucially, edit the file and fill in the required values.** Halycon will prompt you to select defaults interactively if multiple clients/merchants/addresses are defined and none are marked as `default: true`. The selections will be saved back to the config file.

    ```yaml
    # .halycon.yaml
    amazon:
      auth:
        # Define one or more SP-API applications
        clients:
          - id: YOUR_CLIENT_ID          # REQUIRED
            secret: YOUR_CLIENT_SECRET  # REQUIRED
            name: MyPrimaryApp         # Optional: Reference name
            auth_endpoint: https://api.amazon.com/auth/o2/token # Optional: Default provided
            api_endpoint: sellingpartnerapi-na.amazon.com       # Optional: Default provided (e.g., sellingpartnerapi-eu.amazon.com)
            default: true              # REQUIRED if multiple clients defined
          # - id: ... (another client if needed)

        # Define one or more Seller accounts you've authorized
        merchants:
          - refresh_token: YOUR_REFRESH_TOKEN # REQUIRED
            seller_token: YOUR_SELLER_ID      # REQUIRED
            marketplace_id:                  # REQUIRED: At least one marketplace ID
              - ATVPDKIKX0DER                 # Example: US
              # - A2EUQ1WTGCTBG2               # Example: CA
            name: MyUSAccount                 # Optional: Reference name
            default: true                     # REQUIRED if multiple merchants defined
          # - refresh_token: ... (another merchant)

      fba:
        # Set to true if you use FBA and need shipment/inventory commands
        enable: true
        # Define one or more ship-from addresses for FBA shipments
        ship_from:
          - address_line_1: 123 Main St          # REQUIRED
            address_line_2: Suite 100            # Optional
            city: Anytown                        # REQUIRED
            company_name: My Company             # Optional
            country_code: US                     # REQUIRED (Defaults to US)
            email: contact@example.com           # Optional
            name: John Doe                       # REQUIRED (Contact Name)
            phone_number: 555-123-4567           # REQUIRED
            postal_code: "12345"                 # REQUIRED
            state_or_province_code: CA           # REQUIRED (State/Province Code, e.g., TX, AZ)
            default: true                        # REQUIRED if multiple addresses defined
          # - address_line_1: ... (another address)

      # Default language tag for operations requiring it (e.g., listings)
      default_language_tag: en_US # Optional: Defaults to en_US

    # SQLite database path (used for inventory caching)
    sqlite:
      path: # Optional: Defaults to $HOME/.halycon.db

    # Required only for the 'halycon generate' command
    groq:
      token: YOUR_GROQ_API_KEY
    ```

## Usage

```bash
halycon [command] [subcommand] [flags]
```

### Common Flags

*   `--config <path>`: Specify a configuration file path (default: `$HOME/.halycon.yaml`).
*   `-v`, `-vv`, `-vvv`: Increase output verbosity (Warn -> Info -> Debug -> Trace).

### Commands

#### `upc-to-asin`

Converts UPC(s) to ASIN(s) using the Catalog API. Handles batching requests.

*   **Single UPC:**
    ```bash
    halycon upc-to-asin --single -i 754603373000
    ```
*   **List of UPCs (from file):**
    ```bash
    halycon upc-to-asin -i upcs.txt -o asins.txt
    ```

#### `asin-to-sku`

Looks up SKUs (and product names) for given ASIN(s) using the **local FBA inventory cache** (built via `inventory build`). Creates a CSV file ready for shipment planning.

*   **Single ASIN:**
    ```bash
    halycon asin-to-sku --single -i B07H2WGKVB
    ```
*   **List of ASINs (from file):**
    ```bash
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

Creates an FBA Inbound Shipment Plan using the FBA Inbound API.

*   **Usage:**
    ```bash
    halycon shipment create -i skus_for_shipment.csv -v
    ```
    *   Outputs the `inbound_plan_id` and `operation_id`.
    *   Prompts to open the plan in Seller Central.
    *   Handles prep/label owner requirements automatically based on API feedback, caching choices in `halycon_item_requirements.json` in the system's temp directory for future use. Retries the plan creation if prep/label requirements were initially missed.

#### `shipment operation status`

Checks the status of an FBA inbound operation (like plan creation) using the operation ID.

*   **Usage:**
    ```bash
    halycon shipment operation status -i <operation_id> -v
    ```
    *   Displays the status (e.g., SUCCESS, FAILED, IN_PROGRESS). Shows detailed problems if the operation failed.

#### `definition search`

Searches for Amazon Product Type Definitions using the Product Type Definitions API. Required for finding the correct `productType` for listing operations.

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
    halycon definition get --type SOCKS -v > socks_definition_summary.txt
    # Use --detailed for full property info including constraints and JSON Schema structure
    halycon definition get --type SOCKS --detailed -v > socks_definition_detailed.txt
    ```
    *   Outputs a summary including the official schema URL (e.g., `https://selling-partner-definitions-prod-iad.s3.amazonaws.com/schema/...`).
    *   With `--detailed`, prints a structured representation of the schema properties, constraints, requirements, and structure. This output is useful for building the attributes JSON and understanding the expected format.

#### `listings create`

Creates a new product listing or a variation relationship using the Listings Items API.

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
    *   `--requirements`: Specifies the requirements set being addressed (e.g., `LISTING`).
    *   `--input`: Path to a JSON file containing the listing attributes (based on `definition get`).
    *   `--fill-marketplace-id`: Automatically adds the default marketplace ID to attribute objects within the `attributes.json` where missing (useful for marketplace-specific attributes).
    *   `--fill-language-tag`: Automatically adds the default language tag (`en_US` unless overridden in config) to attribute objects where missing (useful for localized attributes like title, description, bullet points).
    *   **Validation:** If the submission fails, Halycon prints the issues reported by Amazon. Correct the `attributes.json` and retry.
    *   **Variations:** See the [Variations](#variations) section below.

#### `listings get`

Retrieves details for an existing listing using the Listings Items API.

*   **Usage:**
    ```bash
    halycon listings get --sku YOUR_EXISTING_SKU -v [--attributes] [--related]
    ```
    *   `--attributes`: Displays the listing's current attributes in a structured format, including JSON Pointers (e.g., `/attributes/item_name/0/value`) useful for `listings patch`.
    *   `--related`: Also fetches and displays details for related parent/child SKUs found in the listing's relationships.

#### `listings patch`

Updates an existing listing using JSON Patch operations (RFC 6902) via the Listings Items API.

*   **Usage:**
    ```bash
    halycon listings patch --sku YOUR_EXISTING_SKU --input patch.json -v
    ```
    *   `--input`: Path to a JSON file containing the patch operations. Requires a top-level `productType` key and a `patches` array.
    *   **Finding Paths:** Use `halycon listings get --attributes` or `halycon definition get --detailed` to find the correct JSON Pointers (`path`) for attributes.
    *   **Example `patch.json`:**
        ```json
        {
          "productType": "PRODUCT_TYPE_NAME", // The product type of the listing being patched
          "patches": [
            {
              "op": "replace",
              "path": "/attributes/item_name/0/value", // Target the 'value' field within the first 'item_name' object
              "value": "New Updated Product Title"
            },
            {
              "op": "add",
              "path": "/attributes/bullet_point/-", // Append to the 'bullet_point' array
              "value": { "value": "New Bullet Point Feature", "language_tag": "en_US" }
            },
            {
               "op": "remove", // Use 'remove' to delete an item/attribute
               "path": "/attributes/color/0" // Path to the specific item to remove (e.g., the first color object)
               // Note: 'remove' usually doesn't need a 'value', just the path.
               // If deleting a specific value requires matching, the API might need 'delete' op instead,
               // but standard JSON patch uses 'remove'. Check SP-API docs if needed.
            }
          ]
        }
        ```

#### `listings delete`

Deletes a listing using the Listings Items API.

*   **Usage:**
    ```bash
    halycon listings delete --sku YOUR_SKU_TO_DELETE -v [--related]
    ```
    *   `--related`: Also attempts to delete related parent/child SKUs associated with the target SKU based on its relationships.

#### `catalog get`

Retrieves catalog item details by ASIN using the Catalog API.

*   **Usage:**
    ```bash
    halycon catalog get --asin B07H2WGKVB -v
    ```

#### `feeds upload` / `get` / `report`

Submits feeds, checks their status, and retrieves reports using the Feeds API.
**Note:** Many flat file and XML feed types are being deprecated by Amazon in favor of the Listings API or JSON feeds. See the [deprecation notice](https://developer-docs.amazon.com/sp-api/changelog/deprecation-of-feeds-api-support-for-xml-and-flat-file-listings-feeds). This command is useful for feed types still supported (e.g., JSON feeds, pricing/quantity updates via specific feed types).

*   **Upload Feed:**
    ```bash
    halycon feeds upload -i FeedFile.jsonl \
                        --feed-type JSON_LISTINGS_FEED \
                        --content-type application/json; charset=UTF-8 \
                        -v
    ```
*   **Get Feed Status:**
    ```bash
    halycon feeds get --id <feed_id> -v
    ```
*   **Get Feed Report:**
    ```bash
    halycon feeds report --id <feed_id> -v
    ```

#### `inventory build`

Fetches the FBA inventory summary from the SP-API and populates/updates a local SQLite database (`$HOME/.halycon.db` by default). Creates an FTS5 index for searching product titles. Now includes UPC data collection from Amazon's Catalog API.

*   **Usage:**
    ```bash
    halycon inventory build -v
    halycon inventory build --force-rebuild -v
    ```

#### `inventory count`

Interactive inventory management tool that queries the **local FBA inventory cache** (built by `inventory build`) with advanced filtering, sorting, and output options. Features an interactive form interface for ease of use.

*   **Interactive Features:**
    *   **Search Keywords:** Wildcard pattern support (e.g., `*phone*`)
    *   **Quantity Filtering:** Zero quantity, low stock (≤5), normal stock (6-50), high stock (>50), custom ranges, or show all
    *   **Sorting Options:** By title, total quantity, or fulfillable quantity (ascending/descending)
    *   **Output Formats:** Interactive table, CSV file, or JSON file (with timestamps)
    *   **Preview Mode:** Shows first 5 results with key metrics before generating full output
    *   **UPC Display:** Shows Universal Product Codes alongside inventory data

*   **Usage:**
    ```bash
    halycon inventory count
    halycon inventory count -k "keyword"
    halycon inventory count -k "keyword" -o inventory_report.csv
    ```
    *   Maintains backward compatibility with existing command-line flags while providing enhanced interactive experience

#### `config`

Interactive configuration generator that creates a complete `.halycon.yaml` configuration file through a guided, form-based wizard. Eliminates the need to manually create or edit YAML configuration files.

*   **Interactive Setup Wizard:**
    *   **Amazon SP-API Configuration:** Default language tag, client credentials, API endpoints
    *   **Client Management:** Multiple client support with default selection
    *   **Merchant Configuration:** Refresh tokens, seller tokens, marketplace IDs
    *   **FBA Settings:** Ship-from addresses with complete contact information
    *   **Optional Services:** Groq AI integration, SQLite database settings
    *   **Validation:** Input validation and secure handling of sensitive data

*   **Usage:**
    ```bash
    halycon config
    halycon config --config /path/to/custom/.halycon.yaml
    ```
    *   Creates a fully functional `.halycon.yaml` file with proper formatting and validation
    *   Provides preview option to review generated configuration
    *   Maintains compatibility with existing configuration file structure

#### `generate` (Experimental)

Uses the Groq API for text generation based on an image and prompt. Requires `groq.token` in the configuration file.

*   **Usage:**
    ```bash
    halycon generate --input "https://example.com/image.jpg" --prompt "Describe this image."
    halycon generate --input-file ./product_image.png --prompt-file ./description_prompt.txt
    ```

#### `version`

Displays the Halycon version.

*   **Usage:**
    ```bash
    halycon version
    ```

### Variations (Parent/Child Listings)

Creating variations involves creating a parent listing and then linking child listings to it using the `listings create` command.

1.  **Create Parent Listing:** Use `halycon listings create` with attributes defining it as a parent (`parentage_level`) and specifying the `variation_theme` (e.g., `COLOR`, `SIZE`, `COLOR_SIZE`).
    *   **Key Parent Attributes (Example `parent_attributes.json`):**
        ```json
        {
          "brand": [{"value": "MyBrand", "marketplace_id": "ATVPDKIKX0DER"}],
          "item_name": [{"value": "My Product Base Name", "language_tag": "en_US"}],
          "parentage_level": [{"value": "parent"}],
          "variation_theme": [{"name": "COLOR"}], // Or "SIZE", "SIZE_COLOR" etc.
          // ... other required parent-level attributes (e.g., manufacturer, part_number if applicable)
          // Note: Parents typically DO NOT have identifiers like UPC/EAN, price, quantity.
        }
        ```
    *   Run: `halycon listings create --sku PARENT_SKU --type PRODUCT_TYPE --requirements LISTING --input parent_attributes.json -v`

2.  **Create Child Listing(s):** Use `halycon listings create` for each child variation. Link them to the parent SKU using the `relationship` attribute. Provide child-specific attributes like color/size, identifiers (UPC/EAN), price, quantity, etc.
    *   **Key Child Attributes (Example `child_attributes_red.json`):**
        ```json
        {
          "brand": [{"value": "MyBrand", "marketplace_id": "ATVPDKIKX0DER"}], // Must match parent
          "item_name": [{"value": "My Product - Red", "language_tag": "en_US"}], // Child specific name
          "parentage_level": [{"value": "child"}],
          "variation_theme": [{"name": "COLOR"}], // Must match parent's theme
           // Link to the parent SKU created in step 1
          "relationship": [{"type": "VARIATION", "parent_sku": "PARENT_SKU"}],
          // The specific variation attribute value for this child
          "color": [{"value": "Red", "marketplace_id": "ATVPDKIKX0DER"}],
          // Child-specific external identifier (REQUIRED for most categories)
          "externally_assigned_product_identifier": [{
              "marketplace_id": "ATVPDKIKX0DER",
              "type": "UPC", // or EAN, GTIN, etc.
              "value": "YOUR_CHILD_UPC_HERE"
          }],
          // ... other required child-specific attributes (price, quantity, images, description, etc.)
        }
        ```
    *   Run for each child: `halycon listings create --sku CHILD_SKU_RED --type PRODUCT_TYPE --requirements LISTING --input child_attributes_red.json --fill-marketplace-id --fill-language-tag -v` (use `--fill-*` flags if needed)

## Development

*   **Setup:** Clone the repository, ensure Go (1.23+) and Just are installed.
*   **Generate SP-API Clients:** Run `just generate` to download SP-API OpenAPI specs and generate/update the Go client code in `internal/amazon/` and model files in `models/` using `oapi-codegen`.
*   **Build:**
    *   `just build-current`: Build for your local OS/Arch. Requires C compiler for SQLite FTS5 bindings.
    *   `just build`: Cross-compile for multiple platforms.
    *   `just package`: Cross-compile and create archives.
    *   *Note:* Builds use the `fts5` tag (`--tags 'fts5'`) required for the inventory search functionality.
*   **Database Migrations:** Migrations are in `internal/db/migrations`. `goose` is used internally to apply them when commands needing the DB are run. `sqlc` is used (via `sqlc.yaml`) to generate Go DB access code from `internal/db/queries.sql`.
*   **Linting:** Run `just lint` to execute `golangci-lint` using the `.golangci.yml` configuration.
*   **Tidy Modules:** Run `just tidy` or `go mod tidy`.
*   **Pre-commit:** Uses `pre-commit` for automated checks (formatting, linting). Install with `pip install pre-commit` and run `pre-commit install` in the repo root.

## Contributing

Contributions are welcome! Whether it's bug reports, feature requests, or code improvements, please feel free to open an issue or submit a pull request.

## License

Halycon is licensed under the **GNU General Public License v3.0**. See the [LICENSE](./LICENSE) file for details.
