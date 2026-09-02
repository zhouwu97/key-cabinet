import { existsSync, readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const app = JSON.parse(readFileSync(resolve(root, 'miniprogram/app.json'), 'utf8'))

const problems = []
const pages = app.pages ?? []
if (pages.length === 0) problems.push('app.json pages 为空')

for (const page of pages) {
  for (const ext of ['ts', 'wxml']) {
    if (!existsSync(resolve(root, 'miniprogram', `${page}.${ext}`))) {
      problems.push(`缺少页面文件: ${page}.${ext}`)
    }
  }
}

const tabs = (app.tabBar?.list ?? []).map(item => item.pagePath)
for (const tab of tabs) {
  if (!pages.includes(tab)) problems.push(`tabBar 页面未注册进 pages: ${tab}`)
}
if (new Set(tabs).size !== tabs.length) problems.push('tabBar 页面重复')

if (problems.length > 0) {
  console.error(problems.map(p => `FAIL: ${p}`).join('\n'))
  process.exit(1)
}

console.log(`OK: ${pages.length} 个页面注册完整，${tabs.length} 个 tabBar 页面`)
