// @ts-check
import js from "@eslint/js";
import { defineConfig } from "eslint/config";
import { configs as tsConfigs } from "typescript-eslint";
import {
  configs as angularConfigs,
  processInlineTemplates,
} from "angular-eslint";

export default defineConfig(
  {
    ignores: ["src/api/**"],
  },
  {
    files: ["**/*.ts"],
    extends: [
      js.configs.recommended,
      tsConfigs.recommended,
      tsConfigs.stylistic,
      angularConfigs.tsRecommended,
    ],
    processor: processInlineTemplates,
    rules: {
      "@angular-eslint/directive-selector": [
        "error",
        {
          type: "attribute",
          prefix: "app",
          style: "camelCase",
        },
      ],
      "@angular-eslint/component-selector": [
        "error",
        {
          type: "element",
          prefix: "app",
          style: "kebab-case",
        },
      ],
    },
  },
  {
    files: ["**/*.html"],
    extends: [
      angularConfigs.templateRecommended,
      angularConfigs.templateAccessibility,
    ],
    rules: {},
  }
);
