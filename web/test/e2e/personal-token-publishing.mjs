// Local, real-service acceptance. Run with `node test/e2e/personal-token-publishing.mjs`.
import { chromium, expect } from '@playwright/test';
import { mkdir, writeFile } from 'node:fs/promises';

const account = process.env.PAT_TEST_ACCOUNT_ORIGIN || 'http://account-blog.dev.yuelili.test:3600';
const blog = process.env.PAT_TEST_BLOG_ORIGIN || 'http://blog.dev.yuelili.test:3002';
for (const origin of [account, blog]) {
  if (!new URL(origin).hostname.endsWith('.dev.yuelili.test')) throw new Error('Use an isolated local development environment');
}
const output = 'test-results/personal-token-publishing';
await mkdir(output, { recursive: true });
const nonce = Date.now().toString();
const name = `PAT publishing acceptance ${nonce}`;
const browser = await chromium.launch({ headless: true });
const owner = await browser.newContext({ viewport: { width: 1280, height: 900 } });
const plain = await browser.newContext();
const writer = await browser.newContext();
const page = await owner.newPage();
const pageErrors = [];
page.on('pageerror', error => pageErrors.push(error.message));
const results = [];
const tokens = [];
const grants = [];
const categories = [];
const series = [];
const posts = [];
let fullToken;
let prefix;

async function call(client, base, path, method = 'GET', data, token, expected = 200) {
  const response = await client.fetch(base + path, {
    method, ...(data === undefined ? {} : { data }),
    ...(token ? { headers: { Authorization: `Bearer ${token}` } } : {}),
  });
  results.push({ method, path, status: response.status(), expected });
  if (response.status() !== expected) {
    let code = '';
    try { code = (await response.json()).code || ''; } catch {}
    throw new Error(`${method} ${path}: ${response.status()}, expected ${expected}; ${code}`);
  }
  return response.status() === 204 ? null : response.json();
}

async function mint(client, suffix, keys) {
  const result = await call(client, account, '/api/v1/pat', 'POST', {
    name: `${name} ${suffix}`, scopes: keys.map(key => prefix + key), expiresInDays: 1,
  }, undefined, 201);
  tokens.push({ client, id: result.id });
  return result.token;
}

async function upload(path, token) {
  const image = Buffer.from('R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7', 'base64');
  const opened = await call(plain.request, blog, path, 'POST', {
    filename: `pat-${nonce}.gif`, mime: 'image/gif', size: image.length,
  }, token, 201);
  const uploadURL = new URL(opened.uploadUrl, blog);
  const host = uploadURL.hostname;
  if (host !== '127.0.0.1' && !host.endsWith('.dev.yuelili.test')) throw new Error('Test upload must remain local');
  const put = await plain.request.put(uploadURL.href, { data: image, headers: { 'Content-Type': 'image/gif', ...opened.uploadHeaders } });
  expect([200, 201, 204]).toContain(put.status());
  results.push({ method: 'PUT', path: 'local upload URL', status: put.status() });
  return call(plain.request, blog, `${path}/finalize`, 'POST', { uploadToken: opened.uploadToken }, token);
}

