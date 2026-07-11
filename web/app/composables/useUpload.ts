// useUpload drives the asset service's three-step upload for a post's cover
// image: the blog backend mints a presigned blob URL (init), the browser PUTs the
// bytes straight to the asset service (bypassing the BFF proxy — hence the asset
// service's CORS), then the backend finalizes + snapshots the public cover URL.
// init/finalize go through the BFF (`useApi`, Bearer-injected); the PUT is a
// direct, credential-less signed link.
export function useUpload() {
  const { call } = useApi()

  type UploadInit = { uploadUrl: string; uploadToken: string; uploadHeaders?: Record<string, string> }

  function putWithProgress(url: string, file: File, onProgress?: (pct: number) => void, headers?: Record<string, string>): Promise<void> {
    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest()
      xhr.open('PUT', url)
      for (const [key, value] of Object.entries(headers ?? {})) xhr.setRequestHeader(key, value)
      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable && onProgress) onProgress(Math.round((e.loaded / e.total) * 100))
      }
      xhr.onload = () =>
        xhr.status >= 200 && xhr.status < 300
          ? resolve()
          : reject(new Error(`上传失败 (HTTP ${xhr.status})`))
      xhr.onerror = () => reject(new Error('上传网络错误(检查素材服务 CORS / 是否在线)'))
      xhr.send(file)
    })
  }

  // uploadCover: init → PUT → finalize for a post's (public) cover image.
  async function uploadCover(
    postId: string,
    file: File,
    onProgress?: (pct: number) => void
  ): Promise<{ coverAssetId: string; coverUrl: string }> {
    const init = await call<UploadInit>(
      `/api/v1/posts/${postId}/cover`,
      { method: 'POST', body: { filename: file.name, mime: file.type, size: file.size } }
    )
    await putWithProgress(init.uploadUrl, file, onProgress, init.uploadHeaders)
    return await call<{ coverAssetId: string; coverUrl: string }>(
      `/api/v1/posts/${postId}/cover/finalize`,
      { method: 'POST', body: { uploadToken: init.uploadToken } }
    )
  }

  // uploadImage: init → PUT → finalize for a standalone inline content image
  // (editor E2). Returns the public URL to embed in the post markdown.
  async function uploadImage(file: File, onProgress?: (pct: number) => void): Promise<{ url: string }> {
    const init = await call<UploadInit>(
      '/api/v1/images',
      { method: 'POST', body: { filename: file.name, mime: file.type, size: file.size } }
    )
    await putWithProgress(init.uploadUrl, file, onProgress, init.uploadHeaders)
    return await call<{ url: string }>(
      '/api/v1/images/finalize',
      { method: 'POST', body: { uploadToken: init.uploadToken } }
    )
  }

  return { uploadCover, uploadImage, putWithProgress }
}
