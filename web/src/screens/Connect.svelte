<script lang="ts">
  import { createGame, joinGame, login, logout, me, register, type MeResponse } from '../lib/api'
  import { clearCredentials, saveCredentials, type Credentials } from '../lib/credentials'

  let {
    onConnect,
    inviteGameId = null,
    onLoggedOut,
  }: {
    onConnect: (creds: Credentials) => void
    inviteGameId?: string | null
    onLoggedOut?: () => void
  } = $props()

  let nick = $state('')
  let password = $state('')
  let sessionNick = $state<string | null>(null)
  let sessionStats = $state<Omit<MeResponse, 'nick'> | null>(null)
  let created = $state<Credentials | null>(null)
  let error = $state('')
  let sessionChecked = $state(false)
  let loggingOut = $state(false)

  function applyMe(result: MeResponse): void {
    sessionNick = result.nick
    sessionStats = {
      played: result.played,
      wins: result.wins,
      losses: result.losses,
      win_percent: result.win_percent,
    }
  }

  function clearSession(): void {
    sessionNick = null
    sessionStats = null
  }

  $effect(() => {
    let cancelled = false
    void (async () => {
      try {
        const result = await me()
        if (!cancelled) applyMe(result)
      } catch {
        // 401 / network / malformed — stay anonymous; show login/register.
        // Do not invent a zero stats record.
      } finally {
        if (!cancelled) sessionChecked = true
      }
    })()
    return () => {
      cancelled = true
    }
  })

  function inviteLink(id: string): string {
    return `${location.origin}/?game=${id}`
  }

  function authErrorMessage(err: unknown): string {
    const status = typeof err === 'object' && err !== null && 'status' in err
      ? Number((err as { status: unknown }).status)
      : NaN
    if (status === 409) return 'Этот ник уже занят'
    if (status === 401) return 'Неверный ник или пароль'
    if (status === 400) return 'Проверьте ник и пароль'
    return 'Не удалось выполнить вход'
  }

  async function handleRegister(): Promise<void> {
    error = ''
    try {
      await register(nick, password)
      // Auth body is nick-only — re-fetch GET /api/me for stats (card #23).
      const result = await me()
      applyMe(result)
      password = ''
    } catch (err) {
      clearSession()
      error = authErrorMessage(err)
    }
  }

  async function handleLogin(): Promise<void> {
    error = ''
    try {
      await login(nick, password)
      // Auth body is nick-only — re-fetch GET /api/me for stats (card #23).
      const result = await me()
      applyMe(result)
      password = ''
    } catch (err) {
      clearSession()
      error = authErrorMessage(err)
    }
  }

  async function handleCreate(): Promise<void> {
    error = ''
    try {
      const creds = await createGame()
      saveCredentials(creds)
      created = creds
    } catch {
      error = 'Не удалось создать игру'
    }
  }

  async function handleJoin(): Promise<void> {
    if (inviteGameId === null) return
    error = ''
    try {
      const creds = await joinGame(inviteGameId)
      saveCredentials(creds)
      onConnect(creds)
    } catch {
      error = 'Не удалось войти в игру'
    }
  }

  function enterCreatedGame(): void {
    if (created !== null) onConnect(created)
  }

  function copyLink(): void {
    if (created !== null) void navigator.clipboard?.writeText(inviteLink(created.gameId))
  }

  async function handleLogout(): Promise<void> {
    if (loggingOut) return
    loggingOut = true
    error = ''
    try {
      await logout()
      clearCredentials()
      clearSession()
      created = null
      password = ''
      onLoggedOut?.()
    } catch {
      error = 'Не удалось выйти'
    } finally {
      loggingOut = false
    }
  }

  const showStats = $derived(
    sessionNick !== null &&
      sessionStats !== null &&
      created === null &&
      inviteGameId === null,
  )
</script>

