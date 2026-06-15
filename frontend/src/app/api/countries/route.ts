import { NextResponse } from 'next/server'

type FirstCountriesResponse = {
  data?: Record<string, { country?: string }>
}

const COUNTRIES_URL = 'https://api.first.org/data/v1/countries?limit=250'

export async function GET() {
  try {
    const response = await fetch(COUNTRIES_URL, { next: { revalidate: 86_400 } })
    if (!response.ok) throw new Error(`FIRST countries returned ${response.status}`)

    const payload = (await response.json()) as FirstCountriesResponse
    const countries = Object.keys(payload.data ?? {}).filter((code) => /^[A-Z]{2}$/.test(code))
    return NextResponse.json({ data: countries })
  } catch {
    return NextResponse.json({ error: 'Countries unavailable.' }, { status: 502 })
  }
}
