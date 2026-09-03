#!/usr/bin/env node

/**
 * 检查微信小程序代码中的目录导入问题
 *
 * 微信小程序的模块解析不会自动将 './services' 解析为 './services/index.ts'
 * 必须显式写成 './services/index'
 *
 * 本脚本检查所有 .ts 文件中的 import 语句，确保：
 * 1. 从 services/ 导入时必须使用 '/index'
 * 2. 从其他可能是目录的路径导入时给出警告
 */

import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const rootDir = path.resolve(__dirname, '..')

// 已知的目录（需要显式 /index 的路径）
const KNOWN_DIRECTORIES = [
  'services',
  'services/key',
  'services/reservation',
  'services/borrow',
  'services/user',
  'services/device',
  'services/operation',
]

let errors = 0
let warnings = 0

function checkFile(filePath) {
  const content = fs.readFileSync(filePath, 'utf-8')
  const lines = content.split('\n')

  lines.forEach((line, index) => {
    const lineNum = index + 1

    // 匹配 import ... from '...' 或 import ... from "..."
    const importMatch = line.match(/from\s+['"]([^'"]+)['"]/)
    if (!importMatch) return

    const importPath = importMatch[1]

    // 检查是否是相对路径
    if (!importPath.startsWith('.')) return

    // 检查是否已经包含 /index
    if (importPath.endsWith('/index')) return

    // 获取导入路径的最后一部分
    const lastPart = importPath.split('/').pop()

    // 检查是否是已知的目录导入
    const isKnownDirectory = KNOWN_DIRECTORIES.some(dir => {
      return importPath.includes(dir) && !importPath.endsWith('/index')
    })

    if (isKnownDirectory) {
      console.error(`❌ [ERROR] ${path.relative(rootDir, filePath)}:${lineNum}`)
      console.error(`   Import from directory without /index: ${importPath}`)
      console.error(`   Should be: ${importPath}/index`)
      console.error()
      errors++
    }
  })
}

function walkDirectory(dir) {
  const files = fs.readdirSync(dir, { withFileTypes: true })

  for (const file of files) {
    const fullPath = path.join(dir, file.name)

    if (file.isDirectory()) {
      // 跳过 node_modules, .git 等
      if (['node_modules', '.git', 'miniprogram_npm'].includes(file.name)) {
        continue
      }
      walkDirectory(fullPath)
    } else if (file.name.endsWith('.ts')) {
      checkFile(fullPath)
    }
  }
}

console.log('🔍 Checking directory imports in WeChat MiniProgram...\n')

walkDirectory(rootDir)

console.log('✅ Check completed!\n')
console.log(`   Errors: ${errors}`)
console.log(`   Warnings: ${warnings}`)

if (errors > 0) {
  console.error('\n❌ Found directory import issues. Please fix them.')
  process.exit(1)
}

console.log('\n✨ All imports are correct!')
