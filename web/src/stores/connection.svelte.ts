// connection — состояние WS-сокета для UI. App обновляет его из колбека
// WSClient (#26b); ActionBar блокируется при 'reconnecting' (#26c).
// Модульный $state-объект, как gameState: единственная мутация —
// через setConnectionState.

export type ConnectionState = 'idle' | 'connecting' | 'connected' | 'reconnecting' | 'error'

export const connection = $state<{ state: ConnectionState }>({ state: 'idle' })

export function setConnectionState(state: ConnectionState): void {
  connection.state = state
}

export function resetConnectionState(): void {
  connection.state = 'idle'
}
