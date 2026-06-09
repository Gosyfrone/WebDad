import { describe, expect, it } from 'vitest'

import { shouldAttemptTranslation } from '@/lib/post-translation'

describe('shouldAttemptTranslation', () => {
  it('traduit une phrase clairement anglaise vers le français', () => {
    expect(shouldAttemptTranslation('my name is maxime', 'fr')).toBe(true)
  })

  it('ne traduit pas une phrase française avec un mot anglais isolé', () => {
    expect(shouldAttemptTranslation('hello les gars comment ca va ?', 'fr')).toBe(false)
  })

  it('ne traduit pas un texte trop court ou ambigu', () => {
    expect(shouldAttemptTranslation('hello', 'fr')).toBe(false)
    expect(shouldAttemptTranslation('ok lets go', 'fr')).toBe(false)
  })

  it('ne traduit pas une phrase française avec des fautes ou accents manquants', () => {
    expect(shouldAttemptTranslation('salut les gars coment sa va ?', 'fr')).toBe(false)
  })

  it('traduit une phrase clairement française vers l’anglais', () => {
    expect(shouldAttemptTranslation('bonjour les gars comment ca va', 'en')).toBe(true)
  })

  it('traduit les textes en chinois vers le français', () => {
    expect(shouldAttemptTranslation('你好，我叫马克西姆，今天很高兴', 'fr')).toBe(true)
  })

  it('traduit les textes en arabe vers le français', () => {
    expect(shouldAttemptTranslation('مرحبا اسمي ماكسيم كيف الحال اليوم', 'fr')).toBe(true)
  })

  it('ne traduit pas si le texte est déjà dans une écriture de la langue cible', () => {
    expect(shouldAttemptTranslation('你好，我叫马克西姆，今天很高兴', 'zh')).toBe(false)
  })

  it('traduit une phrase espagnole claire vers le français', () => {
    expect(shouldAttemptTranslation('hola como estas mi amigo', 'fr')).toBe(true)
  })
})
