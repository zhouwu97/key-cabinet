#!/usr/bin/env node

/**
 * 智能钥匙柜 ESP32 硬件行为模拟器启动脚本
 * 启动已编译的 Go 硬件行为模拟器 (cmd/simulator)，支持本地全流程 MQTT 实机时序模拟。
 */

import { spawn } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const serverDir = path.resolve(__dirname, '../server')

const args = process.argv.slice(2)
console.log('🚀 正在启动智能钥匙柜硬件行为模拟器...')

const proc = spawn('go', ['run', './cmd/simulator/main.go', ...args], {
  cwd: serverDir,
  stdio: 'inherit',
  shell: true,
})

proc.on('close', (code) => {
  process.exit(code || 0)
})
