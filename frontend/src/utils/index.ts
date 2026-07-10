import type { storage } from '@/wailsjs/go/models'
import * as libsysi from '@/wailsjs/go/sysinfo/SysInfo'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import * as semver from 'semver'

export async function latestRelease(currentVersion: string, binaryType?: string) {
  return fetch('https://api.github.com/repos/markmybytes/install-it/releases/latest')
    .then(response => response.json())
    .then(async body => {
      const version = semver.clean(body.tag_name) || '0.0.0'

      let assetUrl = ''
      let webviewAssetUrl: string | undefined
      if (binaryType && Array.isArray(body.assets)) {
        for (const asset of body.assets) {
          if (asset.name === `install-it.${binaryType}.zip`) {
            assetUrl = asset.browser_download_url as string
          } else if (asset.name === `install-it.${binaryType}.webview.zip`) {
            webviewAssetUrl = asset.browser_download_url as string
          }
        }
      }

      return {
        hasUpdate: semver.gt(version, currentVersion),
        name: body.name as string,
        releaseAt: new Date(Date.parse(body.published_at)),
        releaseNotes: DOMPurify.sanitize(await marked.parse(body.body)),
        tag: body.tag_name as string,
        url: body.html_url as string,
        version: version,
        assetUrl,
        webviewAssetUrl
      }
    })
}

/**
 * Retrieves detailed hardware information from the system.
 */
export async function getHardware() {
  return libsysi.ResolvedHardware()
}

/**
 * Tests whether the given input string satisfies the specified match rule.
 */
export function testMatchRule(rule: storage.Rule, input: string) {
  input = rule.is_case_sensitive ? input : input.toLowerCase()
  const values = rule.is_case_sensitive ? rule.values : rule.values.map(v => v.toLowerCase())
  const hits = values.map((v: string): boolean => {
    switch (rule.operator) {
      case 'contain':
        return input.includes(v)
      case 'notContain':
        return !input.includes(v)
      case 'equal':
        return input === v
      case 'notEqual':
        return input !== v
      case 'regex': {
        try {
          return new RegExp(v, rule.is_case_sensitive ? '' : 'i').test(input)
        } catch {
          return false
        }
      }
      default:
        return false
    }
  })

  return rule.should_hit_all ? hits.every(Boolean) : hits.some(Boolean)
}
