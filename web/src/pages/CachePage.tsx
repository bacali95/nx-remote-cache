import { useCallback, useEffect, useState } from "react"
import { toast } from "sonner"
import { api, type CacheEntry, type ListCacheResponse } from "@/lib/api"
import { formatBytes, formatDate } from "@/lib/format"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"

export function CachePage() {
  const [entries, setEntries] = useState<CacheEntry[]>([])
  const [nextCursor, setNextCursor] = useState<string | undefined>()
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [pruneDays, setPruneDays] = useState("30")
  const [pruneOpen, setPruneOpen] = useState(false)

  const load = useCallback(async (cursor?: string, replace = true) => {
    setLoading(true)
    try {
      const qs = new URLSearchParams()
      if (cursor) qs.set("cursor", cursor)
      const page = await api.get<ListCacheResponse>(`/cache?${qs.toString()}`)
      setEntries((prev) => (replace ? page.entries ?? [] : [...prev, ...(page.entries ?? [])]))
      setNextCursor(page.nextCursor)
    } catch {
      toast.error("Failed to load cache entries")
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  function toggle(hash: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(hash)) next.delete(hash)
      else next.add(hash)
      return next
    })
  }

  function toggleAll() {
    setSelected((prev) => (prev.size === entries.length ? new Set() : new Set(entries.map((e) => e.hash))))
  }

  async function deleteOne(hash: string) {
    try {
      await api.del(`/cache/${encodeURIComponent(hash)}`)
      setEntries((prev) => prev.filter((e) => e.hash !== hash))
      toast.success(`Deleted ${hash}`)
    } catch {
      toast.error(`Failed to delete ${hash}`)
    }
  }

  async function deleteSelected() {
    try {
      const result = await api.post<{ deleted: number }>("/cache/bulk-delete", {
        hashes: Array.from(selected),
      })
      toast.success(`Deleted ${result.deleted} entr${result.deleted === 1 ? "y" : "ies"}`)
      setSelected(new Set())
      load()
    } catch {
      toast.error("Bulk delete failed")
    }
  }

  async function prune() {
    const days = Number(pruneDays)
    if (!Number.isFinite(days) || days <= 0) {
      toast.error("Enter a positive number of days")
      return
    }
    try {
      const result = await api.post<{ deleted: number }>("/cache/prune", { olderThanDays: days })
      toast.success(`Pruned ${result.deleted} entr${result.deleted === 1 ? "y" : "ies"} older than ${days} days`)
      setPruneOpen(false)
      load()
    } catch {
      toast.error("Prune failed")
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">Cache entries</h1>
        <div className="flex items-center gap-2">
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button variant="destructive" size="sm" disabled={selected.size === 0}>
                Delete selected ({selected.size})
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Delete {selected.size} cache entries?</AlertDialogTitle>
                <AlertDialogDescription>
                  This permanently removes the selected artifacts from storage. This cannot be undone.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction onClick={deleteSelected}>Delete</AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>

          <Dialog open={pruneOpen} onOpenChange={setPruneOpen}>
            <DialogTrigger asChild>
              <Button variant="outline" size="sm">
                Prune by age
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Prune old cache entries</DialogTitle>
                <DialogDescription>
                  Deletes every entry whose last write is older than the given number of days. This
                  scans the whole cache and cannot be undone.
                </DialogDescription>
              </DialogHeader>
              <div className="flex flex-col gap-2">
                <Label htmlFor="prune-days">Older than (days)</Label>
                <Input
                  id="prune-days"
                  type="number"
                  min={1}
                  value={pruneDays}
                  onChange={(e) => setPruneDays(e.target.value)}
                />
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setPruneOpen(false)}>
                  Cancel
                </Button>
                <Button variant="destructive" onClick={prune}>
                  Prune
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>

          <Button variant="outline" size="sm" onClick={() => load()}>
            Refresh
          </Button>
        </div>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-10">
              <Checkbox
                checked={entries.length > 0 && selected.size === entries.length}
                onCheckedChange={toggleAll}
                aria-label="Select all"
              />
            </TableHead>
            <TableHead>Hash</TableHead>
            <TableHead>Size</TableHead>
            <TableHead>Last modified</TableHead>
            <TableHead>Reads</TableHead>
            <TableHead>Last read</TableHead>
            <TableHead className="w-16" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {entries.map((entry) => (
            <TableRow key={entry.hash}>
              <TableCell>
                <Checkbox
                  checked={selected.has(entry.hash)}
                  onCheckedChange={() => toggle(entry.hash)}
                  aria-label={`Select ${entry.hash}`}
                />
              </TableCell>
              <TableCell className="font-mono text-sm">{entry.hash}</TableCell>
              <TableCell>{formatBytes(entry.size)}</TableCell>
              <TableCell>{formatDate(entry.modTime)}</TableCell>
              <TableCell>{entry.readCount}</TableCell>
              <TableCell>{entry.lastReadAt ? formatDate(entry.lastReadAt) : "never"}</TableCell>
              <TableCell>
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <Button variant="ghost" size="sm">
                      Delete
                    </Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>Delete this entry?</AlertDialogTitle>
                      <AlertDialogDescription>
                        Permanently removes <span className="font-mono">{entry.hash}</span> from storage.
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>Cancel</AlertDialogCancel>
                      <AlertDialogAction onClick={() => deleteOne(entry.hash)}>Delete</AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </TableCell>
            </TableRow>
          ))}
          {!loading && entries.length === 0 && (
            <TableRow>
              <TableCell colSpan={7} className="text-center text-muted-foreground">
                No cache entries.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>

      {nextCursor && (
        <div className="flex justify-center">
          <Button variant="outline" size="sm" disabled={loading} onClick={() => load(nextCursor, false)}>
            {loading ? "Loading…" : "Load more"}
          </Button>
        </div>
      )}
    </div>
  )
}
