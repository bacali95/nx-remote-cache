import { describe, expect, it } from "vitest"
import { formatBytes, formatDate } from "@/lib/format"

describe("formatBytes", () => {
  it("renders sub-1024 byte counts as B", () => {
    expect(formatBytes(0)).toBe("0 B")
    expect(formatBytes(1023)).toBe("1023 B")
  })

  it("scales up through KB/MB/GB/TB", () => {
    expect(formatBytes(1024)).toBe("1.0 KB")
    expect(formatBytes(1536)).toBe("1.5 KB")
    expect(formatBytes(1024 * 1024)).toBe("1.0 MB")
    expect(formatBytes(1024 * 1024 * 1024)).toBe("1.0 GB")
    expect(formatBytes(1024 * 1024 * 1024 * 1024)).toBe("1.0 TB")
  })

  it("stays at TB instead of overflowing into a further unit", () => {
    expect(formatBytes(1024 * 1024 * 1024 * 1024 * 1024)).toBe("1024.0 TB")
  })
})

describe("formatDate", () => {
  it("formats an ISO string using the locale date/time representation", () => {
    const iso = "2024-03-05T12:00:00.000Z"
    expect(formatDate(iso)).toBe(new Date(iso).toLocaleString())
  })
})
