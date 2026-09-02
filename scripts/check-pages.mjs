import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs'
import { dirname, resolve, join } from 'node:path'
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

// 扫描所有 JSON 与 WXML
function scanFiles(dir, ext) {
  const files = []
  const list = readdirSync(dir)
  for (const item of list) {
    const full = join(dir, item)
    const stat = statSync(full)
    if (stat.isDirectory()) {
      files.push(...scanFiles(full, ext))
    } else if (item.endsWith(ext)) {
      files.push(full)
    }
  }
  return files
}

const jsonFiles = scanFiles(resolve(root, 'miniprogram'), '.json')
for (const jsonPath of jsonFiles) {
  try {
    const json = JSON.parse(readFileSync(jsonPath, 'utf8'))
    if (json.usingComponents) {
      for (const [name, compPath] of Object.entries(json.usingComponents)) {
        let resolvedWxml = ''
        if (compPath.startsWith('/')) {
          resolvedWxml = resolve(root, 'miniprogram', `.${compPath}.wxml`)
        } else {
          resolvedWxml = resolve(dirname(jsonPath), `${compPath}.wxml`)
        }
        if (!existsSync(resolvedWxml)) {
          problems.push(`组件引用文件不存在 [${jsonPath.replace(root, '')}] -> ${compPath} (${resolvedWxml})`)
        }
      }
    }
  } catch (err) {
    problems.push(`JSON 解析错误 [${jsonPath.replace(root, '')}]: ${err.message}`)
  }
}

const wxmlFiles = scanFiles(resolve(root, 'miniprogram'), '.wxml')
const fnCallRegex = /\{\{[^}]*?\b[a-zA-Z0-9_$]+\s*\([^}]*?\)[^}]*?\}\}/g
const htmlTagRegex = /<\/?(br|hr|div|span|p|table|tr|td)\b[^>]*>/gi

for (const wxmlPath of wxmlFiles) {
  const content = readFileSync(wxmlPath, 'utf8')
  const relativePath = wxmlPath.replace(root, '')

  // 1. 检查是否存在直接调用 Page 函数的非法语法
  const fnMatches = content.match(fnCallRegex)
  if (fnMatches) {
    for (const match of fnMatches) {
      problems.push(`WXML 表达式禁止直接调用函数 [${relativePath}]: ${match}`)
    }
  }

  // 2. 检查非法 HTML 标签
  const tagMatches = content.match(htmlTagRegex)
  if (tagMatches) {
    for (const match of tagMatches) {
      problems.push(`WXML 禁止使用非小程序 HTML 标签 [${relativePath}]: ${match}`)
    }
  }
}

if (problems.length > 0) {
  console.error(problems.map(p => `FAIL: ${p}`).join('\n'))
  process.exit(1)
}

console.log(`OK: ${pages.length} 个页面注册完整，${tabs.length} 个 tabBar 页面，${jsonFiles.length} 个 JSON 组件路径校验通过，${wxmlFiles.length} 个 WXML 模板语法全部通过！`)
