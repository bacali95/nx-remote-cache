import { render, screen, waitFor } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import { ThemeProvider } from "@/components/theme-provider"

describe("ThemeProvider", () => {
  it("renders its children", () => {
    render(
      <ThemeProvider>
        <div>app content</div>
      </ThemeProvider>
    )
    expect(screen.getByText("app content")).toBeInTheDocument()
  })

  it("applies a forced theme as a class on the document element", async () => {
    render(
      <ThemeProvider forcedTheme="dark">
        <div>app content</div>
      </ThemeProvider>
    )
    await waitFor(() => expect(document.documentElement).toHaveClass("dark"))
  })
})
