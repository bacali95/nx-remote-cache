import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { toast } from "sonner"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { SettingsPage } from "@/pages/SettingsPage"
import { api, ApiError, type Settings } from "@/lib/api"

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api")
  return {
    ...actual,
    api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), del: vi.fn() },
  }
})
vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

const baseSettings: Settings = {
  storageBackend: "local",
  localDir: "./data",
  s3Bucket: "",
  s3Region: "",
  s3Prefix: "",
  s3Endpoint: "",
  s3UsePathStyle: false,
  s3AccessKeyIdSet: true,
  s3SecretAccessKeySet: false,
  gcsBucket: "",
  gcsPrefix: "",
  gcsCredentialsSet: false,
  sessionTtlSeconds: 86400,
  maxCacheEntryBytes: 524288000,
  updatedAt: "2024-01-01T00:00:00Z",
}

describe("SettingsPage", () => {
  beforeEach(() => {
    vi.mocked(api.get).mockResolvedValue(baseSettings)
    vi.mocked(api.put).mockResolvedValue(baseSettings)
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it("shows a loading state before settings arrive", () => {
    vi.mocked(api.get).mockReturnValue(new Promise(() => {}))
    render(<SettingsPage />)
    expect(screen.getByText("Loading settings…")).toBeInTheDocument()
  })

  it("stays on the loading state and toasts when the load fails", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("boom"))
    render(<SettingsPage />)
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("Failed to load settings"))
    expect(screen.getByText("Loading settings…")).toBeInTheDocument()
  })

  it("applies the loaded settings into the local disk form", async () => {
    render(<SettingsPage />)
    expect(await screen.findByLabelText("Directory")).toHaveValue("./data")
    expect(screen.getByLabelText("Admin session lifetime (hours)")).toHaveValue(24)
    expect(screen.getByLabelText("Max cache entry size (MB)")).toHaveValue(500)
  })

  it("fills in the S3 tab, clears a configured secret, and saves with those values", async () => {
    const user = userEvent.setup()
    render(<SettingsPage />)
    await screen.findByLabelText("Directory")

    await user.click(screen.getByRole("tab", { name: "S3" }))
    expect(screen.getByText("configured")).toBeInTheDocument()
    expect(screen.getByText("not set")).toBeInTheDocument()

    await user.type(screen.getByLabelText("Bucket"), "my-bucket")
    await user.type(screen.getByLabelText("Region"), "us-east-1")
    await user.type(screen.getByLabelText("Prefix (optional)"), "nx/")
    await user.type(
      screen.getByLabelText("Endpoint (R2/MinIO, optional)"),
      "https://r2.example.com"
    )
    await user.click(screen.getByLabelText("Use path-style addressing (required for MinIO)"))
    await user.type(screen.getByLabelText("Secret access key"), "shh-its-a-secret")

    const clearButtons = screen.getAllByRole("button", { name: "Clear" })
    expect(clearButtons).toHaveLength(1)
    await user.click(clearButtons[0])
    expect(screen.getByLabelText("Access key ID")).toHaveValue("")

    await user.click(screen.getByRole("button", { name: "Save settings" }))

    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith(
        "/settings",
        expect.objectContaining({
          s3Bucket: "my-bucket",
          s3Region: "us-east-1",
          s3Prefix: "nx/",
          s3Endpoint: "https://r2.example.com",
          s3UsePathStyle: true,
          s3AccessKeyId: "",
          s3SecretAccessKey: "shh-its-a-secret",
        })
      )
    )
  })

  it("fills in the GCS tab and saves with those values", async () => {
    const user = userEvent.setup()
    render(<SettingsPage />)
    await screen.findByLabelText("Directory")

    await user.click(screen.getByRole("tab", { name: "GCS" }))
    await user.type(screen.getByLabelText("Bucket"), "gcs-bucket")
    await user.type(screen.getByLabelText("Prefix (optional)"), "nx/")
    await user.type(
      screen.getByLabelText("Service account key (JSON)"),
      '{{"type":"service_account"}'
    )

    await user.click(screen.getByRole("button", { name: "Save settings" }))

    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith(
        "/settings",
        expect.objectContaining({
          gcsBucket: "gcs-bucket",
          gcsPrefix: "nx/",
          gcsCredentialsJson: '{"type":"service_account"}',
        })
      )
    )
  })

  it("rejects a non-positive session TTL without saving", async () => {
    const user = userEvent.setup()
    render(<SettingsPage />)
    await screen.findByLabelText("Directory")

    const ttl = screen.getByLabelText("Admin session lifetime (hours)")
    await user.clear(ttl)
    await user.type(ttl, "0")
    await user.click(screen.getByRole("button", { name: "Save settings" }))

    expect(toast.error).toHaveBeenCalledWith("Session TTL must be a positive number of hours")
    expect(api.put).not.toHaveBeenCalled()
  })

  it("rejects a non-positive max entry size without saving", async () => {
    const user = userEvent.setup()
    render(<SettingsPage />)
    await screen.findByLabelText("Directory")

    const maxEntry = screen.getByLabelText("Max cache entry size (MB)")
    await user.clear(maxEntry)
    await user.type(maxEntry, "-5")
    await user.click(screen.getByRole("button", { name: "Save settings" }))

    expect(toast.error).toHaveBeenCalledWith("Max cache entry size must be a positive number of MB")
    expect(api.put).not.toHaveBeenCalled()
  })

  it("saves the local disk settings, converting hours/MB to seconds/bytes", async () => {
    const user = userEvent.setup()
    render(<SettingsPage />)
    await screen.findByLabelText("Directory")

    const dir = screen.getByLabelText("Directory")
    await user.clear(dir)
    await user.type(dir, "/data/cache")
    await user.click(screen.getByRole("button", { name: "Save settings" }))

    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith(
        "/settings",
        expect.objectContaining({
          storageBackend: "local",
          localDir: "/data/cache",
          sessionTtlSeconds: 86400,
          maxCacheEntryBytes: 524288000,
        })
      )
    )
    expect(toast.success).toHaveBeenCalledWith("Settings saved and applied — no restart needed")
  })

  it("surfaces the server's error message when saving fails", async () => {
    vi.mocked(api.put).mockRejectedValue(new ApiError(400, "bucket is unreachable"))
    const user = userEvent.setup()
    render(<SettingsPage />)
    await screen.findByLabelText("Directory")

    await user.click(screen.getByRole("button", { name: "Save settings" }))

    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("bucket is unreachable"))
  })

  it("falls back to a generic message when saving fails with a non-API error", async () => {
    vi.mocked(api.put).mockRejectedValue(new Error("network down"))
    const user = userEvent.setup()
    render(<SettingsPage />)
    await screen.findByLabelText("Directory")

    await user.click(screen.getByRole("button", { name: "Save settings" }))

    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("Failed to save settings"))
  })
})
