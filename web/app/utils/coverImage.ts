import { publicAssetMediaUrl } from './asset-media.mjs'

interface CoverImageSource {
  coverAssetId?: string
  coverUrl?: string
}

export function coverThumbUrl(source: CoverImageSource): string {
  const coverUrl = source.coverUrl || ''
  const assetId = source.coverAssetId || ''
  if (!assetId) return coverUrl

  if (!coverUrl) return publicAssetMediaUrl(assetId, 'card', '')

  try {
    const url = new URL(coverUrl)
    return publicAssetMediaUrl(assetId, 'card', url.origin)
  } catch {
    return publicAssetMediaUrl(assetId, 'card', '')
  }
}
