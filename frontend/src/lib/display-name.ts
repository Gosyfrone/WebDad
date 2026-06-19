const DISPLAY_NAME_PATTERN = /^[\p{L}\p{M}\p{N} ._-]+$/u

/** Lettres Unicode, chiffres, espaces, points, tirets et underscores uniquement. */
export function isValidDisplayName(value: string): boolean {
  const name = value.trim()
  return name.length > 0 && DISPLAY_NAME_PATTERN.test(name)
}
