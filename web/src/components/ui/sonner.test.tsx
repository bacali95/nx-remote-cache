import { render, screen, waitFor } from "@testing-library/react"
import { toast } from "sonner"
import { describe, expect, it } from "vitest"
import { Toaster } from "@/components/ui/sonner"

describe("Toaster", () => {
  it("renders and displays a toast pushed through sonner", async () => {
    render(<Toaster />)

    toast.success("Settings saved")

    await waitFor(() => expect(screen.getByText("Settings saved")).toBeInTheDocument())
  })
})
