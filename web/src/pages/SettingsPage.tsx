import { useEffect, useState } from "react"
import { toast } from "sonner"
import { api, ApiError, type Settings, type StorageBackendType, type UpdateSettingsRequest } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

// A secret's local state: `undefined` means "user hasn't touched this
// field, leave whatever's already configured alone." Typing (even
// clearing to "") flips it to a real string, which is sent as-is —
// including "" to explicitly remove a credential.
function SecretField({
  id,
  label,
  isSet,
  value,
  onChange,
}: {
  id: string
  label: string
  isSet: boolean
  value: string | undefined
  onChange: (v: string | undefined) => void
}) {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2">
        <Label htmlFor={id}>{label}</Label>
        <Badge variant={isSet ? "default" : "secondary"} className="text-xs">
          {isSet ? "configured" : "not set"}
        </Badge>
      </div>
      <div className="flex gap-2">
        <Input
          id={id}
          type="password"
          placeholder={isSet ? "Leave blank to keep current value" : "Not configured"}
          value={value ?? ""}
          onChange={(e) => onChange(e.target.value)}
        />
        {isSet && (
          <Button type="button" variant="outline" size="sm" onClick={() => onChange("")}>
            Clear
          </Button>
        )}
      </div>
    </div>
  )
}

export function SettingsPage() {
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [current, setCurrent] = useState<Settings | null>(null)

  const [backend, setBackend] = useState<StorageBackendType>("local")
  const [localDir, setLocalDir] = useState("")
  const [s3Bucket, setS3Bucket] = useState("")
  const [s3Region, setS3Region] = useState("")
  const [s3Prefix, setS3Prefix] = useState("")
  const [s3Endpoint, setS3Endpoint] = useState("")
  const [s3UsePathStyle, setS3UsePathStyle] = useState(false)
  const [s3AccessKeyId, setS3AccessKeyId] = useState<string | undefined>(undefined)
  const [s3SecretAccessKey, setS3SecretAccessKey] = useState<string | undefined>(undefined)
  const [gcsBucket, setGcsBucket] = useState("")
  const [gcsPrefix, setGcsPrefix] = useState("")
  const [gcsCredentialsJson, setGcsCredentialsJson] = useState<string | undefined>(undefined)
  const [sessionTtlHours, setSessionTtlHours] = useState("24")
  const [maxEntryMb, setMaxEntryMb] = useState("500")

  function applyToForm(s: Settings) {
    setCurrent(s)
    setBackend(s.storageBackend)
    setLocalDir(s.localDir)
    setS3Bucket(s.s3Bucket)
    setS3Region(s.s3Region)
    setS3Prefix(s.s3Prefix)
    setS3Endpoint(s.s3Endpoint)
    setS3UsePathStyle(s.s3UsePathStyle)
    setS3AccessKeyId(undefined)
    setS3SecretAccessKey(undefined)
    setGcsBucket(s.gcsBucket)
    setGcsPrefix(s.gcsPrefix)
    setGcsCredentialsJson(undefined)
    setSessionTtlHours(String(Math.round(s.sessionTtlSeconds / 3600)))
    setMaxEntryMb(String(Math.round(s.maxCacheEntryBytes / (1024 * 1024))))
  }

  async function load() {
    setLoading(true)
    try {
      const s = await api.get<Settings>("/settings")
      applyToForm(s)
    } catch {
      toast.error("Failed to load settings")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  async function save() {
    const ttlHours = Number(sessionTtlHours)
    const maxMb = Number(maxEntryMb)
    if (!Number.isFinite(ttlHours) || ttlHours <= 0) {
      toast.error("Session TTL must be a positive number of hours")
      return
    }
    if (!Number.isFinite(maxMb) || maxMb <= 0) {
      toast.error("Max cache entry size must be a positive number of MB")
      return
    }

    const req: UpdateSettingsRequest = {
      storageBackend: backend,
      localDir,
      s3Bucket,
      s3Region,
      s3Prefix,
      s3Endpoint,
      s3UsePathStyle,
      s3AccessKeyId,
      s3SecretAccessKey,
      gcsBucket,
      gcsPrefix,
      gcsCredentialsJson,
      sessionTtlSeconds: Math.round(ttlHours * 3600),
      maxCacheEntryBytes: Math.round(maxMb * 1024 * 1024),
    }

    setSaving(true)
    try {
      const updated = await api.put<Settings>("/settings", req)
      applyToForm(updated)
      toast.success("Settings saved and applied — no restart needed")
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to save settings")
    } finally {
      setSaving(false)
    }
  }

  if (loading || !current) {
    return <p className="text-muted-foreground">Loading settings…</p>
  }

  return (
    <div className="flex max-w-2xl flex-col gap-6">
      <div>
        <h1 className="text-xl font-semibold">Settings</h1>
        <p className="text-sm text-muted-foreground">
          Changes apply immediately to the running server — no restart or redeploy.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Storage backend</CardTitle>
          <CardDescription>Where cache artifacts are stored. Switching backends is tested before it's applied.</CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs value={backend} onValueChange={(v) => setBackend(v as StorageBackendType)}>
            <TabsList>
              <TabsTrigger value="local">Local disk</TabsTrigger>
              <TabsTrigger value="s3">S3</TabsTrigger>
              <TabsTrigger value="gcs">GCS</TabsTrigger>
            </TabsList>

            <TabsContent value="local" className="flex flex-col gap-4 pt-4">
              <div className="flex flex-col gap-2">
                <Label htmlFor="local-dir">Directory</Label>
                <Input id="local-dir" value={localDir} onChange={(e) => setLocalDir(e.target.value)} />
              </div>
            </TabsContent>

            <TabsContent value="s3" className="flex flex-col gap-4 pt-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="flex flex-col gap-2">
                  <Label htmlFor="s3-bucket">Bucket</Label>
                  <Input id="s3-bucket" value={s3Bucket} onChange={(e) => setS3Bucket(e.target.value)} />
                </div>
                <div className="flex flex-col gap-2">
                  <Label htmlFor="s3-region">Region</Label>
                  <Input id="s3-region" value={s3Region} onChange={(e) => setS3Region(e.target.value)} />
                </div>
                <div className="flex flex-col gap-2">
                  <Label htmlFor="s3-prefix">Prefix (optional)</Label>
                  <Input id="s3-prefix" value={s3Prefix} onChange={(e) => setS3Prefix(e.target.value)} />
                </div>
                <div className="flex flex-col gap-2">
                  <Label htmlFor="s3-endpoint">Endpoint (R2/MinIO, optional)</Label>
                  <Input id="s3-endpoint" value={s3Endpoint} onChange={(e) => setS3Endpoint(e.target.value)} />
                </div>
              </div>
              <div className="flex items-center gap-2">
                <Checkbox
                  id="s3-path-style"
                  checked={s3UsePathStyle}
                  onCheckedChange={(v) => setS3UsePathStyle(v === true)}
                />
                <Label htmlFor="s3-path-style">Use path-style addressing (required for MinIO)</Label>
              </div>
              <Separator />
              <SecretField
                id="s3-access-key-id"
                label="Access key ID"
                isSet={current.s3AccessKeyIdSet}
                value={s3AccessKeyId}
                onChange={setS3AccessKeyId}
              />
              <SecretField
                id="s3-secret-access-key"
                label="Secret access key"
                isSet={current.s3SecretAccessKeySet}
                value={s3SecretAccessKey}
                onChange={setS3SecretAccessKey}
              />
              <p className="text-xs text-muted-foreground">
                Leave both blank to use the AWS default credential chain (IAM role, env vars on the host, etc.)
                instead of static credentials.
              </p>
            </TabsContent>

            <TabsContent value="gcs" className="flex flex-col gap-4 pt-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="flex flex-col gap-2">
                  <Label htmlFor="gcs-bucket">Bucket</Label>
                  <Input id="gcs-bucket" value={gcsBucket} onChange={(e) => setGcsBucket(e.target.value)} />
                </div>
                <div className="flex flex-col gap-2">
                  <Label htmlFor="gcs-prefix">Prefix (optional)</Label>
                  <Input id="gcs-prefix" value={gcsPrefix} onChange={(e) => setGcsPrefix(e.target.value)} />
                </div>
              </div>
              <Separator />
              <SecretField
                id="gcs-credentials"
                label="Service account key (JSON)"
                isSet={current.gcsCredentialsSet}
                value={gcsCredentialsJson}
                onChange={setGcsCredentialsJson}
              />
              <p className="text-xs text-muted-foreground">
                Leave blank to use Application Default Credentials (workload identity, gcloud ADC, etc.) instead
                of a static key.
              </p>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>General</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <Label htmlFor="session-ttl">Admin session lifetime (hours)</Label>
            <Input
              id="session-ttl"
              type="number"
              min={1}
              value={sessionTtlHours}
              onChange={(e) => setSessionTtlHours(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="max-entry">Max cache entry size (MB)</Label>
            <Input
              id="max-entry"
              type="number"
              min={1}
              value={maxEntryMb}
              onChange={(e) => setMaxEntryMb(e.target.value)}
            />
          </div>
        </CardContent>
      </Card>

      <div>
        <Button onClick={save} disabled={saving}>
          {saving ? "Saving…" : "Save settings"}
        </Button>
      </div>
    </div>
  )
}
