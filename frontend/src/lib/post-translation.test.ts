import { describe, expect, it } from 'vitest'

import { isTranslationCandidate, shouldAttemptTranslation } from '@/lib/post-translation'

describe('shouldAttemptTranslation', () => {
  it('traduit une phrase clairement anglaise vers le français', () => {
    expect(shouldAttemptTranslation('my name is maxime', 'fr')).toBe(true)
  })

  it('ne traduit pas une phrase française avec un mot anglais isolé', () => {
    expect(shouldAttemptTranslation('hello les gars comment ca va ?', 'fr')).toBe(false)
  })

  it('ne traduit pas un texte trop court ou ambigu', () => {
    expect(shouldAttemptTranslation('hello', 'fr')).toBe(false)
    expect(shouldAttemptTranslation('ok go', 'fr')).toBe(false)
  })

  it('ne traduit pas une phrase française avec des fautes ou accents manquants', () => {
    expect(shouldAttemptTranslation('salut les gars coment sa va ?', 'fr')).toBe(false)
    expect(shouldAttemptTranslation('je mapelle maxime', 'fr')).toBe(false)
  })

  it('ne traduit pas du français familier avec quelques mots empruntés', () => {
    expect(
      shouldAttemptTranslation('OMG la nouvelles fonctionnalités jsuis chokbar', 'fr'),
    ).toBe(false)
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

  it('détecte une source claire même si la langue cible utilise un autre alphabet', () => {
    expect(shouldAttemptTranslation('hello my name is maxime', 'ja')).toBe(true)
    expect(shouldAttemptTranslation('bonjour je suis maxime etudiant', 'ru')).toBe(true)
  })

  const samples = {
    fr: 'bonjour je suis maxime et je suis très heureux aujourd’hui',
    en: 'hello my name is maxime and i am very happy today',
    zh: '你好，我叫马克西姆，今天真的非常高兴',
    es: 'hola mi nombre es maxime y estoy muy feliz hoy',
    pt: 'olá meu nome é maxime e estou muito feliz hoje',
    ru: 'привет меня зовут максим и сегодня я очень счастлив',
    ja: 'こんにちは、私の名前はマキシムです。今日はとても幸せです',
    ko: '안녕하세요 제 이름은 막심이고 오늘 정말 행복합니다',
    ar: 'مرحبا اسمي ماكسيم وأنا سعيد جدا اليوم',
    hi: 'नमस्ते मेरा नाम मैक्सिम है और मैं आज बहुत खुश हूँ',
    de: 'hallo ich heiße maxime und ich bin heute sehr glücklich',
    it: 'ciao mi chiamo maxime e oggi sono molto felice',
  } as const

  it('laisse le fournisseur traiter toutes les paires de langues différentes', () => {
    for (const [source, text] of Object.entries(samples)) {
      for (const target of Object.keys(samples)) {
        if (source === target) continue
        expect(
          shouldAttemptTranslation(text, target),
          `${source} vers ${target}`,
        ).toBe(true)
      }
    }
  })

  it('évite localement les textes clairement déjà dans la langue cible', () => {
    for (const [locale, text] of Object.entries(samples)) {
      expect(shouldAttemptTranslation(text, locale), locale).toBe(false)
    }
  })
})

describe('isTranslationCandidate', () => {
  it('traduit les phrases courtes en écriture dense (1 caractère ≈ 1 mot)', () => {
    // Phrases normales mais < 12 caractères : le seuil latin les filtrait à tort.
    expect(isTranslationCandidate('안녕하세요 반갑습니다', 'fr')).toBe(true) // coréen
    expect(isTranslationCandidate('こんにちは、元気ですか', 'fr')).toBe(true) // japonais
    expect(isTranslationCandidate('你好，今天很高兴', 'fr')).toBe(true) // chinois
    expect(isTranslationCandidate('행복합니다', 'en')).toBe(true) // coréen très court
    expect(isTranslationCandidate('สวัสดีครับ', 'fr')).toBe(true) // thaï
  })

  it('traduit les phrases denses très courtes (2-3 caractères)', () => {
    expect(isTranslationCandidate('你好吗 ？', 'fr')).toBe(true) // 3 sinogrammes
    expect(isTranslationCandidate('你要去哪里？', 'fr')).toBe(true) // 5 sinogrammes
    expect(isTranslationCandidate('你好', 'fr')).toBe(true) // 2 sinogrammes
    expect(isTranslationCandidate('元気？', 'fr')).toBe(true) // japonais kanji court
  })

  it('rejette un unique caractère dense (trop ambigu)', () => {
    expect(isTranslationCandidate('好', 'fr')).toBe(false)
    expect(isTranslationCandidate('好 ？', 'fr')).toBe(false)
  })

  it('ne traduit pas une écriture dense déjà dans la langue cible', () => {
    expect(isTranslationCandidate('안녕하세요 반갑습니다', 'ko')).toBe(false)
    expect(isTranslationCandidate('你好，今天很高兴', 'zh')).toBe(false)
    expect(isTranslationCandidate('你好吗 ？', 'zh')).toBe(false)
  })

  it('garde le seuil de longueur pour les alphabets (anti-fragments)', () => {
    expect(isTranslationCandidate('hello', 'fr')).toBe(false)
    expect(isTranslationCandidate('ok go', 'fr')).toBe(false)
    expect(isTranslationCandidate('hello my name is maxime', 'fr')).toBe(true)
  })

  it('rejette les textes vides ou sans langue cible', () => {
    expect(isTranslationCandidate('   ', 'fr')).toBe(false)
    expect(isTranslationCandidate('안녕하세요 반갑습니다', '')).toBe(false)
  })
})
