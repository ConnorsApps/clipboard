import { generatedFileName } from './format'

// iOS Safari's navigator.clipboard.writeText() runs URL-detection on the
// string: if it looks like it contains a link, WebKit stores the whole
// clipboard payload as a `public.url` UTI instead of plain text, which
// apps like Notes/Messages then render percent-encoded (spaces as %20,
// newlines as %0A). The legacy execCommand('copy') path via a selected
// textarea skips that heuristic and always writes plain text.
function copyPlainText(text: string): boolean {
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.style.position = 'fixed'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)
  textarea.focus()
  textarea.select()
  let success = false
  try {
    success = document.execCommand('copy')
  } catch {
    success = false
  }
  document.body.removeChild(textarea)
  return success
}

export async function copyText(text: string): Promise<void> {
  if (!copyPlainText(text)) {
    await navigator.clipboard.writeText(text)
  }
}

export function canCopyImages(): boolean {
  return typeof ClipboardItem !== 'undefined' && typeof navigator.clipboard?.write === 'function'
}

async function toPng(blob: Blob): Promise<Blob> {
  if (blob.type === 'image/png') return blob
  const url = URL.createObjectURL(blob)
  try {
    const img = new Image()
    img.src = url
    await img.decode()
    const canvas = document.createElement('canvas')
    canvas.width = img.naturalWidth
    canvas.height = img.naturalHeight
    canvas.getContext('2d')!.drawImage(img, 0, 0)
    return await new Promise<Blob>((resolve, reject) =>
      canvas.toBlob((png) => (png ? resolve(png) : reject(new Error('Image conversion failed'))), 'image/png'),
    )
  } finally {
    URL.revokeObjectURL(url)
  }
}

// Call directly from a tap. Only PNG is reliable; a promise keeps Safari's user gesture while converting.
export async function copyImage(blob: Blob | Promise<Blob>): Promise<void> {
  const png = Promise.resolve(blob).then(toPng)
  await navigator.clipboard.write([new ClipboardItem({ 'image/png': png })])
}

// Text + image: Word adds a picture of copied text (text wins); a copied web image carries its URL (image wins)
export function textWinsOverImage(text: string): boolean {
  return !/^\s*[a-z][\w+.-]*:\/\/\S+\s*$/i.test(text)
}

export interface ClipboardContents {
  text: string | null
  files: File[]
}

export async function readClipboard(): Promise<ClipboardContents> {
  if (typeof navigator.clipboard?.read === 'function') {
    const items = await navigator.clipboard.read()
    const files: File[] = []
    let text: string | null = null
    for (const item of items) {
      const imageType = item.types.find((type) => type.startsWith('image/'))
      const itemText = item.types.includes('text/plain') ? await (await item.getType('text/plain')).text() : null
      if (imageType && (itemText === null || !textWinsOverImage(itemText))) {
        const blob = await item.getType(imageType)
        files.push(new File([blob], generatedFileName(imageType), { type: imageType }))
      } else if (itemText !== null) {
        text = itemText
      }
    }
    return { text, files }
  }
  if (typeof navigator.clipboard?.readText === 'function') {
    return { text: await navigator.clipboard.readText(), files: [] }
  }
  throw new Error('Clipboard access is not supported in this browser')
}
