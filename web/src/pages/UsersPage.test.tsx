import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { toast } from "sonner"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { UsersPage } from "@/pages/UsersPage"
import { api, ApiError } from "@/lib/api"

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api")
  return {
    ...actual,
    api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), del: vi.fn() },
  }
})
vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

const currentUser = { id: 1, email: "admin@example.com", createdAt: "2024-01-01T00:00:00Z" }
const otherUser = { id: 2, email: "other@example.com", createdAt: "2024-01-02T00:00:00Z" }

vi.mock("@/lib/auth-context", () => ({ useAuth: () => ({ user: currentUser }) }))

describe("UsersPage", () => {
  beforeEach(() => {
    vi.mocked(api.get).mockResolvedValue([currentUser, otherUser])
    vi.mocked(api.post).mockResolvedValue(undefined)
    vi.mocked(api.del).mockResolvedValue(undefined)
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it("marks the current user's row and hides its own delete button", async () => {
    render(<UsersPage />)
    await screen.findByText("other@example.com")

    const ownRow = screen.getByText("admin@example.com").closest("tr")!
    expect(within(ownRow).getByText("(you)")).toBeInTheDocument()
    expect(within(ownRow).queryByRole("button", { name: "Delete" })).not.toBeInTheDocument()

    const otherRow = screen.getByText("other@example.com").closest("tr")!
    expect(within(otherRow).getByRole("button", { name: "Delete" })).toBeInTheDocument()
  })

  it("validates the new-user form before submitting", async () => {
    const user = userEvent.setup()
    render(<UsersPage />)
    await screen.findByText("other@example.com")

    await user.click(screen.getByRole("button", { name: "New user" }))
    await user.type(screen.getByLabelText("Email"), "new@example.com")
    await user.type(screen.getByLabelText("Password"), "short")
    await user.click(screen.getByRole("button", { name: "Create" }))

    expect(toast.error).toHaveBeenCalledWith(
      "Email required, password must be at least 8 characters"
    )
    expect(api.post).not.toHaveBeenCalled()
  })

  it("shows a toast when loading users fails", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("boom"))
    render(<UsersPage />)
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("Failed to load users"))
  })

  it("cancels the new-user dialog without creating one", async () => {
    const user = userEvent.setup()
    render(<UsersPage />)
    await screen.findByText("other@example.com")

    await user.click(screen.getByRole("button", { name: "New user" }))
    await user.type(screen.getByLabelText("Email"), "abandoned@example.com")
    await user.click(screen.getByRole("button", { name: "Cancel" }))

    await waitFor(() => expect(screen.queryByText("Create admin user")).not.toBeInTheDocument())
    expect(api.post).not.toHaveBeenCalled()
  })

  it("creates a user and reloads the list", async () => {
    const user = userEvent.setup()
    render(<UsersPage />)
    await screen.findByText("other@example.com")

    await user.click(screen.getByRole("button", { name: "New user" }))
    await user.type(screen.getByLabelText("Email"), "new@example.com")
    await user.type(screen.getByLabelText("Password"), "longenoughpassword")
    await user.click(screen.getByRole("button", { name: "Create" }))

    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/users", {
        email: "new@example.com",
        password: "longenoughpassword",
      })
    )
    expect(toast.success).toHaveBeenCalledWith("User created")
    await waitFor(() => expect(screen.queryByText("Create admin user")).not.toBeInTheDocument())
  })

  it("surfaces the server's error message when creating a user fails", async () => {
    vi.mocked(api.post).mockRejectedValue(new ApiError(409, "email already in use"))
    const user = userEvent.setup()
    render(<UsersPage />)
    await screen.findByText("other@example.com")

    await user.click(screen.getByRole("button", { name: "New user" }))
    await user.type(screen.getByLabelText("Email"), "new@example.com")
    await user.type(screen.getByLabelText("Password"), "longenoughpassword")
    await user.click(screen.getByRole("button", { name: "Create" }))

    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("email already in use"))
  })

  it("falls back to a generic message when creating a user fails with a non-API error", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("network down"))
    const user = userEvent.setup()
    render(<UsersPage />)
    await screen.findByText("other@example.com")

    await user.click(screen.getByRole("button", { name: "New user" }))
    await user.type(screen.getByLabelText("Email"), "new@example.com")
    await user.type(screen.getByLabelText("Password"), "longenoughpassword")
    await user.click(screen.getByRole("button", { name: "Create" }))

    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("Failed to create user"))
  })

  it("treats a null user list from the API as empty", async () => {
    vi.mocked(api.get).mockResolvedValue(null)
    render(<UsersPage />)
    expect(await screen.findByText("No users.")).toBeInTheDocument()
  })

  it("deletes another user via the confirm dialog", async () => {
    const user = userEvent.setup()
    render(<UsersPage />)
    await screen.findByText("other@example.com")

    const otherRow = screen.getByText("other@example.com").closest("tr")!
    await user.click(within(otherRow).getByRole("button", { name: "Delete" }))
    const dialog = await screen.findByRole("alertdialog")
    await user.click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(api.del).toHaveBeenCalledWith("/users/2"))
    expect(toast.success).toHaveBeenCalledWith("User deleted")
  })

  it("surfaces the server's error message when deleting a user fails", async () => {
    vi.mocked(api.del).mockRejectedValue(new ApiError(400, "cannot delete the last admin"))
    const user = userEvent.setup()
    render(<UsersPage />)
    await screen.findByText("other@example.com")

    const otherRow = screen.getByText("other@example.com").closest("tr")!
    await user.click(within(otherRow).getByRole("button", { name: "Delete" }))
    const dialog = await screen.findByRole("alertdialog")
    await user.click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("cannot delete the last admin"))
  })

  it("falls back to a generic message when deleting a user fails with a non-API error", async () => {
    vi.mocked(api.del).mockRejectedValue(new Error("network down"))
    const user = userEvent.setup()
    render(<UsersPage />)
    await screen.findByText("other@example.com")

    const otherRow = screen.getByText("other@example.com").closest("tr")!
    await user.click(within(otherRow).getByRole("button", { name: "Delete" }))
    const dialog = await screen.findByRole("alertdialog")
    await user.click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("Failed to delete user"))
  })

  it("validates the new password's length before submitting", async () => {
    const user = userEvent.setup()
    render(<UsersPage />)
    await screen.findByText("other@example.com")

    await user.type(screen.getByLabelText("Current password"), "current-pw")
    await user.type(screen.getByLabelText("New password"), "short")
    await user.click(screen.getByRole("button", { name: "Update password" }))

    expect(toast.error).toHaveBeenCalledWith("New password must be at least 8 characters")
    expect(api.post).not.toHaveBeenCalled()
  })

  it("changes the password and clears the fields", async () => {
    const user = userEvent.setup()
    render(<UsersPage />)
    await screen.findByText("other@example.com")

    await user.type(screen.getByLabelText("Current password"), "current-pw")
    await user.type(screen.getByLabelText("New password"), "new-long-password")
    await user.click(screen.getByRole("button", { name: "Update password" }))

    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/account/password", {
        currentPassword: "current-pw",
        newPassword: "new-long-password",
      })
    )
    expect(toast.success).toHaveBeenCalledWith("Password updated")
    expect(screen.getByLabelText("Current password")).toHaveValue("")
    expect(screen.getByLabelText("New password")).toHaveValue("")
  })

  it("surfaces the server's error message when changing the password fails", async () => {
    vi.mocked(api.post).mockRejectedValue(new ApiError(401, "current password is incorrect"))
    const user = userEvent.setup()
    render(<UsersPage />)
    await screen.findByText("other@example.com")

    await user.type(screen.getByLabelText("Current password"), "wrong")
    await user.type(screen.getByLabelText("New password"), "new-long-password")
    await user.click(screen.getByRole("button", { name: "Update password" }))

    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("current password is incorrect"))
  })

  it("falls back to a generic message when changing the password fails with a non-API error", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("network down"))
    const user = userEvent.setup()
    render(<UsersPage />)
    await screen.findByText("other@example.com")

    await user.type(screen.getByLabelText("Current password"), "wrong")
    await user.type(screen.getByLabelText("New password"), "new-long-password")
    await user.click(screen.getByRole("button", { name: "Update password" }))

    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("Failed to update password"))
  })
})
