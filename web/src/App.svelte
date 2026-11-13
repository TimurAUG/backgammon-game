<script lang="ts">
  import Connect from './screens/Connect.svelte'
  import Game from './screens/Game.svelte'
  import Notifications from './components/Notifications.svelte'
  import { me } from './lib/api'
  import { clearCredentials, loadCredentials, saveCredentials, type Credentials } from './lib/credentials'
  import type { ClientMessage, ServerMessage } from './protocol/messages'
  import { resetConnectionState, setConnectionState } from './stores/connection.svelte'
  import { applyServerMessage, gameState, resetGameState } from './stores/game.svelte'
  import { pushNotification, resetNotifications } from './stores/notifications.svelte'
  import { applyChat, applyChatHistory, chat, resetChat } from './stores/chat.svelte'
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
    if (msg.type === 'TURN_SKIPPED') {
      // Объясняем «молчаливый» авто-пропуск: выпали кубики, но ходить нечем,
      // ход перешёл. Кубики берём из сообщения (в STATE их уже сбросили).
      const dice = `${msg.dice.a} и ${msg.dice.b}`
      const text =
        msg.color === gameState.myColor
          ? `Выпало ${dice} — ходить нечем, ход переходит сопернику`
          : `Сопернику выпало ${dice} — ходить нечем, ваш ход`
      pushNotification('turnSkipped', text)
      return
    }
    if (msg.type === 'CHAT') {
      applyChat({ sender: msg.sender, text: msg.text })
      // Тост — только на чужое сообщение и только когда панель свёрнута:
      // открытую ленту игрок видит сам, своё эхо тостить незачем.
      if (msg.sender !== gameState.myColor && !chat.open) {
        pushNotification('chat', msg.text)
      }
      return
    }
    if (msg.type === 'CHAT_HISTORY') {
      applyChatHistory(msg.chat)
      return
    }
    applyServerMessage(msg)
  }

  // Закрыть сокет (стоп реконнект), сбросить креды и игровое состояние,
  // вернуться к Connect. Seat exit / UNAUTHORIZED — без logout аккаунта.
  function endSession(): void {
    client?.close()
    client = null
    clearCredentials()
    resetGameState()
    resetConnectionState()
    resetNotifications()
    resetChat()
  }

  // Убирает token из адресной строки; опционально ставит/чистит ?game=.
  // inviteGameId — $state, чтобы Connect после 401/seat-exit не держал
  // устаревший join-invite из значения, прочитанного на init.
  function replaceGameQuery(gameId: string | null): void {
    const url = new URL(location.href)
    if (gameId === null) {
      url.searchParams.delete('game')
    } else {
      url.searchParams.set('game', gameId)
    }
    url.searchParams.delete('token')
    window.history.replaceState(null, '', url.pathname + url.search)
    inviteGameId = gameId
  }

  // GameOver «Выйти»: только seat teardown + clear ?game= (без logout аккаунта).
  function handleExit(): void {
    endSession()
    replaceGameQuery(null)
  }

  // Убирает token из адресной строки, оставляя ?game=<id> (публичный) — чтобы
  // личный токен не оставался в истории браузера и не копировался случайно.
  function stripTokenFromUrl(): void {
    const url = new URL(location.href)
    url.searchParams.delete('token')
    window.history.replaceState(null, '', url.pathname + url.search)
  }

  // Вход по приглашению: ?game=<id> в URL → Connect предложит войти в игру
  // (только после login). inviteGameId — $state, обновляется при seat-exit.
  const initialParams = new URLSearchParams(location.search)
  const initialGameId = initialParams.get('game')
  const urlToken = initialParams.get('token')
  let inviteGameId = $state<string | null>(initialGameId)

  // Auth gate (#19): await me() first. Only if nick present, reconnect from
  // URL token / saved credentials. me fail → Connect (Вход); no board.
  // While me pending, client stays null → Connect title-only (sessionChecked false).
  $effect(() => {
    let cancelled = false
    void (async () => {
      try {
        await me()
      } catch {
        // 401 / network — stay on Connect; do not open board from seat creds.
        return
      }
      if (cancelled || client !== null) return

      const saved = loadCredentials()
      if (initialGameId !== null && urlToken !== null) {
        const creds = { gameId: initialGameId, token: urlToken }
        saveCredentials(creds)
        stripTokenFromUrl()
        startSession(creds)
      } else if (saved !== null && (initialGameId === null || saved.gameId === initialGameId)) {
        // Авто-подключение по сохранённым кредам — только когда нет приглашения
        // (обычный возврат) ИЛИ приглашение в СВОЮ же сохранённую игру
        // (F5/повторный заход по ссылке). Если ?game= указывает на ДРУГУЮ игру —
        // по saved не подключаемся: Connect покажет join-invite.
        startSession(saved)
      }
    })()
    return () => {
      cancelled = true
    }
  })
</script>

{#if client === null}
  <Connect onConnect={startSession} {inviteGameId} onLoggedOut={() => replaceGameQuery(null)} />
{:else}
  <Game onAction={handleAction} onExit={handleExit} />
{/if}

<Notifications />
