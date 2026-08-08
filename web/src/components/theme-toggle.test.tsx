import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it } from "vitest"
import { ThemeProvider } from "@/components/theme-provider"
import { ThemeToggle } from "@/components/theme-toggle"

function renderToggle() {
  return render(
    <ThemeProvider>
      <ThemeToggle />
    </ThemeProvider>
  )
}

describe("ThemeToggle", () => {
  it("switches to dark by opening the menu and picking Dark", async () => {
    const user = userEvent.setup()
    renderToggle()

    await user.click(screen.getByRole("button", { name: "Toggle theme" }))
    await user.click(await screen.findByText("Dark"))

    await waitFor(() => expect(document.documentElement).toHaveClass("dark"))
  })

  it("switches to light by picking Light", async () => {
    const user = userEvent.setup()
    renderToggle()

    await user.click(screen.getByRole("button", { name: "Toggle theme" }))
    await user.click(await screen.findByText("Light"))

    await waitFor(() => expect(document.documentElement).not.toHaveClass("dark"))
  })

  it("offers a System option that follows the OS preference", async () => {
    const user = userEvent.setup()
    renderToggle()

    await user.click(screen.getByRole("button", { name: "Toggle theme" }))
    await user.click(await screen.findByText("System"))

    await waitFor(() => expect(localStorage.getItem("theme")).toBe("system"))
  })
})
