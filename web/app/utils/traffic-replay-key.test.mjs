import assert from 'node:assert/strict'
import test from 'node:test'

import { createTrafficReplayKey } from './traffic-replay-key.mjs'

test('uses randomUUID when the browser exposes it', () => {
  const expected = '019c0000-0000-4000-8000-000000000001'
  assert.equal(createTrafficReplayKey({ randomUUID: () => expected }), expected)
})

test('creates a UUID replay key when randomUUID is unavailable', () => {
  const cryptoSource = {
    getRandomValues(target) {
      target.fill(0)
      return target
    },
  }
  assert.equal(
    createTrafficReplayKey(cryptoSource),
    '00000000-0000-4000-8000-000000000000',
  )
})

test('keeps a non-crypto fallback within the Traffic event contract', () => {
  const value = createTrafficReplayKey(null)
  assert.match(value, /^view-[a-z0-9]+-[a-z0-9]{11,}$/)
  assert.ok(value.length >= 16)
  assert.ok(value.length <= 200)
})
