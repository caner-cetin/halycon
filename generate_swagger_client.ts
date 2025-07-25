#!/usr/bin/env bun
import { $ } from "bun";
import { mkdir } from "fs/promises";

const REMOTE_MODEL_REPO = "https://raw.githubusercontent.com/amzn/selling-partner-api-models/refs/heads/main/models";

interface ApiModel {
  json: string;
  remotePath: string;
  packageFolder: string;
}

const API_MODELS: ApiModel[] = [
  {
    json: "fulfillmentInbound_2024-03-20.json",
    remotePath: "fulfillment-inbound-api-model/fulfillmentInbound_2024-03-20.json",
    packageFolder: "internal/amazon/fba_inbound"
  },
  {
    json: "fbaInventory.json",
    remotePath: "fba-inventory-api-model/fbaInventory.json",
    packageFolder: "internal/amazon/fba_inventory"
  },
  {
    json: "catalogItems_2022-04-01.json",
    remotePath: "catalog-items-api-model/catalogItems_2022-04-01.json",
    packageFolder: "internal/amazon/catalog"
  },
  {
    json: "listingsItems_2021-08-01.json",
    remotePath: "listings-items-api-model/listingsItems_2021-08-01.json",
    packageFolder: "internal/amazon/listings"
  },
  {
    json: "definitionsProductTypes_2020-09-01.json",
    remotePath: "product-type-definitions-api-model/definitionsProductTypes_2020-09-01.json",
    packageFolder: "internal/amazon/product_type_definitions"
  },
  {
    json: "feeds_2021-06-30.json",
    remotePath: "feeds-api-model/feeds_2021-06-30.json",
    packageFolder: "internal/amazon/feeds"
  }
];

async function setupDirectories() {
  const dirs = [
    "internal/amazon/catalog",
    "internal/amazon/fba_inbound", 
    "internal/amazon/fba_inventory",
    "internal/amazon/listings",
    "internal/amazon/product_type_definitions",
    "internal/amazon/feeds",
    "models"
  ];

  for (const dir of dirs) {
    await mkdir(dir, { recursive: true });
  }
}

async function installOapiCodegen() {
  await $`go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest`;
}

async function downloadWithTimeout(url: string, outputPath: string, timeoutMs: number = 60000): Promise<void> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const response = await fetch(url, { signal: controller.signal });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }
    
    const content = await response.text();
    await Bun.write(outputPath, content);
  } finally {
    clearTimeout(timeoutId);
  }
}

async function convertToYaml(specUrl: string, outputPath: string, timeoutMs: number = 120000): Promise<void> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const convertUrl = `https://converter.swagger.io/api/convert?url=${encodeURIComponent(specUrl)}`;
    const response = await fetch(convertUrl, {
      signal: controller.signal,
      headers: { "Accept": "application/yaml" }
    });

    if (!response.ok) {
      throw new Error(`Conversion failed: HTTP ${response.status}`);
    }

    const yamlContent = await response.text();
    if (!yamlContent.trim()) {
      throw new Error("Conversion produced empty content");
    }

    await Bun.write(outputPath, yamlContent);
  } finally {
    clearTimeout(timeoutId);
  }
}

async function convertToYamlFallback(localPath: string, outputPath: string, timeoutMs: number = 120000): Promise<void> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const jsonContent = await Bun.file(localPath).text();
    const response = await fetch("https://converter.swagger.io/api/convert", {
      method: "POST",
      signal: controller.signal,
      headers: {
        "Accept": "application/yaml",
        "Content-Type": "application/json"
      },
      body: jsonContent
    });

    if (!response.ok) {
      throw new Error(`Fallback conversion failed: HTTP ${response.status}`);
    }

    const yamlContent = await response.text();
    await Bun.write(outputPath, yamlContent);
  } finally {
    clearTimeout(timeoutId);
  }
}

async function generateClient(packageName: string, specPath: string, outputPath: string): Promise<void> {
  const oapiPath = `${process.env.HOME}/go/bin/oapi-codegen`;
  
  try {
    await $`${oapiPath} -package ${packageName} -generate types,client,spec -o ${outputPath} -response-type-suffix Resp ${specPath}`;
  } catch (error) {
    const jsonPath = specPath.replace('.yaml', '.json').replace('halycon_sp_api_', 'halycon_sp_api_');
    try {
      await $`${oapiPath} -package ${packageName} -generate types,client,spec -o ${outputPath} ${jsonPath}`;
    } catch (fallbackError) {
      throw new Error(`Could not generate client for ${packageName} even from original JSON: ${fallbackError}`);
    }
  }
}

async function processModel(model: ApiModel): Promise<void> {
  const localModelPath = `models/halycon_sp_api_${model.json}`;
  const convertedModelPath = `models/halycon_sp_api_${model.json.replace('.json', '.yaml')}`;
  const specUrl = `${REMOTE_MODEL_REPO}/${model.remotePath}`;
  const packageName = model.packageFolder.split('/').pop()!;

  try {
    await downloadWithTimeout(specUrl, localModelPath);

    try {
      await convertToYaml(specUrl, convertedModelPath);
    } catch (conversionError) {
      await convertToYamlFallback(localModelPath, convertedModelPath);
    }

    const convertedFile = Bun.file(convertedModelPath);
    if (!(await convertedFile.exists()) || convertedFile.size === 0) {
      throw new Error("Conversion failed or produced empty file");
    }

    await generateClient(packageName, convertedModelPath, `${model.packageFolder}/client.go`);
  } catch (error) {
    console.error(`Error processing ${model.json}:`, error);
    throw error;
  }
}

async function main() {
  try {
    await setupDirectories();
    await installOapiCodegen();
    for (const model of API_MODELS) {
      await processModel(model);
    }
    await $`go mod tidy`;
    console.log("All Swagger clients generated successfully.");
  } catch (error) {
    console.error("Script failed:", error);
    process.exit(1);
  }
}

main();