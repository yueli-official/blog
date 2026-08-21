import assert from 'node:assert/strict'
import test from 'node:test'

import { trafficSource } from './traffic-source.mjs'

test('classifies missing and same-site referrers without exposing paths', () => {
  assert.equal(trafficSource('', 'https://blog.example/posts/a'), 'direct')
  assert.equal(
    trafficSource('https://blog.example/category/writing?q=private', 'https://blog.example/posts/a'),
    'internal',
  )
})

test('keeps only the normalized external host', () => {
  assert.equal(
    trafficSource('https://www.Google.COM/search?q=private', 'https://blog.example/posts/a'),
    'google.com',
  )
})

test('rejects non-web and invalid referrers', () => {
  assert.equal(trafficSource('mailto:reader@example.com', 'https://blog.example/posts/a'), 'direct')
  assert.equal(trafficSource('not a url', 'https://blog.example/posts/a'), 'direct')
})
