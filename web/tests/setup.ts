import '@testing-library/jest-dom/vitest'

// Node 25+ exposes a global localStorage that is unavailable without
// --localstorage-file and shadows jsdom's Storage. Drop it so jsdom's
// window.localStorage is used (CI runs Node 22 where this is a no-op).
const desc = Object.getOwnPropertyDescriptor(globalThis, 'localStorage')
if (desc && (!desc.get || desc.configurable)) {
  try {
    const probe = globalThis.localStorage
    if (probe != null && typeof probe.clear === 'function') {
      // usable — leave alone (Node 22 / proper --localstorage-file)
    } else {
      // broken placeholder
      Reflect.deleteProperty(globalThis, 'localStorage')
    }
  } catch {
    Reflect.deleteProperty(globalThis, 'localStorage')
  }
}

// After deleting Node's broken global, bind a simple Storage if jsdom's
// window.localStorage is still missing (opaque-origin SecurityError).
function installMemoryLocalStorage(): void {
  const map = new Map<string, string>()
  const memory: Storage = {
    get length() {
      return map.size
    },
    clear() {
      map.clear()
    },
    getItem(key: string) {
      return map.has(key) ? map.get(key)! : null
    },
    key(index: number) {
      return [...map.keys()][index] ?? null
    },
    removeItem(key: string) {
      map.delete(key)
    },
    setItem(key: string, value: string) {
      map.set(key, String(value))
    },
  }
  Object.defineProperty(window, 'localStorage', {
    configurable: true,
    enumerable: true,
    value: memory,
  })
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    enumerable: true,
    value: memory,
  })
}

try {
  void window.localStorage.clear
} catch {
  installMemoryLocalStorage()
}
if (typeof globalThis.localStorage?.clear !== 'function') {
  installMemoryLocalStorage()
}