try {
  await call(owner.request, account, '/api/v1/auth/login', 'POST', {
    email: process.env.PAT_TEST_EMAIL || 'test@example.com',
    password: process.env.PAT_TEST_PASSWORD || 'Yueli-local-development-2026',
  });
  await page.goto(`${blog}/auth/login?return_to=/manage`);
  await page.waitForURL(`${blog}/manage`, { timeout: 60000 });
  const directory = await call(owner.request, account, '/api/v1/pat/scopes');
  expect(directory.unavailableSites).toEqual([]);
  const blogPermissions = directory.items.filter(item => item.site === 'blog-main-web');
  expect(blogPermissions).toHaveLength(13);
  prefix = blogPermissions.find(item => item.key.endsWith(':blog.post.create')).key.slice(0, -'blog.post.create'.length);

  await page.goto(`${account}/developer-tokens`);
  await page.waitForFunction(() => document.querySelector('#__nuxt')?.__vue_app__?.$nuxt?.isHydrating === false);
  await page.getByRole('button', { name: '创建令牌', exact: true }).click();
  const dialog = page.getByRole('dialog');
  await dialog.getByPlaceholder('例如：本地脚本').fill(name);
  for (const permission of blogPermissions) await dialog.getByRole('checkbox', { name: permission.label, exact: true }).check();
  for (const width of [1280, 390]) {
    await page.setViewportSize({ width, height: 900 });
    await expect(dialog.getByRole('checkbox', { name: '管理分类与标签', exact: true })).toBeChecked();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    await page.screenshot({ path: `${output}/permissions-${width}.png`, animations: 'disabled' });
  }
  await page.setViewportSize({ width: 1280, height: 900 });
  const created = page.waitForResponse(response => response.url().endsWith('/api/v1/pat') && response.request().method() === 'POST');
  await dialog.getByRole('button', { name: '创建令牌', exact: true }).click();
  const mintedResponse = await created;
  expect(mintedResponse.status()).toBe(201);
  const minted = await mintedResponse.json();
  fullToken = minted.token;
  tokens.push({ client: owner.request, id: minted.id });
  expect(minted.scopes).toHaveLength(13);
  await expect(dialog).toHaveCount(0);
  await expect(page.getByText(name, { exact: true })).toBeVisible();
  console.log('Account UI: 13 permissions selected, temporary token created');

  const tokenCall = (path, method, data, token = fullToken, status = 200) => call(plain.request, blog, path, method, data, token, status);
  const createTaxonomy = async (kind, suffix, parentId) => {
    const data = await tokenCall('/api/v1/taxonomies', 'POST', { taxonomy: kind, name: `PAT ${suffix} ${nonce}`, slug: `pat-${suffix}-${nonce}`, parentId }, fullToken, 201);
    categories.push(data.taxonomy.id);
    return data.taxonomy;
  };
  const root = await createTaxonomy('category', 'root');
  const child = await createTaxonomy('category', 'child', root.id);
  expect(child.parentId).toBe(root.id);
  await tokenCall(`/api/v1/taxonomies/${child.id}`, 'PATCH', { name: `PAT renamed ${nonce}` });
  const tag = await createTaxonomy('tag', 'tag');
  const sourceTag = await createTaxonomy('tag', 'merge');
  await tokenCall(`/api/v1/taxonomies/${sourceTag.id}/merge`, 'POST', { targetId: tag.id }, fullToken, 204);

  const post = (await tokenCall('/api/v1/posts', 'POST', { title: `PAT publishing ${nonce}`, content: '# Draft\n\nImported markdown.' }, fullToken, 201)).post;
  posts.push(post.id);
  await tokenCall(`/api/v1/posts/${post.id}/taxonomies`, 'PUT', { taxonomyIds: [child.id, tag.id] }, fullToken, 204);
  const collection = (await tokenCall('/api/v1/series', 'POST', { name: `PAT series ${nonce}`, slug: `pat-series-${nonce}` }, fullToken, 201)).series;
  series.push(collection.id);
  await tokenCall(`/api/v1/series/${collection.id}`, 'PATCH', { description: 'Imported series' });
  await tokenCall(`/api/v1/posts/${post.id}/series`, 'PUT', { seriesId: collection.id, seriesOrder: 3 }, fullToken, 204);
  await tokenCall(`/api/v1/posts/${post.id}/flags`, 'PUT', { pinned: true, featured: true });
  console.log('Category hierarchy, tags, article assignment, series and flags passed');

  const edit = await mint(owner.request, 'edit', ['blog.post.update']);
  const media = await mint(owner.request, 'media', ['media.upload']);
  const tags = await mint(owner.request, 'tags', ['blog.tag.create']);
  const publish = await mint(owner.request, 'publish', ['blog.post.publish']);
  await tokenCall('/api/v1/taxonomies', 'POST', { taxonomy: 'category', name: 'Denied', slug: `denied-${nonce}` }, tags, 403);
  await tokenCall(`/api/v1/taxonomies/${child.id}`, 'PATCH', { name: 'Denied' }, tags, 403);
  await tokenCall(`/api/v1/posts/${post.id}/cover`, 'POST', { filename: 'denied.gif', mime: 'image/gif', size: 43 }, edit, 403);
  await tokenCall(`/api/v1/posts/${post.id}/cover`, 'POST', { filename: 'denied.gif', mime: 'image/gif', size: 43 }, media, 403);
  await tokenCall(`/api/v1/posts/${post.id}/flags`, 'PUT', { pinned: false, featured: false }, edit, 403);
  await tokenCall(`/api/v1/posts/${post.id}`, 'PATCH', { status: 'published', content: 'Denied' }, publish, 403);
  await tokenCall('/api/v1/authorization/manage/grants', 'POST', { subject: 'nobody', role: 'administrator' }, fullToken, 403);
  await tokenCall('/api/v1/posts/batch', 'POST', { ids: [post.id], action: 'publish' }, publish, 403);

  const bodyImage = await upload('/api/v1/images', media);
  const cover = await upload(`/api/v1/posts/${post.id}/cover`, fullToken);
  console.log('Body and cover uploads completed');
  expect(cover.coverUrl).toContain('v=');
  await tokenCall(`/api/v1/posts/${post.id}`, 'PATCH', { content: `# Imported article\n\n![fixture](${bodyImage.url})` });
  await tokenCall(`/api/v1/posts/${post.id}`, 'PATCH', { status: 'published' }, publish);
  const detail = await call(plain.request, blog, `/api/v1/posts/${post.slug}`);
  expect(detail.taxonomies.map(item => item.id).sort()).toEqual([child.id, tag.id].sort());
  expect(detail.series.id).toBe(collection.id);
  expect(detail.post.coverUrl).toBe(cover.coverUrl);

  await page.goto(`${blog}/posts/${post.slug}`);
  await expect(page.getByRole('heading', { name: post.title, exact: true })).toBeVisible();
  await expect(page.getByAltText('fixture', { exact: true })).toBeVisible();
  expect(await page.getByAltText('fixture', { exact: true }).evaluate(image => image.complete && image.naturalWidth > 0)).toBe(true);
  await page.screenshot({ path: `${output}/published.png`, animations: 'disabled' });

  const registered = await call(writer.request, account, '/api/v1/auth/register', 'POST', {
    email: `pat-publishing-${nonce}@example.test`, password: `Local-Pat-${nonce}!`, displayName: 'PAT publishing test author',
  }, undefined, 201);
  await call(writer.request, account, '/api/v1/auth/login', 'POST', { email: `pat-publishing-${nonce}@example.test`, password: `Local-Pat-${nonce}!` });
  const authorGrant = (await call(owner.request, blog, '/api/v1/authorization/manage/grants', 'POST', { subject: registered.userKey, role: 'author' }, undefined, 201)).grant.id;
  grants.push(authorGrant);
  const writerDirectory = await call(writer.request, account, '/api/v1/pat/scopes');
  expect(writerDirectory.items.some(item => item.key === prefix + 'blog.taxonomy.manage')).toBe(false);
  const writerToken = await mint(writer.request, 'writer', ['blog.post.create', 'blog.post.update', 'blog.tag.create', 'blog.series.create']);
  await tokenCall(`/api/v1/posts/${post.id}/taxonomies`, 'PUT', { taxonomyIds: [root.id] }, writerToken, 403);
  await tokenCall(`/api/v1/series/${collection.id}`, 'PATCH', { name: 'Denied' }, writerToken, 403);
  await tokenCall('/api/v1/taxonomies', 'POST', { taxonomy: 'category', name: 'Denied', slug: `denied-author-${nonce}` }, writerToken, 403);
  const writerTag = await tokenCall('/api/v1/taxonomies', 'POST', { taxonomy: 'tag', name: `PAT writer ${nonce}`, slug: `pat-writer-${nonce}` }, writerToken, 201);
  categories.push(writerTag.taxonomy.id);
  const writerPost = (await tokenCall('/api/v1/posts', 'POST', { title: `PAT writer ${nonce}`, content: 'Own draft' }, writerToken, 201)).post;
  posts.push(writerPost.id);
  await tokenCall(`/api/v1/posts/${writerPost.id}/taxonomies`, 'PUT', { taxonomyIds: [child.id, writerTag.taxonomy.id] }, writerToken, 204);
  // A delegated administrator's content scope works without granting authorization.manage.
  const elevatedGrant = (await call(owner.request, blog, '/api/v1/authorization/manage/grants', 'POST', { subject: registered.userKey, role: 'administrator' }, undefined, 201)).grant.id;
  grants.push(elevatedGrant);
  const elevated = await mint(writer.request, 'temporary-admin', ['blog.taxonomy.manage', 'blog.post.update']);
  await tokenCall(`/api/v1/posts/${post.id}/series`, 'PUT', { seriesId: collection.id, seriesOrder: 4 }, elevated, 204);
  await tokenCall(`/api/v1/taxonomies/${child.id}`, 'PATCH', { description: 'Temporary admin' }, elevated);
  await call(owner.request, blog, `/api/v1/authorization/manage/grants/${elevatedGrant}`, 'DELETE');
  grants.splice(grants.indexOf(elevatedGrant), 1);
  await tokenCall(`/api/v1/taxonomies/${child.id}`, 'PATCH', { description: 'Denied after revocation' }, elevated, 403);
  await tokenCall(`/api/v1/posts/${post.id}/series`, 'PUT', { seriesId: collection.id }, elevated, 403);
  await tokenCall(`/api/v1/posts/${post.id}`, 'PATCH', { status: 'archived' });
  await tokenCall(`/api/v1/posts/${post.id}`, 'DELETE', undefined, fullToken, 204);
  await tokenCall(`/api/v1/posts/${post.id}/restore`, 'POST', undefined);
  for (const id of posts) {
    await tokenCall(`/api/v1/posts/${id}/taxonomies`, 'PUT', { taxonomyIds: [] }, fullToken, 204);
    await tokenCall(`/api/v1/posts/${id}/series`, 'PUT', { seriesId: '' }, fullToken, 204);
  }
  for (const id of [...series].reverse()) {
    await tokenCall(`/api/v1/series/${id}`, 'DELETE', undefined, fullToken, 204);
    series.splice(series.indexOf(id), 1);
  }
  // Merged source records retain replacement references. Delete sources before targets.
  for (const id of [...categories].reverse()) {
    await tokenCall(`/api/v1/taxonomies/${id}`, 'DELETE', undefined, fullToken, 204);
    categories.splice(categories.indexOf(id), 1);
  }
  const revoked = tokens.find(item => item.id === minted.id);
  await call(owner.request, account, `/api/v1/pat/${revoked.id}`, 'DELETE', undefined, undefined, 204);
  tokens.splice(tokens.indexOf(revoked), 1);
  await tokenCall('/api/v1/posts/mine', 'GET', undefined, fullToken, 401);
  expect(pageErrors).toEqual([]);
  console.log(`Real local PAT publishing passed: ${results.length} requests`);
} catch (error) {
  await page.screenshot({ path: `${output}/failure.png`, animations: 'disabled' }).catch(() => {});
  await writeFile(`${output}/failure.txt`, page.url() + '\n' + await page.locator('body').innerText()).catch(() => {});
  throw error;
} finally {
  // Cleanup uses the test owner's browser session, never broad database deletion.
  const cleanup = [];
  for (const id of posts) {
    for (const [suffix, method, data] of [['taxonomies', 'PUT', { taxonomyIds: [] }], ['series', 'PUT', { seriesId: '' }], ['', 'DELETE', undefined]]) {
      const response = await owner.request.fetch(`${blog}/api/v1/posts/${id}${suffix ? '/' + suffix : ''}`, { method, ...(data ? { data } : {}) });
      cleanup.push({ kind: 'post', id, method, status: response.status() });
    }
  }
  for (const id of series.reverse()) cleanup.push({ kind: 'series', id, status: (await owner.request.delete(`${blog}/api/v1/series/${id}`)).status() });
  for (const id of categories.reverse()) cleanup.push({ kind: 'taxonomy', id, status: (await owner.request.delete(`${blog}/api/v1/taxonomies/${id}`)).status() });
  for (const id of grants.reverse()) cleanup.push({ kind: 'grant', id, status: (await owner.request.delete(`${blog}/api/v1/authorization/manage/grants/${id}`)).status() });
  for (const entry of tokens) cleanup.push({ kind: 'token', id: entry.id, status: (await entry.client.delete(`${account}/api/v1/pat/${entry.id}`)).status() });
  await writeFile(`${output}/acceptance.json`, JSON.stringify({ environment: 'real local services and PostgreSQL', results, pageErrors, cleanup }, null, 2));
  await browser.close();
  expect(cleanup.filter(item => item.status >= 300), 'All temporary tokens, grants and content must be cleaned up').toEqual([]);
}
