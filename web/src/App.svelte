<script lang="ts">
  import Connect from './screens/Connect.svelte'
  import Game from './screens/Game.svelte'
  import Notifications from './components/Notifications.svelte'
  import { clearCredentials, loadCredentials, saveCredentials, type Credentials } from './lib/credentials'
  import type { ClientMessage, ServerMessage } from './protocol/messages'
  import { resetConnectionState, setConnectionState } from './stores/connection.svelte'
  import { applyServerMessage, resetGameState } from './stores/game.svelte'
  import { pushNotification, resetNotifications } from './stores/notifications.svelte'
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
  // Креды активной сессии — для личной ссылки возврата в Game.
  let activeCreds = $state<Credentials | null>(null)

  function startSession(creds: Credentials): void {
    const c = createClient(creds)
    c.onMessage(handleMessage)
    c.onStateChange(setConnectionState)
    c.connect()
    client = c
    activeCreds = creds
  }

  function handleAction(msg: ClientMessage): void {
    client?.send(msg)
  }

  // Входящие: UNAUTHORIZED означает невалидный токен — завершаем сессию
  // и возвращаемся в Connect (FRONTEND_SPEC #25); OPPONENT_JOINED — тост-
  // уведомление (#34a); остальное — в gameStore.
  function handleMessage(msg: ServerMessage): void {
    if (msg.type === 'ERROR' && msg.code === 'UNAUTHORIZED') {
      endSession()
      return
    }
    if (msg.type === 'OPPONENT_JOINED') {
      pushNotification('opponentJoined', 'Соперник присоединился')
    }
    applyServerMessage(msg)
  }

  // Закрыть сокет (стоп реконнект), сбросить креды и игровое состояние,
  // вернуться к Connect. Общий путь для «Новой игры» и UNAUTHORIZED.
  function endSession(): void {
    client?.close()
    client = null
    activeCreds = null
    clearCredentials()
    resetGameState()
    resetConnectionState()
    resetNotifications()
  }

  function handleNewGame(): void {
    endSession()
  }

  // Убирает token из адресной строки, оставляя ?game=<id> (публичный) — чтобы
  // личный токен не оставался в истории браузера и не копировался случайно.
  function stripTokenFromUrl(): void {
    const url = new URL(location.href)
    url.searchParams.delete('token')
    window.history.replaceState(null, '', url.pathname + url.search)
  }

  // Вход по приглашению: ?game=<id> в URL → Connect предложит войти в игру.
  const params = new URLSearchParams(location.search)
  const inviteGameId = params.get('game')
  const urlToken = params.get('token')

  // Личная ссылка для возврата: ?game=<id>&token=<token>. Реконнектимся по ним
  // (приоритет над сохранёнными — это явный заход по своей ссылке), сохраняем
  // креды и убираем token из адресной строки (FRONTEND_SPEC #30). Иначе —
  // авто-подключение по сохранённым кредам, минуя Connect (#24c).
  const saved = loadCredentials()
  if (inviteGameId !== null && urlToken !== null) {
    const creds = { gameId: inviteGameId, token: urlToken }
    saveCredentials(creds)
    stripTokenFromUrl()
    startSession(creds)
  } else if (saved !== null) {
    startSession(saved)
  }
</script>

{#if client === null}
  <Connect onConnect={startSession} {inviteGameId} />
{:else}
  <Game
    onAction={handleAction}
    onNewGame={handleNewGame}
    gameId={activeCreds?.gameId ?? null}
    token={activeCreds?.token ?? null}
  />
{/if}

<Notifications />
