import { assetUploadURL, createAssetUploadMemo } from '@yueli/asset-nuxt/upload'
import { getApiFailure } from '@yueli/http-runtime'
import { optimizeImageFile } from '@yueli/ui/image/browser'

export type BlogImagePurpose = 'cover' | 'content'

export interface PngCompressionRequest {
  file: File
  purpose: BlogImagePurpose
  maxBytes?: number
}

export interface UseUploadOptions {
  confirmPngToJpeg?: (request: PngCompressionRequest) => Promise<boolean>
}

type UploadInit = {
  uploadUrl: string
  uploadToken: string
  uploadHeaders?: Record<string, string>
}

function uploadLimit(error: unknown): number | undefined {
  const failure = getApiFailure(error)
  if (failure?.code !== 'asset.upload.too_large' && failure?.code !== 'blog.asset_too_large') return undefined
  const maxBytes = Number(failure.params?.maxBytes)
  return Number.isFinite(maxBytes) && maxBytes > 0 ? maxBytes : 0
}

function formatMegabytes(bytes?: number) {
  return bytes ? `${(bytes / 1024 / 1024).toFixed(bytes >= 10 * 1024 * 1024 ? 0 : 1)} MB` : '当前配置'
}

function tooLargeMessage(file: File, purpose: BlogImagePurpose, maxBytes?: number, converted = false) {
  const label = purpose === 'cover' ? '封面图片' : '正文图片'
  const fileType = converted ? '转换后的 JPEG' : file.type === 'image/jpeg' ? 'JPEG' : '图片'
  return `${fileType}仍超过${label}上限（${formatMegabytes(maxBytes)}）。请自行压缩后重试，或在资源中心调整对应用途的大小限制。`
}

// Cover and inline-image uploads share one limit-recovery path. The Asset
// profile remains authoritative; the browser only offers a PNG→JPEG recovery
// after Asset explicitly returns asset.upload.too_large.
export function useUpload(options: UseUploadOptions = {}) {
  const { call } = useApi()
  const imageAttempts = createAssetUploadMemo<{ result?: Promise<{ url: string }> }>()

  function putWithProgress(url: string, file: File, onProgress?: (pct: number) => void, headers?: Record<string, string>): Promise<void> {
    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest()
      const transferURL = assetUploadURL(url)
      xhr.open('PUT', transferURL)
      for (const [key, value] of Object.entries(headers ?? {})) xhr.setRequestHeader(key, value)
      xhr.upload.onprogress = (event) => {
        if (event.lengthComputable && onProgress) onProgress(Math.round((event.loaded / event.total) * 100))
      }
      xhr.onload = () => xhr.status >= 200 && xhr.status < 300
        ? resolve()
        : reject(new Error(`上传失败 (HTTP ${xhr.status})`))
      xhr.onerror = () => reject(new Error(transferURL.startsWith('/asset-api/')
        ? '站点素材代理连接失败，请刷新后重试'
        : '对象存储直传失败，请检查网络或存储 CORS'))
      xhr.send(file)
    })
  }

  async function initializeWithRecovery(
    original: File,
    purpose: BlogImagePurpose,
    begin: (file: File) => Promise<UploadInit>,
  ): Promise<{ file: File, init: UploadInit }> {
    try {
      return { file: original, init: await begin(original) }
    } catch (error) {
      const maxBytes = uploadLimit(error)
      if (maxBytes === undefined) throw error
      if (original.type !== 'image/png' || !options.confirmPngToJpeg) {
        throw new Error(tooLargeMessage(original, purpose, maxBytes))
      }

      const confirmed = await options.confirmPngToJpeg({ file: original, purpose, maxBytes: maxBytes || undefined })
      if (!confirmed) {
        throw new Error(`已取消转换。请自行压缩后重试，或在资源中心调整${purpose === 'cover' ? '封面图片' : '正文图片'}限制。`)
      }
      const converted = await optimizeImageFile(original, {
        enabled: true,
        outputType: 'image/jpeg',
        maxSide: purpose === 'cover' ? 3200 : 4096,
        maxPixels: 24_000_000,
        quality: 0.86,
        discardLarger: false,
      })
      if (converted.type !== 'image/jpeg') {
        throw new Error('浏览器无法转换这张 PNG。请自行转为 JPEG 后重试。')
      }
      try {
        return { file: converted, init: await begin(converted) }
      } catch (retryError) {
        const retryLimit = uploadLimit(retryError)
        if (retryLimit !== undefined) {
          throw new Error(tooLargeMessage(converted, purpose, retryLimit || maxBytes, true))
        }
        throw retryError
      }
    }
  }

  async function uploadCover(
    postId: string,
    original: File,
    onProgress?: (pct: number) => void,
  ): Promise<{ coverAssetId: string, coverUrl: string }> {
    const { file, init } = await initializeWithRecovery(original, 'cover', file => call<UploadInit>(
      `/api/v1/posts/${postId}/cover`,
      { method: 'POST', body: { filename: file.name, mime: file.type, size: file.size } },
    ))
    await putWithProgress(init.uploadUrl, file, onProgress, init.uploadHeaders)
    return await call<{ coverAssetId: string, coverUrl: string }>(
      `/api/v1/posts/${postId}/cover/finalize`,
      { method: 'POST', body: { uploadToken: init.uploadToken } },
    )
  }

  async function sendImage(original: File, onProgress?: (pct: number) => void): Promise<{ url: string }> {
    const { file, init } = await initializeWithRecovery(original, 'content', file => call<UploadInit>(
      '/api/v1/images',
      { method: 'POST', body: { filename: file.name, mime: file.type, size: file.size } },
    ))
    await putWithProgress(init.uploadUrl, file, onProgress, init.uploadHeaders)
    return await call<{ url: string }>(
      '/api/v1/images/finalize',
      { method: 'POST', body: { uploadToken: init.uploadToken } },
    )
  }

  async function uploadImage(file: File, onProgress?: (pct: number) => void) {
    const attempt = await imageAttempts.get(file, () => ({}))
    return attempt.result ||= sendImage(file, onProgress).catch(error => { delete attempt.result; throw error })
  }
  return { uploadCover, uploadImage, putWithProgress }
}
