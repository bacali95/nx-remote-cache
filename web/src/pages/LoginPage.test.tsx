import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter, Route, Routes } from "react-router-dom"
import { afterEach, describe, expect, it, vi } from "vitest"
import { LoginPage } from "@/pages/LoginPage"
import { ApiError } from "@/lib/api"

const login = vi.fn()

vi.mock("@/lib/auth-context", () => ({
  useAuth: () => ({ login }),
}))

function renderLoginPage() {
  return render(
    <MemoryRouter initialEntries={["/login"]}>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/" element={<div>cache page</div>} />
      </Routes>
    </MemoryRouter>
  )
}

describe("LoginPage", () => {
  afterEach(() => {
    login.mockReset()
  })

  it("submits the entered credentials and navigates away on success", async () => {
    login.mockResolvedValue(undefined)
    const user = userEvent.setup()
    renderLoginPage()

    await user.type(screen.getByLabelText("Email"), "admin@example.com")
    await user.type(screen.getByLabelText("Password"), "hunter2")
    await user.click(screen.getByRole("button", { name: "Sign in" }))

    expect(login).toHaveBeenCalledWith("admin@example.com", "hunter2")
    await waitFor(() => expect(screen.getByText("cache page")).toBeInTheDocument())
  })

  it("shows the server's error message on a failed login", async () => {
    login.mockRejectedValue(new ApiError(401, "invalid credentials"))
    const user = userEvent.setup()
    renderLoginPage()

    await user.type(screen.getByLabelText("Email"), "admin@example.com")
    await user.type(screen.getByLabelText("Password"), "wrong")
    await user.click(screen.getByRole("button", { name: "Sign in" }))

    expect(await screen.findByText("invalid credentials")).toBeInTheDocument()
    expect(screen.queryByText("cache page")).not.toBeInTheDocument()
  })

  it("falls back to a generic message for a non-API error", async () => {
    login.mockRejectedValue(new Error("network down"))
    const user = userEvent.setup()
    renderLoginPage()

    await user.type(screen.getByLabelText("Email"), "admin@example.com")
    await user.type(screen.getByLabelText("Password"), "hunter2")
    await user.click(screen.getByRole("button", { name: "Sign in" }))

    expect(await screen.findByText("Login failed")).toBeInTheDocument()
  })
})
