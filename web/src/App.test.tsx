import { render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import { describe, expect, it, vi } from "vitest"
import App from "@/App"

const useAuth = vi.fn()

vi.mock("@/lib/auth-context", () => ({ useAuth: () => useAuth() }))
vi.mock("@/pages/LoginPage", () => ({ LoginPage: () => <div>login page</div> }))
vi.mock("@/pages/CachePage", () => ({ CachePage: () => <div>cache page</div> }))
vi.mock("@/pages/TokensPage", () => ({ TokensPage: () => <div>tokens page</div> }))
vi.mock("@/pages/UsersPage", () => ({ UsersPage: () => <div>users page</div> }))
vi.mock("@/pages/SettingsPage", () => ({ SettingsPage: () => <div>settings page</div> }))

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <App />
    </MemoryRouter>
  )
}

describe("App", () => {
  it("renders nothing for a protected route while auth is still loading", () => {
    useAuth.mockReturnValue({ user: null, loading: true })
    const { container } = renderAt("/")
    expect(container).toBeEmptyDOMElement()
  })

  it("redirects to /login when there is no authenticated user", () => {
    useAuth.mockReturnValue({ user: null, loading: false })
    renderAt("/")
    expect(screen.getByText("login page")).toBeInTheDocument()
  })

  it("renders the Layout with the Cache page at the index route once authenticated", () => {
    useAuth.mockReturnValue({ user: { id: 1, email: "admin@example.com" }, loading: false })
    renderAt("/")
    expect(screen.getByText("admin@example.com")).toBeInTheDocument()
    expect(screen.getByText("cache page")).toBeInTheDocument()
  })

  it.each([
    ["/tokens", "tokens page"],
    ["/users", "users page"],
    ["/settings", "settings page"],
  ])("renders the Layout with the matching page at %s", (path, text) => {
    useAuth.mockReturnValue({ user: { id: 1, email: "admin@example.com" }, loading: false })
    renderAt(path)
    expect(screen.getByText(text)).toBeInTheDocument()
  })

  it("does not gate /login behind RequireAuth", () => {
    useAuth.mockReturnValue({ user: null, loading: true })
    renderAt("/login")
    expect(screen.getByText("login page")).toBeInTheDocument()
  })
})
