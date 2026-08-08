import { cleanup } from "@testing-library/react"
import { afterEach } from "vitest"
import "@testing-library/jest-dom/vitest"

afterEach(cleanup)

// jsdom has no matchMedia implementation; next-themes (system-theme
// detection) needs one to even mount.
window.matchMedia ??= (query: string) => ({
  matches: false,
  media: query,
  onchange: null,
  addListener: () => {},
  removeListener: () => {},
  addEventListener: () => {},
  removeEventListener: () => {},
  dispatchEvent: () => false,
})
