import * as React from "react"
import { flexRender } from "@tanstack/react-table"
import { type LegacyColumnDef as ColumnDef, getCoreRowModel, getPaginationRowModel, getSortedRowModel, useLegacyTable as useReactTable } from "@tanstack/react-table/legacy"
import { ChevronLeft, ChevronRight, KeyRoundIcon, MoreHorizontal, SearchIcon, Trash2Icon } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { CreateApiKeyDialog } from "./create-api-key-dialog"
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group"
import { api, errorMessage, formatDate, type AppKey } from "@/lib/api"
import { useLocalert } from "@/lib/context"

export function ApiKeysDataTable({ data }: { data: AppKey[] }) {
  const [search, setSearch] = React.useState("")
  const [revoking, setRevoking] = React.useState<AppKey | null>(null)
  const [busy, setBusy] = React.useState(false)
  const { refresh } = useLocalert()
  const columns = React.useMemo<ColumnDef<AppKey>[]>(() => [
    { accessorKey: "name", header: "Application", cell: ({ row }) => <div className="flex min-w-0 items-center gap-2.5" title={row.original.name}><span className="bg-muted flex size-8 shrink-0 items-center justify-center rounded-lg"><KeyRoundIcon className="text-muted-foreground size-3.5" /></span><span className="truncate font-medium">{row.original.name}</span></div> },
    { accessorKey: "prefix", header: "API key", size: 190, cell: ({ row }) => <span className="text-muted-foreground font-mono text-xs">{row.original.prefix}</span> },
    { accessorKey: "last_used_at", header: "Last used", size: 160, cell: ({ row }) => <span className="text-muted-foreground whitespace-nowrap text-xs">{formatDate(row.original.last_used_at)}</span> },
    { accessorKey: "sent", header: "Sent", size: 70, cell: ({ row }) => <span className="tabular-nums">{row.original.sent.toLocaleString()}</span> },
    { id: "actions", size: 56, cell: ({ row }) => <div className="text-end"><DropdownMenu><DropdownMenuTrigger asChild><Button variant="ghost" size="icon-sm" aria-label={`Actions for ${row.original.name}`}><MoreHorizontal /></Button></DropdownMenuTrigger><DropdownMenuContent align="end"><DropdownMenuLabel>{row.original.name}</DropdownMenuLabel><DropdownMenuItem variant="destructive" onClick={() => setRevoking(row.original)}><Trash2Icon />Revoke key</DropdownMenuItem></DropdownMenuContent></DropdownMenu></div> },
  ], [])
  const rows = React.useMemo(() => data.filter(key => `${key.name} ${key.prefix}`.toLowerCase().includes(search.trim().toLowerCase())), [data, search])
  const table = useReactTable({ data: rows, columns, getCoreRowModel: getCoreRowModel(), getPaginationRowModel: getPaginationRowModel(), getSortedRowModel: getSortedRowModel(), getRowId: row => row.id })
  const { pageIndex, pageSize } = table.getState().pagination
  const firstRow = rows.length === 0 ? 0 : pageIndex * pageSize + 1
  const lastRow = Math.min((pageIndex + 1) * pageSize, rows.length)

  async function revoke() {
    if (!revoking) return
    setBusy(true)
    try { await api(`/api/admin/keys/${revoking.id}`, { method: "DELETE" }); setRevoking(null); toast.success("App key revoked"); await refresh() }
    catch (error) { toast.error(errorMessage(error)) }
    finally { setBusy(false) }
  }

  return <>
    <Card className="@container/card gap-0 py-0">
      <CardContent className="flex flex-col p-0 [&_[data-slot=table-container]]:flex-1">
        <div className="flex min-h-14 flex-wrap items-center gap-3 border-b px-(--card-spacing) py-3">
          <InputGroup className="w-full max-w-56 shrink-0 @max-md/card:min-w-24 @max-md/card:flex-1 @max-md/card:basis-0">
            <InputGroupInput aria-label="Search app keys" placeholder="Search applications…" value={search} onChange={event => { setSearch(event.target.value); table.setPageIndex(0) }} />
            <InputGroupAddon><SearchIcon /></InputGroupAddon>
          </InputGroup>
          <div className="ms-auto shrink-0"><CreateApiKeyDialog /></div>
        </div>
        <Table className="min-w-[680px] table-fixed [&_td:first-child]:ps-(--card-spacing) [&_td:last-child]:pe-(--card-spacing) [&_th:first-child]:ps-(--card-spacing) [&_th:last-child]:pe-(--card-spacing)">
          <TableHeader>{table.getHeaderGroups().map(group => <TableRow key={group.id}>{group.headers.map(header => <TableHead key={header.id} style={{ width: header.getSize() !== 150 ? header.getSize() : undefined }}>{header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}</TableHead>)}</TableRow>)}</TableHeader>
          <TableBody>{table.getRowModel().rows.length ? table.getRowModel().rows.map(row => <TableRow key={row.id}>{row.getVisibleCells().map(cell => <TableCell key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</TableCell>)}</TableRow>) : <TableRow><TableCell colSpan={columns.length} className="h-24 text-center text-muted-foreground">{search ? "No applications match your search." : "No apps connected yet. Create your first app key to get started."}</TableCell></TableRow>}</TableBody>
        </Table>
        <div className="flex items-center justify-end gap-2 border-t px-(--card-spacing) py-3">
          <div className="text-muted-foreground flex-1 text-sm tabular-nums">{firstRow} – {lastRow} of {rows.length} apps</div>
          <div className="flex items-center gap-1"><Button variant="outline" size="icon-sm" onClick={() => table.previousPage()} disabled={!table.getCanPreviousPage()} aria-label="Previous page"><ChevronLeft /></Button><Button variant="outline" size="icon-sm" onClick={() => table.nextPage()} disabled={!table.getCanNextPage()} aria-label="Next page"><ChevronRight /></Button></div>
        </div>
      </CardContent>
    </Card>
    <Dialog open={!!revoking} onOpenChange={open => { if (!open && !busy) setRevoking(null) }}><DialogContent className="sm:max-w-[425px]"><DialogHeader><DialogTitle>Revoke this app key?</DialogTitle><DialogDescription>{revoking?.name} will immediately lose access to the notification API. Its delivery history will remain available.</DialogDescription></DialogHeader><DialogFooter><Button variant="outline" disabled={busy} onClick={() => setRevoking(null)}>Cancel</Button><Button variant="destructive" disabled={busy} onClick={() => void revoke()}>{busy ? "Revoking…" : "Revoke key"}</Button></DialogFooter></DialogContent></Dialog>
  </>
}
