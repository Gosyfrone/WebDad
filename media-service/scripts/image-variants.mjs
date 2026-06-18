import { mkdir, writeFile } from 'node:fs/promises'
import path from 'node:path'
import sharp from 'sharp'

const [, , inputPath, outputDir] = process.argv

if (!inputPath || !outputDir) {
  console.error('usage: node scripts/image-variants.mjs <input> <output-dir>')
  process.exit(2)
}

const presets = [
  { name: 'thumb', maxSide: 320, quality: 80 },
  { name: 'small', maxSide: 640, quality: 82 },
  { name: 'medium', maxSide: 1280, quality: 84 },
  { name: 'large', maxSide: 2048, quality: 86 },
]

await mkdir(outputDir, { recursive: true })

const source = sharp(inputPath, { animated: false, limitInputPixels: 80_000_000 }).rotate()
const metadata = await source.metadata()

if (!metadata.width || !metadata.height) {
  throw new Error('image dimensions unavailable')
}

const hasAlpha = Boolean(metadata.hasAlpha)
const variants = []
const seen = new Set()

for (const preset of presets) {
  const longest = Math.max(metadata.width, metadata.height)
  const scale = longest > preset.maxSide ? preset.maxSide / longest : 1
  const width = Math.max(1, Math.round(metadata.width * scale))
  const height = Math.max(1, Math.round(metadata.height * scale))
  const sizeKey = `${width}x${height}`

  if (seen.has(sizeKey)) continue
  seen.add(sizeKey)

  const outPath = path.join(outputDir, `${preset.name}.webp`)
  const pipeline = sharp(inputPath, { animated: false, limitInputPixels: 80_000_000 })
    .rotate()
    .resize({
      width,
      height,
      fit: 'inside',
      withoutEnlargement: true,
      kernel: sharp.kernel.lanczos3,
    })
    .webp({
      quality: preset.quality,
      alphaQuality: hasAlpha ? 90 : 100,
      effort: 4,
      smartSubsample: true,
    })

  const info = await pipeline.toFile(outPath)
  variants.push({
    name: preset.name,
    path: outPath,
    mime: 'image/webp',
    size: info.size,
    width: info.width,
    height: info.height,
  })
}

await writeFile(
  path.join(outputDir, 'manifest.json'),
  JSON.stringify({
    width: metadata.width,
    height: metadata.height,
    variants,
  }),
)
