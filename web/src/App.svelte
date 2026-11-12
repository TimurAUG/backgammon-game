<script lang="ts">
  import Connect from './screens/Connect.svelte'
  import Game from './screens/Game.svelte'
  import Notifications from './components/Notifications.svelte'
  import { createGame, logout } from './lib/api'
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
  // Состояние create-and-enter из GameOver «Новая игра» (card #6).
  let creatingNewGame = $state(false)
  let newGameError = $state('')

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
  // вернуться к Connect. Путь для «Сменить игру» и UNAUTHORIZED.
  function endSession(): void {
    client?.close()
    client = null
    activeCreds = null
    clearCredentials()
    resetGameState()
    resetConnectionState()
    resetNotifications()
    resetChat()
    creatingNewGame = false
    newGameError = ''
  }

  // «Выйти» из Game: POST /api/logout, затем teardown. Ошибка — до endSession,
  // чтобы Game показал alert и оставил seat/сокет.
  async function handleLogout(): Promise<void> {
    await logout()
    endSession()
    replaceGameQuery(null)
  }

  // Убирает token из адресной строки; опционально ставит/чистит ?game=.
  // inviteGameId — $state, чтобы Connect после 401/новой игры не держал
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

  // GameOver «Новая игра»: POST /api/games и сразу войти в новую сессию
  // (FRONTEND_SPEC #22 / card #6). Сначала create, потом teardown — чтобы
  // при ошибке оверлей остался и URL не менялся.
  async function handleNewGame(): Promise<void> {
    if (creatingNewGame) return
    creatingNewGame = true
    newGameError = ''
    try {
      const creds = await createGame()
      endSession()
      saveCredentials(creds)
      replaceGameQuery(creds.gameId)
      startSession(creds)
    } catch (err) {
      // ApiError и тестовые stubs с полем status — duck-type, без instanceof
      // (мок ./lib/api в тестах подменяет класс).
      const status =
        typeof err === 'object' && err !== null && 'status' in err
          ? (err as { status: unknown }).status
          : undefined
      if (status === 401) {
        endSession()
        replaceGameQuery(null)
        return
      }
      newGameError = 'Не удалось создать игру'
    } finally {
      creatingNewGame = false
    }
  }

  // Убирает token из адресной строки, оставляя ?game=<id> (публичный) — чтобы
  // личный токен не оставался в истории браузера и не копировался случайно.
  function stripTokenFromUrl(): void {
    const url = new URL(location.href)
    url.searchParams.delete('token')
    window.history.replaceState(null, '', url.pathname + url.search)
  }

  // Вход по приглашению: ?game=<id> в URL → Connect предложит войти в игру.
  // initial* — одноразовый bootstrap; inviteGameId — $state, обновляется при
  // create-and-enter / 401, чтобы Connect не держал stale join-invite.
  const initialParams = new URLSearchParams(location.search)
  const initialGameId = initialParams.get('game')
  const urlToken = initialParams.get('token')
  let inviteGameId = $state<string | null>(initialGameId)

  // Личная ссылка для возврата: ?game=<id>&token=<token>. Реконнектимся по ним
  // (приоритет над сохранёнными — это явный заход по своей ссылке), сохраняем
  // креды и убираем token из адресной строки (FRONTEND_SPEC #30). Иначе —
  // авто-подключение по сохранённым кредам, минуя Connect (#24c).
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
    // по saved не подключаемся: Connect покажет join-invite, игрок получит
    // собственный токен (слот соперника). Иначе занял бы чужой слот — в одном
    // браузере стал бы дублем создателя (White), и розыгрыш первого хода
    // не запускался бы («оба бросают — ничего»).
    startSession(saved)
  }
</script>

{#if client === null}
  <Connect onConnect={startSession} {inviteGameId} onLoggedOut={() => replaceGameQuery(null)} />
{:else}
  <Game
    onAction={handleAction}
    onNewGame={handleNewGame}
    onLeave={endSession}
    onLogout={handleLogout}
    gameId={activeCreds?.gameId ?? null}
    token={activeCreds?.token ?? null}
    {creatingNewGame}
    {newGameError}
  />
{/if}

<Notifications />
