import { publicAssetMediaUrl } from './asset-media.mjs'

interface CoverImageSource {
  coverAssetId?: string
  coverUrl?: string
}

export function coverThumbUrl(source: CoverImageSource): string {
	return coverRenditionUrl(source, 'card')
}

export function coverHeroUrl(source: CoverImageSource): string {
	return coverRenditionUrl(source, 'home')
}

export function coverRenditionUrl(source: CoverImageSource, rendition: string): string {
  const coverUrl = source.coverUrl || ''
  const assetId = source.coverAssetId || ''
  if (!assetId) return coverUrl

  if (!coverUrl) return publicAssetMediaUrl(assetId, rendition, '')

  try {
    const url = new URL(coverUrl)
    return publicAssetMediaUrl(assetId, rendition, url.origin)
  } catch {
    return publicAssetMediaUrl(assetId, rendition, '')
  }
}
