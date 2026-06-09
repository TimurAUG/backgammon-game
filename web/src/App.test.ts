import { render, screen } from '@testing-library/svelte'
import { describe, expect, test } from 'vitest'

import App from './App.svelte'

describe('App', () => {
  test('App_renders_greeting', () => {
    render(App)
    const heading = screen.getByRole('heading', { level: 1 })
    expect(heading).toHaveTextContent('Длинные нарды — фронт каркас')
  })
})
