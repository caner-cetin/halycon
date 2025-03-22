# halycon

utilities for Amazon SP API, mostly for my annoyances

- [halycon](#halycon)
  - [usage](#usage)
    - [pre-built binaries](#pre-built-binaries)
    - [build yourself / development](#build-yourself--development)
  - [utilities](#utilities)
    - [UPC to ASIN](#upc-to-asin)
      - [what](#what)
      - [why](#why)
      - [how](#how)
    - [ASIN to SKU](#asin-to-sku)
      - [why](#why-1)
      - [how](#how-1)
    - [Shipments from SKU Data](#shipments-from-sku-data)
      - [how](#how-2)
    - [Search Product Type Definitions](#search-product-type-definitions)
      - [why](#why-2)
      - [how](#how-3)
    - [Get Product Type Definitions](#get-product-type-definitions)
      - [why](#why-3)
      - [how](#how-4)
    - [Create Listing](#create-listing)
      - [how](#how-5)
    - [Delete Listing](#delete-listing)
      - [how](#how-6)
    - [Create Child-Parent Listings](#create-child-parent-listings)
      - [how](#how-7)
    - [Editing Listings](#editing-listings)
      - [why](#why-4)
      - [how](#how-8)
  - [halycon?](#halycon-1)
    - [Image-Text to Text AI INference](#image-text-to-text-ai-inference)
      - [why](#why-5)
      - [how](#how-9)

## usage

fill the `.halycon.dummy.yaml`, rename to `.halycon.yaml`, move the config to home directory, then

### pre-built binaries

https://github.com/caner-cetin/halycon/releases

### build yourself / development

```bash
# for swagger models
just generate
just build-current
mv ./dist/halycon /usr/local/bin/halycon
halycon --help
```
## utilities
### UPC to ASIN

#### what

converts list of UPCs or a single UPC to ASIN

#### why

On `Send To Amazon` page, while creating shipment plans, you can search by SKU, Title, ASIN and FNSKU for the products you want to ship.

![alt text](./static/upc-to-asin-1.jpg)

Guess what is missing? UPC!!!!!!!!!!!!

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

#### how

single upc, for debugging purposes
```bash
halycon upc-to-asin --single --input 754603373000 -vvv
```
for list of upcs
```bash
halycon upc-to-asin --input list.txt --output out.txt
```
where list is newline delimited (one per line) text file
```bash
754603337000
...
```
output will be in same format.
```bash
B07H2WGKVB
...
```

### ASIN to SKU

#### why
for creating shipment plans, you need, SKU and ASIN's.
#### how

single asin, for debugging purposes
```bash
halycon asin-to-sku --single --input B07H2WGKVB -vvv
```
for list of asins
```bash
halycon asin-to-sku -vvv --input out.txt --output out.csv
```
where input is list of ASIN's
```bash
B07H2WGKVB
...
```
and output will be
```
ASIN,SKU,Product Name,Quantity
B07H2WGKVB,some_sku,"Aneco 6 Pairs Over Knee Thigh Socks Knee-High Warm Stocking Women Boot Sock Leg Warmer High Socks for Daily Wear, Cosplay",
```
fill the quantity column and move on to `Shipments from SKU Data`


### Shipments from SKU Data

#### how
```bash
halycon shipment create -i sku.csv -v
```
where the input is the output of `asin-to-sku` command
```
ASIN,SKU,Product Name,Quantity
B07H2WGKVB,some_sku,"Aneco 6 Pairs Over Knee Thigh Socks Knee-High Warm Stocking Women Boot Sock Leg Warmer High Socks for Daily Wear, Cosplay",5
```

so for creating a shipment from list of UPCs (which is one of the main goals here), usual workflow is
```bash
halycon upc-to-asin      -i upc.txt -o asin.txt
halycon asin-to-sku      -i asin.txt -o sku.csv
halycon shipment create  -i sku.csv -v
```
after this, operation and workflow ID will be displayed
```bash
2:40AM INF success! inbound_plan_id=wf00a0e0a5-XXXX-XXX-XXXX-XXXXXXXXXXXX operation_id=78200213-XXXX-XXX-XXXX-XXXXXXXXXXXX
```
then, if requested, `https://sellercentral.amazon.com/fba/sendtoamazon/confirm_content_step?wf=wf00a0e0a5-XXXX-XXX-XXXX-XXXXXXXXXXXX` will open with default browser for confirming and finalizing the shipment on dashboard.

if you see the error `Please review SKUs with errors or unconfirmed SKUs` on dashboard, check the operation status.
```bash
halycon shipment operation status -i 78200213-XXXX-XXX-XXXX-XXXXXXXXXXXX -v
```
which will give you the status, if success or in progress, message will be displayed with `INFO` level, so you may not see anything.
```bash
2:40AM INF id=78200213-XXXX-XXX-XXXX-XXXXXXXXXXXX status=SUCCESS
```
if operation failed, problems will be displayed line by line
```bash
halycon shipment operation status -i 0aa45ad9-XXXX-XXX-XXXX-XXXXXXXXXXXX
2:34AM WRN id=0aa45ad9-XXXX-XXX-XXXX-XXXXXXXXXXXX status=FAILED
2:34AM WRN problem 1 code=FBA_INB_0049 details="There's an input error with the resource 'SU-XXXX-XXXX'." severity=ERROR
```

### Search Product Type Definitions
#### why
amazon name (product type) is required for [`PUT /listings/2021-08-01/items/{sellerId}/{sku}`](https://developer-docs.amazon.com/sp-api/lang-tr_TR/docs/listings-items-api-v2021-08-01-reference#listingsitemputrequest)
#### how

```bash
#     --item string            The title of the ASIN to get the product type recommendation. Note: Cannot be used with keywords
#     --keywords stringArray   A comma-delimited list of keywords to search product types. Note: Cannot be used with itemName.
halycon definition search --keywords SOCKS
# or
halycon definition search --item "Aneco 6 Pairs Over Knee Thigh Socks Knee-High Warm Stocking Women Boot Sock Leg Warmer High Socks for Daily Wear, Cosplay"
```
```
+--------------+-------------+
| DISPLAY NAME | AMAZON NAME |
+--------------+-------------+
| Sock         | SOCKS       |
+--------------+-------------+
```
### Get Product Type Definitions
#### why
attributes are required for [`PUT /listings/2021-08-01/items/{sellerId}/{sku}`](https://developer-docs.amazon.com/sp-api/lang-tr_TR/docs/listings-items-api-v2021-08-01-reference#listingsitemputrequest)
#### how
```bash
halycon definition get --type SOCKS -v --detailed > reference.json
```
example output [can be found here](./static/example/wallet_definition.txt)

required values are prefixed with `*`

ctrl+f is your friend

### Create Listing
#### how
```bash
halycon listings create --input attributes.json --type WALLET --requirements LISTING --sku W9-XXXX-XXXX --fill-marketplace-id --fill-language-tag -v
```
create `attributes.json` and fill with taking `halycon definition get` output as your reference OR

<details>

  <summmary>if you are using VSCode</summary>

  > You can refer your JSON Schema in $schema node and get your intellisense in VS Code right away. No need to configure anywhere else.

  `halycon definition get` outputs the required schema https://selling-partner-definitions-prod-iad.s3.amazonaws.com/schema/..., so you can just do
  ```json
  {
    "$schema": "http://json.schemastore.org/coffeelint", // change the schema link here
  }
  ```
  to get the intellisense. If you get Access Denied error from the schema URL (not from browser, from the VSCode), which is, completely normal, host the schema somewhere else like pastebin.

  if you do not prefer the intellisense, `halycon definition get` output is as detailed as it can get, so it is still a solid reference. your choice.

  if using intellisense and the autofill flags (`--fill-marketplace-id` etc), please ignore the `Missing property "language_tag"` etc.

</details>

```json
  {
     "country_of_origin": [{
        "value": "US",
        "marketplace_id": "ATVPDKIKX0DER"
      }],
      "item_name": [{
        "value": "Aneco 6 Pairs Over Knee Thigh Socks Knee-High Warm Stocking Women Boot Sock Leg Warmer High Socks for Daily Wear, Cosplay",
        "language_tag": "en_US",
        "marketplace_id": "ATVPDKIKX0DER"
      }],
      "item_type_keyword": [{
        "value": "Thigh highs",
        "marketplace_id": "ATVPDKIKX0DER"
      }],
      "brand": [{
        "value":"something something",
        "language_tag": "en_US",
        "marketplace_id": "ATVPDKIKX0DER"
      }],
       "bullet_point": [
        {
          "value": "TEAM HERITAGE: Features team design......",
          "marketplace_id": "ATVPDKIKX0DER",
          "language_tag": "en_US"
        },
        {
          "value": "PREMIUM MATERIAL: Crafted from high-quality black faux leather with durable stitching for long-lasting everyday use",
          "marketplace_id": "ATVPDKIKX0DER",
          "language_tag": "en_US"
        },
        {
          "value": "PREMIUM MATERIAL: Crafted from high-quality black faux leather with durable stitching for long-lasting everyday use",
          "marketplace_id": "ATVPDKIKX0DER",
          "language_tag": "en_US"
        },
      ],
  }
```
if you are gigalazy like me, use the `--fill-marketplace-id` flag, this will visit every object and add the `"marketplace_id": ...` for you, which, should save some time.

when using `--fill-marketplace-id`, first marketplace ID from config is used, if you need to specify a different `marketplace_id`, just write it in the attribute. autofill will skip the object if `marketplace_id` key is already present.

`--fill-language-tag` also exists, and works with the same logic in `--fill-marketplace-id`. so if you use both, you just have to write
```json
  {
      "country_of_origin": [{"value": "US"}],
      "item_name": [{"value": "Aneco 6 Pairs Over Knee Thigh Socks Knee-High Warm Stocking Women Boot Sock Leg Warmer High Socks for Daily Wear, Cosplay"}],
      "item_type_keyword": [{"value": "Thigh highs"}],
      "brand": [{"value":"something something"}],
       "bullet_point": [
        {"value": "TEAM HERITAGE: Features team design......"},
        {"value": "PREMIUM MATERIAL: Crafted from high-quality black faux leather with durable stitching for long-lasting everyday use"},
        {"value": "PREMIUM MATERIAL: Crafted from high-quality black faux leather with durable stitching for long-lasting everyday use"}
      ]
  }
```
and any extra attributes if required.

for validation, you can hit
```bash
halycon listings create
```
whenever you want, there is no need for dry run. on error, operation will fail, and errors will be listed.
```bash
9:03AM WRN 'Model Number' is required but not supplied. attribute=model_number code=90220 severity=ERROR
9:03AM WRN Based on the data from '[ships_globally.value]', the field '"value"' for the attribute 'Compliance - Wallet Type' is not allowed. Expected at most '0' of field '"value"' for attribute 'Compliance - Wallet Type'. attribute=compliance_wallet_type code=90248 severity=ERROR
9:03AM WRN The provided value for 'Item Weight' is invalid. attribute=item_weight code=4000001 severity=ERROR
9:03AM WRN 'Target Gender' is required but not supplied. attribute=target_gender code=90220 severity=ERROR
...
```
also the documentation is misleading. setting `MODE` to `VALIDATION_PREVIEW` while creating a listing ([as guided here](https://developer-docs.amazon.com/sp-api/docs/listings-items-api-v2021-08-01-use-case-guide#step-1-preview-errors-for-a-listings-item-put-request)) will end up with `Invalid Payload` error. soo. i cant provide you a "dry run" option even if I wanted to, so just, attempt creating listing over and over again.

if success,
```bash
halycon listings create -i attributes.json --type WALLET --requirements LISTING --sku W9-XXXX-XXXX --fill-marketplace-id --fill-language-tag -v
10:47AM INF sku=W9-XXXX-XXXX status=ACCEPTED submission_id=582xxxxxxxxxxxx
```
then you can use the same sku for `Get Listing`

### Delete Listing
#### how
```bash
halycon listings delete --sku W9-XXXX-XXXX -v
11:14AM INF sku=W9-XXXX-XXXX status=ACCEPTED submission_id=XXXXXXXXX
```

### Create Child-Parent Listings
#### how
same command as `Create Listing`
```bash
halycon listings create -i attributes.json --type WALLET --requirements LISTING --sku W9-XXXX-XXXX --fill-marketplace-id --fill-language-tag -v
```
create parent listing with the attributes
```json
{
  "parentage_level": [{ "value": "parent" }],
  "child_parent_sku_relationship": [{ "child_relationship_type": "variation" }],
  "variation_theme": [{ "name": "TEAM_NAME" }]
}
```
and then create the child listings with attributes
```json
{
  "parentage_level": [{"value": "children"}],
  "child_parent_sku_relationship": [{"child_relationship_type": "variation", "parent_sku": "W9-XXXX-XXXX"}],
  "variation_theme": [{ "name": "TEAM_NAME" }]
}
```

### Editing Listings
#### why
*you can edit the listing from dashboard, it is easy, why do you need this*

yes, indeed you can edit the listing from dashboard, and I recommend you to do that way if you can. API for editing listings requires a bit too much manual labour, if you can, edit from dashboard. if you cant, my condolences. follow along.
#### how
essentially, all you need to do is writing JSON patches. see the examples here https://datatracker.ietf.org/doc/html/rfc6902#appendix-A

amazon supports `add`, `replace` and `delete` operations.

for example, to change scheduled pricing change for a listing:
```json
{
  "productType": "PRODUCT",
  "patches": [
    {
      "op": "replace",
      "path": "/attributes/purchasable_offer/0/our_price/0/schedule/0/value_with_tax",
      "value": [
        {
          "value":"14.95",
        }
      ]
    }
  ]
}
```
where
```
purchasable_offer - The overall offer details for your product
our_price - Your selling price information
schedule - A scheduled price change configuration
value_with_tax - The price amount including any applicable tax, set to $14.95
```

to figure out the possible paths with your current flags, use `listings get` with the `--attributes` flag
```bash
halycon listings get --sku WOULD-I-LOOK-CUTE-IN-MAID-OUTFIT -v --attributes
```
which will give you a lengthy list of attributes with their corresponding paths and objects
```
      /attributes/purchasable_offer (Array)
        /attributes/purchasable_offer/0 (Object)
          audience: ALL (String)
          currency: USD (String)
          marketplace_id: ATVPDKIKX0DER (String)
          our_price (Array)
          /attributes/purchasable_offer/0/our_price (Array)
            /attributes/purchasable_offer/0/our_price/0 (Object)
              schedule (Array)
              /attributes/purchasable_offer/0/our_price/0/schedule (Array)
                /attributes/purchasable_offer/0/our_price/0/schedule/0 (Object)
                  value_with_tax: 14.95 (Number)
          /attributes/purchasable_offer/0/start_at (Object)
            value: 2025-03-17T06:22:07.258Z (String)
```
if you want to find all possible attributes and dont care for your current values, use `definition get` with the `--detailed` flag and pipe the output to a file
```bash
halycon definition get --type CUTE-OUTFITS -vvv --detailed > log.txt
```
which will give you a loooooong long output.

example:
```
  purchasable_offer:
    type: "array"
    description: "The attribute indicates the Purchasable Offer of the product"
    title: "Purchasable Offer"
    Items:
        type: "object"
        Properties:
          map_price:
            type: "array"
            description: "The attribute indicates the Purchasable Offer Map Price of the product"
            title: "Purchasable Offer Map Price"
            Items:
                type: "object"
                Properties:
                  schedule:
                    type: "array"
                    description: "The attribute indicates the Purchasable Offer Map Price Schedule of the product"
                    title: "Purchasable Offer Map Price Schedule"
                    Items:
                        type: "object"
                        Properties:
                          value_with_tax:
                            type: "number"
                            description: "Provide the minimum advertised price"
                            title: "Minimum Advertised Price"
```
here, path will be
```
/attributes/purchasable_offer/0/map_price/0/schedule/0/value_with_tax
 ^ editing  ^ offer details   ^ first offer
```
see? trust me, it is easy! just needs a bit of manual labour and parsing with your shiny, pretty eyes.

after your patch file is ready, just send it over here
```bash
halycon listings patch --sku WOULD-I-LOOK-CUTE-IN-MAID-OUTFIT -i patch.json -v
```
then
```bash
halycon listings get --sku WOULD-I-LOOK-CUTE-IN-MAID-OUTFIT -v --attributes
```

you can provide multiple patches for same attribute.

as I understand, there are no maximum limit for patches, so, you can send all the patches for a listing in one file.

for deleting, value array must be the same with the one currently registered. example:
```json
{
  "productType": "WALLET",
  "patches": [
    {
      "op": "add",
      "path": "/attributes/fulfillment_availability",
      "value": [
        {
          "fulfillment_channel_code": "AMAZON_NA",
          "quantity": 0
        }
      ]
    }
  ]
}
```
if you have added a fulfillment channel_code like this, instead of doing this
```json
{
  "productType": "WALLET",
  "patches": [
    {
      "op": "delete",
      "path": "/attributes/fulfillment_availability"
    }
  ]
}
```
do this
```json
{
  "productType": "WALLET",
  "patches": [
    {
      "op": "delete",
      "path": "/attributes/fulfillment_availability",
      "value": [
        {
          "fulfillment_channel_code": "AMAZON_NA",
          "quantity": 0
        }
      ]
    }
  ]
}
```
again, see
```bash
halycon listings get --sku WOULD-I-LOOK-CUTE-IN-MAID-OUTFIT -v --attributes
```
for current attribute values.
## halycon?

one of ma favourite mono song https://www.youtube.com/watch?v=2_OYaI37bi0
### Image-Text to Text AI INference
#### why
This is not related with SP-API, but I need it for generating details, title, bullet points, proofreading, etc.
#### how
load prompt from file
```bash
halycon generate --prompt-file  prompt.txt --input 'https://example.com/image.jpeg' -vvv
```
prompt from command
```bash
halycon generate --prompt  "would i look cute in maid outfit?" --input 'https://i.imgur.com/XXXXXXX.png"' -vvv
```
outputs
```
Wearing a maid outfit can be a fun and playful way to express yourself, but it's ultimately up to personal preference and how confident you feel in the outfit. If you're comfortable and excited to wear it, then go for it and have fun with it. If not, there are many other ways to express your personality and style.
```
groq API key is required, see config.

refer to `--help` for default models, local files, etc.
