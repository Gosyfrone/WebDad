import { describe, expect, it } from 'vitest'

import {
  MAX_MODERATION_TEXT,
  MAX_BUG_TEXT,
  MAX_ATTACHMENT_BYTES,
  MAX_WARNING_MESSAGE_HINT,
  ALL_REASONS,
  categoryForReason,
  ReportApiError,
} from '@/lib/reports'

// ─── constantes ──────────────────────────────────────────────────────────────

describe('constantes', () => {
  it('MAX_MODERATION_TEXT vaut 255', () => {
    expect(MAX_MODERATION_TEXT).toBe(255)
  })

  it('MAX_BUG_TEXT vaut 500', () => {
    expect(MAX_BUG_TEXT).toBe(500)
  })

  it('MAX_ATTACHMENT_BYTES vaut 5 Mo', () => {
    expect(MAX_ATTACHMENT_BYTES).toBe(5 * 1024 * 1024)
  })

  it('MAX_WARNING_MESSAGE_HINT vaut 500', () => {
    expect(MAX_WARNING_MESSAGE_HINT).toBe(500)
  })

  it('ALL_REASONS contient les motifs attendus', () => {
    expect(ALL_REASONS).toContain('inappropriate')
    expect(ALL_REASONS).toContain('offensive')
    expect(ALL_REASONS).toContain('spam')
    expect(ALL_REASONS).toContain('bug')
    expect(ALL_REASONS).toContain('other')
    expect(ALL_REASONS.length).toBe(5)
  })
})

// ─── categoryForReason ────────────────────────────────────────────────────────

describe('categoryForReason', () => {
  it('bug → catégorie bug', () => {
    expect(categoryForReason('bug')).toBe('bug')
  })

  it('inappropriate → catégorie moderation', () => {
    expect(categoryForReason('inappropriate')).toBe('moderation')
  })

  it('offensive → catégorie moderation', () => {
    expect(categoryForReason('offensive')).toBe('moderation')
  })

  it('spam → catégorie moderation', () => {
    expect(categoryForReason('spam')).toBe('moderation')
  })

  it('other → catégorie moderation', () => {
    expect(categoryForReason('other')).toBe('moderation')
  })
})

// ─── ReportApiError ───────────────────────────────────────────────────────────

describe('ReportApiError', () => {
  it('stocke le status HTTP', () => {
    const err = new ReportApiError('Not found', 404)
    expect(err.status).toBe(404)
    expect(err.message).toBe('Not found')
    expect(err.name).toBe('ReportApiError')
  })

  it('est une instance d\'Error', () => {
    const err = new ReportApiError('Internal', 500)
    expect(err instanceof Error).toBe(true)
  })
})
