import { act, render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { AuthProvider, useAuth } from "@/lib/auth-context"
import { api, ApiError } from "@/lib/api"

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api")
  return {
    ...actual,
    api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), del: vi.fn() },
  }
})

const mockUser = { id: 1, email: "admin@example.com", createdAt: "2024-01-01T00:00:00Z" }

function Consumer() {
  const { user, loading, login, logout } = useAuth()
  return (
    <div>
      <span data-testid="loading">{String(loading)}</span>
      <span data-testid="user">{user?.email ?? "none"}</span>
      <button onClick={() => login("admin@example.com", "hunter2")}>login</button>
      <button onClick={() => logout()}>logout</button>
    </div>
  )
}

describe("AuthProvider", () => {
  beforeEach(() => {
    vi.mocked(api.get).mockReset()
    vi.mocked(api.post).mockReset()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it("loads the current user on mount when the session is valid", async () => {
    vi.mocked(api.get).mockResolvedValue(mockUser)

    render(
      <AuthProvider>
        <Consumer />
      </AuthProvider>
    )

    expect(screen.getByTestId("loading")).toHaveTextContent("true")
    await waitFor(() => expect(screen.getByTestId("loading")).toHaveTextContent("false"))
    expect(screen.getByTestId("user")).toHaveTextContent("admin@example.com")
    expect(api.get).toHaveBeenCalledWith("/auth/me")
  })

  it("treats a 401 from /auth/me as logged out, not an error", async () => {
    vi.mocked(api.get).mockRejectedValue(new ApiError(401, "not authenticated"))

    render(
      <AuthProvider>
        <Consumer />
      </AuthProvider>
    )

    await waitFor(() => expect(screen.getByTestId("loading")).toHaveTextContent("false"))
    expect(screen.getByTestId("user")).toHaveTextContent("none")
  })

  it("login posts credentials then refreshes the user", async () => {
    vi.mocked(api.get).mockRejectedValueOnce(new ApiError(401, "not authenticated"))
    vi.mocked(api.post).mockResolvedValue(undefined)
    const user = userEvent.setup()

    render(
      <AuthProvider>
        <Consumer />
      </AuthProvider>
    )
    await waitFor(() => expect(screen.getByTestId("user")).toHaveTextContent("none"))

    vi.mocked(api.get).mockResolvedValueOnce(mockUser)
    await act(() => user.click(screen.getByText("login")))

    expect(api.post).toHaveBeenCalledWith("/auth/login", {
      email: "admin@example.com",
      password: "hunter2",
    })
    await waitFor(() => expect(screen.getByTestId("user")).toHaveTextContent("admin@example.com"))
  })

  it("logout posts to /auth/logout and clears the user without waiting on a refresh", async () => {
    vi.mocked(api.get).mockResolvedValue(mockUser)
    vi.mocked(api.post).mockResolvedValue(undefined)
    const user = userEvent.setup()

    render(
      <AuthProvider>
        <Consumer />
      </AuthProvider>
    )
    await waitFor(() => expect(screen.getByTestId("user")).toHaveTextContent("admin@example.com"))

    await act(() => user.click(screen.getByText("logout")))

    expect(api.post).toHaveBeenCalledWith("/auth/logout")
    expect(screen.getByTestId("user")).toHaveTextContent("none")
  })

  it("useAuth throws when rendered outside an AuthProvider", () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {})
    expect(() => render(<Consumer />)).toThrow("useAuth must be used within AuthProvider")
    consoleError.mockRestore()
  })
})
