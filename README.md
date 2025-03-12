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
  - [halycon?](#halycon-1)

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
halycon definition search --keywords wallet
```
```
+--------------+-------------+
| DISPLAY NAME | AMAZON NAME |
+--------------+-------------+
| Wallet       | WALLET      |
+--------------+-------------+
```
### Get Product Type Definitions
#### why
attributes are required for [`PUT /listings/2021-08-01/items/{sellerId}/{sku}`](https://developer-docs.amazon.com/sp-api/lang-tr_TR/docs/listings-items-api-v2021-08-01-reference#listingsitemputrequest)
#### how
```bash
halycon definition get --type WALLET -v
```

<details>
  <summary> Result </summary>

  ```
    * "brand" - Max. 50 characters
      * Examples
        * "Ralph Lauren; North Face; Patagonia"
      * Type - array
    * "bullet_point" - Max. 100 characters per line. Use these to highlight some of the product's most important qualities. Each line will be displayed as a separate bullet point above the product description.
      * Examples
        * "Delicious honey-apricot glaze"
      * Type - array
    * "country_of_origin" - The country in which the product was published.
      * Examples
      * Type - array
      * Enums:
        * "AF" - "Afghanistan"
        * "AX" - "Aland Islands"
        * "AL" - "Albania"
        * "DZ" - "Algeria"
        * "AS" - "American Samoa"
        * "AD" - "Andorra"
        * "AO" - "Angola"
        * "AI" - "Anguilla"
        * "AQ" - "Antarctica"
        * "AG" - "Antigua and Barbuda"
        * "AR" - "Argentina"
        * "AM" - "Armenia"
        * "AW" - "Aruba"
        * "AC" - "Ascension Island"
        * "AU" - "Australia"
        * "AT" - "Austria"
        * "AZ" - "Azerbaijan"
        * "BH" - "Bahrain"
        * "BD" - "Bangladesh"
        * "BB" - "Barbados"
        * "BY" - "Belarus"
        * "BE" - "Belgium"
        * "BZ" - "Belize"
        * "BJ" - "Benin"
        * "BM" - "Bermuda"
        * "BT" - "Bhutan"
        * "BO" - "Bolivia"
        * "BQ" - "Bonaire, Saint Eustatius and Saba"
        * "BA" - "Bosnia and Herzegovina"
        * "BW" - "Botswana"
        * "BV" - "Bouvet Island"
        * "BR" - "Brazil"
        * "IO" - "British Indian Ocean Territory"
        * "VG" - "British Virgin Islands"
        * "BN" - "Brunei Darussalam"
        * "BG" - "Bulgaria"
        * "BF" - "Burkina Faso"
        * "BI" - "Burundi"
        * "KH" - "Cambodia"
        * "CM" - "Cameroon"
        * "CA" - "Canada"
        * "IC" - "Canary Islands"
        * "CV" - "Cape Verde"
        * "KY" - "Cayman Islands"
        * "CF" - "Central African Republic"
        * "TD" - "Chad"
        * "CL" - "Chile"
        * "CN" - "China"
        * "CX" - "Christmas Island"
        * "CC" - "Cocos (Keeling) Islands"
        * "CO" - "Colombia"
        * "KM" - "Comoros"
        * "CG" - "Congo"
        * "CK" - "Cook Islands"
        * "CR" - "Costa Rica"
        * "HR" - "Croatia"
        * "CU" - "Cuba"
        * "CW" - "Curaçao"
        * "CY" - "Cyprus"
        * "CZ" - "Czech Republic"
        * "KP" - "Democratic People's Republic of Korea"
        * "DK" - "Denmark"
        * "DJ" - "Djibouti"
        * "DM" - "Dominica"
        * "DO" - "Dominican Republic"
        * "TP" - "East Timor"
        * "EC" - "Ecuador"
        * "EG" - "Egypt"
        * "SV" - "El Salvador"
        * "GQ" - "Equatorial Guinea"
        * "ER" - "Eritrea"
        * "EE" - "Estonia"
        * "ET" - "Ethiopia"
        * "FK" - "Falkland Islands (Malvinas)"
        * "FO" - "Faroe Islands"
        * "FM" - "Federated States of Micronesia"
        * "FJ" - "Fiji"
        * "FI" - "Finland"
        * "FR" - "France"
        * "GF" - "French Guiana"
        * "PF" - "French Polynesia"
        * "TF" - "French Southern Territories"
        * "GA" - "Gabon"
        * "GE" - "Georgia"
        * "DE" - "Germany"
        * "GH" - "Ghana"
        * "GI" - "Gibraltar"
        * "GB" - "Great Britain"
        * "GR" - "Greece"
        * "GL" - "Greenland"
        * "GD" - "Grenada"
        * "GP" - "Guadeloupe"
        * "GU" - "Guam"
        * "GT" - "Guatemala"
        * "GG" - "Guernsey"
        * "GN" - "Guinea"
        * "GW" - "Guinea-Bissau"
        * "GY" - "Guyana"
        * "HT" - "Haiti"
        * "HM" - "Heard Island and the McDonald Islands"
        * "VA" - "Holy See"
        * "HN" - "Honduras"
        * "HK" - "Hong Kong"
        * "HU" - "Hungary"
        * "IS" - "Iceland"
        * "IN" - "India"
        * "ID" - "Indonesia"
        * "IR" - "Iran"
        * "IE" - "Ireland"
        * "IQ" - "Islamic Republic of Iraq"
        * "IM" - "Isle of Man"
        * "IL" - "Israel"
        * "IT" - "Italy"
        * "CI" - "Ivory Coast"
        * "JM" - "Jamaica"
        * "JP" - "Japan"
        * "JE" - "Jersey"
        * "JO" - "Jordan"
        * "KZ" - "Kazakhstan"
        * "KE" - "Kenya"
        * "KI" - "Kiribati"
        * "KW" - "Kuwait"
        * "KG" - "Kyrgyzstan"
        * "LA" - "Lao People's Democratic Republic"
        * "LV" - "Latvia"
        * "LB" - "Lebanon"
        * "LS" - "Lesotho"
        * "LR" - "Liberia"
        * "LY" - "Libya"
        * "LI" - "Liechtenstein"
        * "LT" - "Lithuania"
        * "LU" - "Luxembourg"
        * "MO" - "Macao"
        * "MK" - "Macedonia"
        * "MG" - "Madagascar"
        * "MW" - "Malawi"
        * "MY" - "Malaysia"
        * "MV" - "Maldives"
        * "ML" - "Mali"
        * "MT" - "Malta"
        * "MH" - "Marshall Islands"
        * "MQ" - "Martinique"
        * "MR" - "Mauritania"
        * "MU" - "Mauritius"
        * "YT" - "Mayotte"
        * "MX" - "Mexico"
        * "MC" - "Monaco"
        * "MN" - "Mongolia"
        * "ME" - "Montenegro"
        * "MS" - "Montserrat"
        * "MA" - "Morocco"
        * "MZ" - "Mozambique"
        * "MM" - "Myanmar"
        * "NA" - "Namibia"
        * "NR" - "Nauru"
        * "NP" - "Nepal"
        * "NL" - "Netherlands"
        * "AN" - "Netherlands Antilles"
        * "NC" - "New Caledonia"
        * "NZ" - "New Zealand"
        * "NI" - "Nicaragua"
        * "NE" - "Niger"
        * "NG" - "Nigeria"
        * "NU" - "Niue"
        * "NF" - "Norfolk Island"
        * "MP" - "Northern Mariana Islands"
        * "NO" - "Norway"
        * "OM" - "Oman"
        * "PK" - "Pakistan"
        * "PW" - "Palau"
        * "PS" - "Palestinian Territories"
        * "PA" - "Panama"
        * "PG" - "Papua New Guinea"
        * "PY" - "Paraguay"
        * "PE" - "Peru"
        * "PH" - "Philippines"
        * "PN" - "Pitcairn"
        * "PL" - "Poland"
        * "PT" - "Portugal"
        * "PR" - "Puerto Rico"
        * "QA" - "Qatar"
        * "KR" - "Republic of Korea"
        * "MD" - "Republic of Moldova"
        * "RE" - "Reunion"
        * "RO" - "Romania"
        * "RU" - "Russian Federation"
        * "RW" - "Rwanda"
        * "BL" - "Saint Barthelemy"
        * "SH" - "Saint Helena, Ascension and Tristan da Cunha"
        * "KN" - "Saint Kitts and Nevis"
        * "LC" - "Saint Lucia"
        * "MF" - "Saint Martin"
        * "PM" - "Saint Pierre and Miquelon"
        * "VC" - "Saint Vincent and the Grenadines"
        * "WS" - "Samoa"
        * "SM" - "San Marino"
        * "ST" - "Sao Tome and Principe"
        * "SA" - "Saudi Arabia"
        * "SN" - "Senegal"
        * "RS" - "Serbia"
        * "CS" - "Serbia and Montenegro"
        * "SC" - "Seychelles"
        * "SL" - "Sierra Leone"
        * "SG" - "Singapore"
        * "SX" - "Sint Maarten"
        * "SK" - "Slovakia"
        * "SI" - "Slovenia"
        * "SB" - "Solomon Islands"
        * "SO" - "Somalia"
        * "ZA" - "South Africa"
        * "GS" - "South Georgia and the South Sandwich Islands"
        * "SS" - "South Sudan"
        * "ES" - "Spain"
        * "LK" - "Sri Lanka"
        * "SD" - "Sudan"
        * "SR" - "Suriname"
        * "SJ" - "Svalbard and Jan Mayen"
        * "SZ" - "Swaziland"
        * "SE" - "Sweden"
        * "CH" - "Switzerland"
        * "SY" - "Syria"
        * "TW" - "Taiwan"
        * "TJ" - "Tajikistan"
        * "TH" - "Thailand"
        * "BS" - "The Bahamas"
        * "CD" - "The Democratic Republic of the Congo"
        * "GM" - "The Gambia"
        * "TL" - "Timor-Leste"
        * "TG" - "Togo"
        * "TK" - "Tokelau"
        * "TO" - "Tonga"
        * "TT" - "Trinidad and Tobago"
        * "TA" - "Tristan da Cunha"
        * "TN" - "Tunisia"
        * "TR" - "Turkey"
        * "TM" - "Turkmenistan"
        * "TC" - "Turks and Caicos Islands"
        * "TV" - "Tuvalu"
        * "VI" - "U.S. Virgin Islands"
        * "UG" - "Uganda"
        * "UA" - "Ukraine"
        * "AE" - "United Arab Emirates"
        * "UK" - "United Kingdom"
        * "TZ" - "United Republic of Tanzania"
        * "US" - "United States"
        * "UM" - "United States Minor Outlying Islands"
        * "unknown" - "Unknown"
        * "UY" - "Uruguay"
        * "UZ" - "Uzbekistan"
        * "VU" - "Vanuatu"
        * "VE" - "Venezuela"
        * "VN" - "Vietnam"
        * "WF" - "Wallis and Futuna"
        * "WD" - "WD"
        * "EH" - "Western Sahara"
        * "WZ" - "WZ"
        * "XB" - "XB"
        * "XC" - "XC"
        * "XE" - "XE"
        * "XK" - "XK"
        * "XM" - "XM"
        * "XN" - "XN"
        * "XY" - "XY"
        * "YE" - "Yemen"
        * "YU" - "Yugoslavia"
        * "ZR" - "Zaire"
        * "ZM" - "Zambia"
        * "ZW" - "Zimbabwe"
    * "item_name" - Provide a title for the item that may be customer facing
      * Examples
        * "Adidas Blue Sneakers"
      * Type - array
    * "item_type_keyword" - Item type keywords are used to place new ASINs in the appropriate place(s) within the graph. Select the most specific accurate term for optimal placement.
      * Examples
        * "Carry on luggage"
      * Type - array
    * "product_description" - The description you provide should pertain to the product in general, not your particular item. There is a 2,000 character maximum.
      * Examples
        * "This ham has been smoked for 12 hours..."
      * Type - array
    * "supplier_declared_dg_hz_regulation" - If the product is a Dangerous Good or Hazardous Material, Substance or Waste that is regulated for transportation, storage, and/or waste select from the list of valid values
      * Examples
        * "GHS, Storage, Transportation"
      * Type - array
      * Enums:
        * "ghs" - "GHS"
        * "not_applicable" - "Not Applicable"
        * "other" - "Other"
        * "storage" - "Storage"
        * "transportation" - "Transportation"
        * "unknown" - "Unknown"
        * "waste" - "Waste"
  ```

</details>



## halycon?

one of ma favourite mono song https://www.youtube.com/watch?v=2_OYaI37bi0