import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { getStoredIdentity, setStoredIdentity, getLegacyIdentity } from '@/lib/key-store'
import type { KeyPair } from '@/lib/crypto'

// IndexedDB minimale en mémoire : couvre open/upgrade/transaction/get/put utilisés
// par key-store. Les callbacks asynchrones sont déclenchés au prochain microtask.
function makeFakeIndexedDB() {
  const stores = new Map<string, Map<string, unknown>>()
  const db = {
    objectStoreNames: { contains: (n: string) => stores.has(n) },
    createObjectStore: (n: string) => { stores.set(n, new Map()); return {} },
    transaction: (name: string) => {
      const tx: { oncomplete: (() => void) | null; onerror: (() => void) | null; error: unknown; objectStore: (n: string) => unknown } = {
        oncomplete: null, onerror: null, error: null,
        objectStore: (n: string) => ({
          get: (key: string) => {
            const req: { result?: unknown; onsuccess: (() => void) | null; onerror: (() => void) | null } = { onsuccess: null, onerror: null }
            queueMicrotask(() => { req.result = stores.get(n)?.get(key); req.onsuccess?.() })
            return req
          },
          put: (val: unknown, key: string) => {
            stores.get(n)?.set(key, val)
            queueMicrotask(() => tx.oncomplete?.())
            return {}
          },
        }),
      }
      return tx
    },
    close: () => {},
  }
  return {
    open: () => {
      const req: { result: unknown; onupgradeneeded: (() => void) | null; onsuccess: (() => void) | null; onerror: (() => void) | null } = { result: db, onupgradeneeded: null, onsuccess: null, onerror: null }
      queueMicrotask(() => { req.onupgradeneeded?.(); req.onsuccess?.() })
      return req
    },
  }
}

const kp = (a: number, b: number): KeyPair => ({ publicKey: new Uint8Array([a]), privateKey: new Uint8Array([b]) })

beforeEach(() => vi.stubGlobal('indexedDB', makeFakeIndexedDB()))
afterEach(() => vi.unstubAllGlobals())

describe('key-store (IndexedDB par compte)', () => {
  it('aller-retour : setStoredIdentity puis getStoredIdentity', async () => {
    await setStoredIdentity('u1', kp(2, 9))
    const got = await getStoredIdentity('u1')
    expect([...got!.publicKey]).toEqual([2])
    expect([...got!.privateKey]).toEqual([9])
  })

  it('compte sans clé → null', async () => {
    expect(await getStoredIdentity('absent')).toBeNull()
  })

  it('clés isolées par userId', async () => {
    await setStoredIdentity('uA', kp(1, 1))
    await setStoredIdentity('uB', kp(2, 2))
    expect([...(await getStoredIdentity('uA'))!.publicKey]).toEqual([1])
    expect([...(await getStoredIdentity('uB'))!.publicKey]).toEqual([2])
  })

  it('userId vide → no-op (pas de lecture/écriture)', async () => {
    await setStoredIdentity('', kp(1, 1))
    expect(await getStoredIdentity('')).toBeNull()
  })

  it('getLegacyIdentity lit la clé globale « self »', async () => {
    await setStoredIdentity('self', kp(7, 7))
    const legacy = await getLegacyIdentity()
    expect([...legacy!.publicKey]).toEqual([7])
  })

  it('sans IndexedDB → null', async () => {
    vi.stubGlobal('indexedDB', undefined)
    expect(await getStoredIdentity('u1')).toBeNull()
    expect(await getLegacyIdentity()).toBeNull()
  })
})
