import fs from 'fs'
import path from 'path'
import zlib from 'zlib'

// Generate a valid PNG buffer using Node.js built-in zlib
function createPng(width, height, drawFn) {
  // RGBA buffer
  const rgba = Buffer.alloc(width * height * 4, 0)
  drawFn(rgba, width, height)

  // Filter type 0 (None) prefixed to each scanline
  const scanlineLength = width * 4 + 1
  const rawData = Buffer.alloc(height * scanlineLength)

  for (let y = 0; y < height; y++) {
    rawData[y * scanlineLength] = 0 // Filter type 0
    for (let x = 0; x < width; x++) {
      const srcIdx = (y * width + x) * 4
      const dstIdx = y * scanlineLength + 1 + x * 4
      rawData[dstIdx] = rgba[srcIdx]
      rawData[dstIdx + 1] = rgba[srcIdx + 1]
      rawData[dstIdx + 2] = rgba[srcIdx + 2]
      rawData[dstIdx + 3] = rgba[srcIdx + 3]
    }
  }

  const deflated = zlib.deflateSync(rawData)

  // PNG Signature
  const signature = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10])

  // IHDR Chunk
  const ihdr = Buffer.alloc(13)
  ihdr.writeUInt32BE(width, 0)
  ihdr.writeUInt32BE(height, 4)
  ihdr[8] = 8 // bit depth
  ihdr[9] = 6 // color type RGBA
  ihdr[10] = 0 // compression
  ihdr[11] = 0 // filter
  ihdr[12] = 0 // interlace

  const ihdrChunk = createChunk('IHDR', ihdr)
  const idatChunk = createChunk('IDAT', deflated)
  const iendChunk = createChunk('IEND', Buffer.alloc(0))

  return Buffer.concat([signature, ihdrChunk, idatChunk, iendChunk])
}

function createChunk(type, data) {
  const length = data.length
  const chunk = Buffer.alloc(12 + length)
  chunk.writeUInt32BE(length, 0)
  chunk.write(type, 4, 4, 'ascii')
  data.copy(chunk, 8)

  const crc = calculateCrc32(chunk.subarray(4, 8 + length))
  chunk.writeUInt32BE(crc, 8 + length)
  return chunk
}

// CRC32 table
const crcTable = new Uint32Array(256)
for (let n = 0; n < 256; n++) {
  let c = n
  for (let k = 0; k < 8; k++) {
    c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1
  }
  crcTable[n] = c
}

function calculateCrc32(buf) {
  let crc = 0xffffffff
  for (let i = 0; i < buf.length; i++) {
    crc = crcTable[(crc ^ buf[i]) & 0xff] ^ (crc >>> 8)
  }
  return (crc ^ 0xffffffff) >>> 0
}

function setPixel(buf, width, x, y, r, g, b, a) {
  if (x < 0 || x >= width || y < 0) return
  const idx = (Math.round(y) * width + Math.round(x)) * 4
  if (idx < 0 || idx >= buf.length - 3) return
  buf[idx] = r
  buf[idx + 1] = g
  buf[idx + 2] = b
  buf[idx + 3] = a
}

function drawFilledRect(buf, width, x0, y0, x1, y1, r, g, b, a) {
  for (let y = y0; y <= y1; y++) {
    for (let x = x0; x <= x1; x++) {
      setPixel(buf, width, x, y, r, g, b, a)
    }
  }
}

function drawFilledCircle(buf, width, cx, cy, radius, r, g, b, a) {
  for (let y = cy - radius; y <= cy + radius; y++) {
    for (let x = cx - radius; x <= cx + radius; x++) {
      const dist = Math.sqrt((x - cx) ** 2 + (y - cy) ** 2)
      if (dist <= radius) {
        setPixel(buf, width, x, y, r, g, b, a)
      }
    }
  }
}

// 81x81 icons for TabBar
const SIZE = 81

const ICONS = [
  {
    name: 'home',
    color: [115, 122, 135, 255], // #737a87
    activeColor: [49, 94, 246, 255], // #315ef6
    draw: (buf, w, h, [r, g, b, a]) => {
      // Roof triangle
      const midX = 40
      for (let y = 16; y <= 38; y++) {
        const span = (y - 16) * 1.1
        drawFilledRect(buf, w, Math.floor(midX - span), y, Math.ceil(midX + span), y, r, g, b, a)
      }
      // House body
      drawFilledRect(buf, w, 20, 38, 60, 64, r, g, b, a)
      // Door cutout
      drawFilledRect(buf, w, 34, 46, 46, 64, 255, 255, 255, 0)
    }
  },
  {
    name: 'key',
    color: [115, 122, 135, 255],
    activeColor: [49, 94, 246, 255],
    draw: (buf, w, h, [r, g, b, a]) => {
      // Key head circle
      drawFilledCircle(buf, w, 30, 36, 16, r, g, b, a)
      drawFilledCircle(buf, w, 30, 36, 8, 255, 255, 255, 0) // Hole
      // Key stem
      drawFilledRect(buf, w, 44, 32, 66, 40, r, g, b, a)
      // Teeth
      drawFilledRect(buf, w, 56, 40, 62, 50, r, g, b, a)
      drawFilledRect(buf, w, 63, 40, 66, 46, r, g, b, a)
    }
  },
  {
    name: 'record',
    color: [115, 122, 135, 255],
    activeColor: [49, 94, 246, 255],
    draw: (buf, w, h, [r, g, b, a]) => {
      // Clipboard body
      drawFilledRect(buf, w, 20, 22, 60, 66, r, g, b, a)
      // Inner sheet cutout
      drawFilledRect(buf, w, 26, 28, 54, 60, 255, 255, 255, 220)
      // Clip on top
      drawFilledRect(buf, w, 32, 14, 48, 24, r, g, b, a)
      // Text lines on sheet
      drawFilledRect(buf, w, 30, 36, 50, 39, r, g, b, a)
      drawFilledRect(buf, w, 30, 44, 46, 47, r, g, b, a)
      drawFilledRect(buf, w, 30, 52, 42, 55, r, g, b, a)
    }
  },
  {
    name: 'profile',
    color: [115, 122, 135, 255],
    activeColor: [49, 94, 246, 255],
    draw: (buf, w, h, [r, g, b, a]) => {
      // Head
      drawFilledCircle(buf, w, 40, 28, 14, r, g, b, a)
      // Body / shoulders
      drawFilledCircle(buf, w, 40, 68, 26, r, g, b, a)
      // Clear outside bottom half
      for (let y = 65; y <= 80; y++) {
        for (let x = 0; x < w; x++) {
          if (x < 16 || x > 64) {
            setPixel(buf, w, x, y, 0, 0, 0, 0)
          }
        }
      }
    }
  }
]

const outDir = path.resolve('miniprogram/assets/icons/tabbar')
fs.mkdirSync(outDir, { recursive: true })

for (const icon of ICONS) {
  // Inactive
  const pngBuf = createPng(SIZE, SIZE, (buf, w, h) => icon.draw(buf, w, h, icon.color))
  fs.writeFileSync(path.join(outDir, `${icon.name}.png`), pngBuf)

  // Active
  const pngActiveBuf = createPng(SIZE, SIZE, (buf, w, h) => icon.draw(buf, w, h, icon.activeColor))
  fs.writeFileSync(path.join(outDir, `${icon.name}-active.png`), pngActiveBuf)

  console.log(`Generated tabbar icon: ${icon.name}.png and ${icon.name}-active.png`)
}
