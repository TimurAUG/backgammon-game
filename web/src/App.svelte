<script lang="ts">
  import Connect from './screens/Connect.svelte'
  import Game from './screens/Game.svelte'
  import { clearCredentials, loadCredentials, type Credentials } from './lib/credentials'
  import type { ClientMessage, ServerMessage } from './protocol/messages'
  import { resetConnectionState, setConnectionState } from './stores/connection.svelte'
  import { applyServerMessage, resetGameState } from './stores/game.svelte'
  import { WSClient } from './transport/ws'

  // createClient инжектируется в тестах (WSClient поверх MockWebSocket).
  // Дефолт — реальный сокет к /ws на текущем хосте; чистый bootstrap-glue.
  function defaultCreateClient(creds: Credentials): WSClient {
    const scheme = location.protocol === 'https:' ? 'wss' : 'ws'
    return new WSClient({
      url: `${scheme}://${location.host}/ws`,
      gameId: creds.gameId,
      token: creds.token,
    })
  }

  let { createClient = defaultCreateClient }: { createClient?: (creds: Credentials) => WSClient } =
    $props()

  // client === null → экран Connect; иначе → экран Game.
  let client = $state<WSClient | null>(null)

  function startSession(creds: Credentials): void {
    const c = createClient(creds)
    c.onMessage(handleMessage)
    c.onStateChange(setConnectionState)
    c.connect()
    client = c
  }

  function handleAction(msg: ClientMessage): void {
    client?.send(msg)
  }

  // Входящие: UNAUTHORIZED означает невалидный токен — завершаем сессию
  // и возвращаемся в Connect (FRONTEND_SPEC #25); остальное — в gameStore.
  function handleMessage(msg: ServerMessage): void {
    if (msg.type === 'ERROR' && msg.code === 'UNAUTHORIZED') {
      endSession()
      return
    }
    applyServerMessage(msg)
  }

  // Закрыть сокет (стоп реконнект), сбросить креды и игровое состояние,
  // вернуться к Connect. Общий путь для «Новой игры» и UNAUTHORIZED.
  function endSession(): void {
    client?.close()
    client = null
    clearCredentials()
    resetGameState()
    resetConnectionState()
  }

  function handleNewGame(): void {
    endSession()
  }

  // Авто-подключение: при сохранённых кредах минуем Connect и сразу
  // открываем сессию — сервер вернёт текущий STATE (FRONTEND_SPEC #24c).
  const saved = loadCredentials()
  if (saved !== null) startSession(saved)
</script>

{#if client === null}
  <Connect onConnect={startSession} />
{:else}
  <Game onAction={handleAction} onNewGame={handleNewGame} />
{/if}