<main class="connect">
  <h1>Длинные нарды</h1>

  {#if sessionChecked}
    {#if sessionNick === null}
      <!-- Вход: nick/password only -->
      <section class="auth">
        <label>
          Ник
          <input type="text" bind:value={nick} autocomplete="username" />
        </label>
        <label>
          Пароль
          <input type="password" bind:value={password} autocomplete="current-password" />
        </label>
        <div class="auth-actions">
          <button type="button" onclick={handleLogin}>Log in</button>
          <button type="button" onclick={handleRegister}>Register</button>
        </div>
      </section>
    {:else if created !== null}
      <!-- Игра создана: hint + invite + copy + enter; no session line, no account logout -->
      <p class="hint">Игра создана. Отправь сопернику ссылку:</p>
      <div class="invite">
        <input class="link" readonly value={inviteLink(created.gameId)} data-testid="invite-link" />
        <button type="button" onclick={copyLink}>Копировать</button>
      </div>
      <button type="button" class="primary" data-testid="enter-game" onclick={enterCreatedGame}>
        Войти в игру
      </button>
    {:else if inviteGameId !== null}
      <!-- Invite join: session + join CTA; no create, no logout -->
      <p class="session" data-testid="session-nick">Вы вошли как {sessionNick}</p>
      <p class="hint">Тебя пригласили в игру.</p>
      <button type="button" class="primary" data-testid="join-invite" onclick={handleJoin}>
        Войти в игру
      </button>
    {:else}
      <!-- Вошли: session + create + account logout + stats stacked -->
      <p class="session" data-testid="session-nick">Вы вошли как {sessionNick}</p>
      <button type="button" class="primary" data-testid="create-game" onclick={handleCreate}>
        Создать игру
      </button>
      <button type="button" class="primary account-logout" data-testid="logout" onclick={handleLogout}>
        Выйти из аккаунта
      </button>
      {#if showStats && sessionStats !== null}
        <section class="stats" data-testid="account-stats">
          <h2>Статистика</h2>
          <div class="stats-row">
            <span>Сыграно игр</span>
            <span data-testid="stats-played">{sessionStats.played}</span>
          </div>
          <div class="stats-row">
            <span>Победы</span>
            <span data-testid="stats-wins">{sessionStats.wins}</span>
          </div>
          <div class="stats-row">
            <span>Поражения</span>
            <span data-testid="stats-losses">{sessionStats.losses}</span>
          </div>
          <div class="stats-row">
            <span>Процент побед</span>
            <span data-testid="stats-win-percent">{sessionStats.win_percent}%</span>
          </div>
        </section>
      {/if}
    {/if}
  {/if}

  {#if error !== ''}<p class="error" role="alert">{error}</p>{/if}
</main>

<style>
  .connect {
    max-width: 360px;
    margin: 4rem auto;
    color: #2a1e10;
  }
  h1 {
    text-align: center;
    font-size: 1.5rem;
  }
  .hint {
    font-size: 14px;
    color: #5a3a1e;
  }
  .session {
    font-size: 14px;
    font-weight: 600;
    margin: 0 0 0.75rem;
    overflow-wrap: anywhere;
    word-break: break-word;
  }
  .auth {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
    margin-bottom: 1rem;
  }
  .auth-actions {
    display: flex;
    gap: 0.5rem;
  }
  .auth-actions button {
    flex: 1;
    background: #c19a6b;
    color: #2a1e10;
    border: none;
    border-radius: 6px;
    padding: 0.5rem;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
  }
  .primary {
    display: block;
    width: 100%;
    background: #2a1e10;
    color: #f4ece1;
    border: none;
    border-radius: 6px;
    padding: 0.7rem;
    font-size: 15px;
    font-weight: 600;
    cursor: pointer;
    margin-bottom: 0.5rem;
  }
  .primary:hover {
    background: #5a3a1e;
  }
  .account-logout {
    margin-top: 0;
  }
  .stats {
    margin-top: 0.75rem;
  }
  .stats h2 {
    margin: 0 0 0.5rem;
    font-size: 14px;
    font-weight: 600;
    color: #2a1e10;
    text-align: left;
  }
  .stats-row {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    font-size: 14px;
    color: #2a1e10;
    margin-bottom: 0.35rem;
  }
  .invite {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 0.75rem;
  }
  .invite .link {
    flex: 1;
    border: 2px solid #2a1e10;
    border-radius: 6px;
    padding: 0.5rem;
    font-size: 13px;
  }
  .invite button {
    border: 2px solid #2a1e10;
    background: #f4ece1;
    border-radius: 6px;
    padding: 0 0.75rem;
    font-weight: 600;
    cursor: pointer;
  }
  label {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    font-size: 14px;
    font-weight: 600;
  }
  input {
    border: 2px solid #2a1e10;
    border-radius: 6px;
    padding: 0.5rem;
    font-size: 14px;
  }
  .error {
    color: #8b1c1c;
    font-size: 14px;
    font-weight: 600;
  }
</style>
