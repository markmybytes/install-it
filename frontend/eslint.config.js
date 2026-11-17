import skipFormatting from '@vue/eslint-config-prettier/skip-formatting'
import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'
import pluginVue from 'eslint-plugin-vue'

export default defineConfigWithVueTs(
  {
    name: 'app/files-to-ignore',
    ignores: ['**/dist/**', '**/dist-ssr/**', '**/coverage/**', '**/wailsjs/**']
  },
  pluginVue.configs['flat/recommended'],
  vueTsConfigs.recommended,
  {
    rules: {
      'vue/multi-word-component-names': ['off'],
      'vue/block-lang': ['error', { script: { lang: 'ts' } }],
      'vue/block-order': ['error', { order: ['script', 'template', 'style'] }],
      'vue/component-api-style': ['error', ['script-setup']],
      'vue/prefer-define-options': ['error'],
      'vue/prefer-use-template-ref': ['error'],
      'vue/define-props-declaration': ['error', 'type-based'],
      'vue/define-emits-declaration': ['error', 'type-based'],
      'vue/define-macros-order': [
        'error',
        { order: ['defineProps', 'defineEmits', 'defineModel', 'defineExpose'] }
      ],
      'vue/require-typed-ref': ['error'],
      'vue/no-required-prop-with-default': ['warn', { autofix: false }],
      'vue/html-comment-content-spacing': ['error', 'always'],
      // 'vue/html-button-has-type': [
      //   'warn',
      //   {
      //     button: true,
      //     submit: true,
      //     reset: true
      //   }
      // ],
      'vue/padding-line-between-tags': [
        'error',
        [
          {
            blankLine: 'always',
            prev: '*',
            next: '*'
          }
        ]
      ]
    }
  },
  {
    files: ['src/__tests__/**/*.ts'],
    languageOptions: {
      parserOptions: {
        projectService: false
      }
    }
  },
  {
    files: ['src/**/*.{ts,vue}', '!src/__tests__/**'],
    rules: {
      '@typescript-eslint/naming-convention': [
        'error',
        { selector: 'function', format: ['camelCase'] },
        { selector: 'typeLike', format: ['PascalCase'] }
      ]
    }
  },
  skipFormatting
)
