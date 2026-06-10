<script lang="ts">
  import { createGame, joinGame } from '../lib/api'
  import { saveCredentials, type Credentials } from '../lib/credentials'

  let {
    onConnect,
    inviteGameId = null,
  }: { onConnect: (creds: Credentials) => void; inviteGameId?: string | null } = $props()

  let gameId = $state('')
  let token = $state('')
  let created = $state<Credentials | null>(null)
  let error = $state('')

  function inviteLink(id: string): string {
    return `${location.origin}/?game=${id}`
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
