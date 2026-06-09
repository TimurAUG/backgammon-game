<script lang="ts">
  import Connect from './screens/Connect.svelte'
  import Game from './screens/Game.svelte'
  import { clearCredentials, type Credentials } from './lib/credentials'
  import type { ClientMessage } from './protocol/messages'
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
    c.onMessage(applyServerMessage)
    c.connect()
    client = c
  }

  function handleAction(msg: ClientMessage): void {
    client?.send(msg)
  }

  function handleNewGame(): void {
    client?.close()
    client = null
    clearCredentials()
    resetGameState()
  }
</script>

{#if client === null}
  <Connect onConnect={startSession} />
{:else}
  <Game onAction={handleAction} onNewGame={handleNewGame} />
{/if}
