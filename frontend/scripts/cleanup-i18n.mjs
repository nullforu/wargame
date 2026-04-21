import { readdirSync, readFileSync, writeFileSync } from 'node:fs'
import path from 'node:path'

const ROOT = process.cwd()
const SRC_DIR = path.join(ROOT, 'src')
const LOCALES_DIR = path.join(SRC_DIR, 'locales')
const VALID_EXTENSIONS = new Set(['.ts', '.tsx', '.js', '.jsx'])

const args = new Set(process.argv.slice(2))
const shouldWrite = args.has('--write')

const walkFiles = (dir) => {
    const entries = readdirSync(dir, { withFileTypes: true })
    const files = []

    for (const entry of entries) {
        if (entry.name === 'node_modules' || entry.name === 'dist' || entry.name.startsWith('.')) {
            continue
        }

        const fullPath = path.join(dir, entry.name)

        if (entry.isDirectory()) {
            files.push(...walkFiles(fullPath))
            continue
        }

        if (VALID_EXTENSIONS.has(path.extname(entry.name))) {
            files.push(fullPath)
        }
    }

    return files
}

const collectUsedKeys = () => {
    const files = walkFiles(SRC_DIR)
    const used = new Set()
    const patterns = [/\b(?:t|translate)\(\s*'([^']+)'/g, /\b(?:t|translate)\(\s*"([^"]+)"/g, /\b(?:t|translate)\(\s*`([^`]+)`/g, /'((?:categories|roles)\.[^']+)'/g, /"((?:categories|roles)\.[^"]+)"/g]

    for (const filePath of files) {
        const content = readFileSync(filePath, 'utf8')
        for (const pattern of patterns) {
            let match = pattern.exec(content)
            while (match) {
                used.add(match[1])
                match = pattern.exec(content)
            }
            pattern.lastIndex = 0
        }
    }

    return used
}

const collectLocaleFiles = () => {
    return readdirSync(LOCALES_DIR)
        .filter((name) => name.endsWith('.json'))
        .map((name) => path.join(LOCALES_DIR, name))
}

const usedKeys = collectUsedKeys()
const localeFiles = collectLocaleFiles()

let totalUnused = 0

for (const localeFile of localeFiles) {
    const raw = readFileSync(localeFile, 'utf8')
    const locale = JSON.parse(raw)
    const entries = Object.entries(locale)
    const unusedEntries = entries.filter(([key]) => !usedKeys.has(key))
    const usedEntries = entries.filter(([key]) => usedKeys.has(key))

    totalUnused += unusedEntries.length

    const relative = path.relative(ROOT, localeFile)
    console.log(`\\n${relative}`)
    console.log(`used: ${usedEntries.length}, unused: ${unusedEntries.length}`)

    if (unusedEntries.length > 0) {
        for (const [key] of unusedEntries) {
            console.log(`- ${key}`)
        }
    }

    if (shouldWrite) {
        const nextLocale = Object.fromEntries(usedEntries)
        writeFileSync(localeFile, `${JSON.stringify(nextLocale, null, 4)}\n`, 'utf8')
    }
}

console.log(`\\nTotal unused keys: ${totalUnused}`)

if (shouldWrite) {
    console.log('Applied cleanup with --write')
} else {
    console.log('Dry run complete (use --write to apply)')
}
