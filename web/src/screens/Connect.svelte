<script lang="ts">
  import { createGame, joinGame, login, logout, me, register } from '../lib/api'
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

  let gameId = $state('')
  let token = $state('')
  let nick = $state('')
  let password = $state('')
  let sessionNick = $state<string | null>(null)
  let created = $state<Credentials | null>(null)
  let error = $state('')
  let sessionChecked = $state(false)
  let loggingOut = $state(false)

  $effect(() => {
    let cancelled = false
    void (async () => {
      try {
        const result = await me()
        if (!cancelled) sessionNick = result.nick
      } catch {
        // 401 / network — stay anonymous; show login/register.
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
      const result = await register(nick, password)
      sessionNick = result.nick
      password = ''
    } catch (err) {
      error = authErrorMessage(err)
    }
  }

  async function handleLogin(): Promise<void> {
    error = ''
    try {
      const result = await login(nick, password)
      sessionNick = result.nick
      password = ''
    } catch (err) {
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
      sessionNick = null
      created = null
      password = ''
      onLoggedOut?.()
    } catch {
      error = 'Не удалось выйти'
    } finally {
      loggingOut = false
    }
  }

  // Ручной ввод — фоллбэк для отладки (gameId/token известны заранее).
  function handleSubmit(e: SubmitEvent): void {
    e.preventDefault()
    const creds = { gameId: gameId.trim(), token: token.trim() }
    if (creds.gameId === '' || creds.token === '') return
    saveCredentials(creds)
    onConnect(creds)
  }
</script>

<main class="connect">
  <h1>Длинные нарды</h1>

  {#if sessionChecked}
    {#if sessionNick !== null}
      <div class="session-row">
        <p class="session" data-testid="session-nick">{sessionNick}</p>
        <button type="button" data-testid="logout" onclick={handleLogout}>Выйти</button>
      </div>
    {:else}
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
    {/if}
  {/if}

  {#if created !== null}
    <p class="hint">Игра создана. Отправь сопернику ссылку:</p>
    <div class="invite">
      <input class="link" readonly value={inviteLink(created.gameId)} data-testid="invite-link" />
      <button type="button" onclick={copyLink}>Копировать</button>
    </div>
    <button type="button" class="primary" data-testid="enter-game" onclick={enterCreatedGame}>
      Войти в игру
    </button>
  {:else if inviteGameId !== null}
    <p class="hint">Тебя пригласили в игру.</p>
    <button type="button" class="primary" data-testid="join-invite" onclick={handleJoin}>
      Войти в игру
    </button>
  {:else}
    <button type="button" class="primary" data-testid="create-game" onclick={handleCreate}>
      Создать игру
    </button>

    <section class="manual">
      <p class="hint">или войти вручную:</p>
      <form onsubmit={handleSubmit}>
        <label>
          ID игры
          <input type="text" bind:value={gameId} autocomplete="off" />
        </label>
        <label>
          Токен
          <input type="password" bind:value={token} autocomplete="off" />
        </label>
        <button type="submit">Подключиться</button>
      </form>
    </section>
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
  .session-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 0.75rem;
  }
  .session {
    font-size: 14px;
    font-weight: 600;
    margin: 0;
  }
  .session-row button {
    border: 2px solid #2a1e10;
    background: #f4ece1;
    border-radius: 6px;
    padding: 0.25rem 0.75rem;
    font-size: 13px;
    font-weight: 600;
    color: #2a1e10;
    cursor: pointer;
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
    width: 100%;
    background: #2a1e10;
    color: #f4ece1;
    border: none;
    border-radius: 6px;
    padding: 0.7rem;
    font-size: 15px;
    font-weight: 600;
    cursor: pointer;
  }
  .primary:hover {
    background: #5a3a1e;
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
  .manual {
    margin-top: 1.5rem;
    border-top: 1px solid #c19a6b;
    padding-top: 1rem;
  }
  .manual form {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
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
  .manual button {
    background: #c19a6b;
    color: #2a1e10;
    border: none;
    border-radius: 6px;
    padding: 0.5rem;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
  }
  .error {
    color: #8b1c1c;
    font-size: 14px;
    font-weight: 600;
  }
</style>
