<script lang="ts">
  import Connect from './screens/Connect.svelte'
  import Game from './screens/Game.svelte'
  import Notifications from './components/Notifications.svelte'
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
  // вернуться к Connect. Общий путь для «Новой игры» и UNAUTHORIZED.
  function endSession(): void {
    client?.close()
    client = null
    activeCreds = null
    clearCredentials()
    resetGameState()
    resetConnectionState()
    resetNotifications()
    resetChat()
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
  } else if (saved !== null && (inviteGameId === null || saved.gameId === inviteGameId)) {
    // Авто-подключение по сохранённым кредам — только когда нет приглашения
    // (обычный возврат) ИЛИ приглашение в СВОЮ же сохранённую игру
    // (F5/повторный заход по ссылке). Если ?game= указывает на ДРУГУЮ игру —
    // по saved не подключаемся: Connect покажет join-invite, игрок получит
    // собственный токен (слот соперника). Иначе занял бы чужой слот — в одном
    // браузере стал бы дублем создателя (White), и розыгрыш первого хода
    // не запускался бы («оба бросают — ничего»).
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
