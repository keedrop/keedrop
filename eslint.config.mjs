import {defineConfig} from "eslint/config";
import cypressPlugin from "eslint-plugin-cypress";
import globals from "globals";
import js from "@eslint/js";

export default defineConfig([{
  extends: [
    cypressPlugin.configs.recommended,
  ],
  files: ["site/scripts/*.js", "cypress/**/*.js"],

  plugins: {
    cypressPlugin,
  },

  languageOptions: {
    globals: {
      ...globals.browser,
      ...globals.mocha,
    },

    ecmaVersion: "latest",
    sourceType: "script",
  },

  rules: {
    ...js.configs.recommended.rules,
    indent: ["error", 2],
    "linebreak-style": ["error", "unix"],
    quotes: ["error", "double"],
    semi: ["error", "always"],
  },
}]);
