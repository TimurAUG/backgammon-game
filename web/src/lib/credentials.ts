// Персистентность кредов подключения (gameId + token) в localStorage.
// Connect сохраняет, App читает для авто-подключения, обработка
// UNAUTHORIZED чистит.

export interface Credentials {
  gameId: string
  token: string
}

const STORAGE_KEY = 'nardy.credentials'

export function saveCredentials(creds: Credentials): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(creds))
}

export function loadCredentials(): Credentials | null {
  const raw = localStorage.getItem(STORAGE_KEY)
  if (raw === null) return null
  try {
    const parsed: unknown = JSON.parse(raw)
    if (
      typeof parsed !== 'object' ||
      parsed === null ||
      typeof (parsed as Credentials).gameId !== 'string' ||
      typeof (parsed as Credentials).token !== 'string'
    ) {
      return null
    }
    const { gameId, token } = parsed as Credentials
    return { gameId, token }
  } catch {
    return null
  }
}

export function clearCredentials(): void {
  localStorage.removeItem(STORAGE_KEY)
}
