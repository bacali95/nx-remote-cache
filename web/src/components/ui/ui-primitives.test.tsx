// Covers the shadcn primitive variants/subcomponents that no page in the
// app currently wires up (e.g. DropdownMenu's checkbox/radio/submenu
// items, Table's caption/footer, Card's action slot, Badge's asChild
// mode) so the component library itself has real coverage independent of
// which pages happen to use which parts of it today.
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useState } from "react"
import { describe, expect, it } from "vitest"
import { Badge } from "@/components/ui/badge"
import {
  Card,
  CardAction,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogMedia,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableFooter,
  TableRow,
} from "@/components/ui/table"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuPortal,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

describe("Badge asChild", () => {
  it("renders as its child element instead of a span", () => {
    render(
      <Badge asChild>
        <a href="/x">link badge</a>
      </Badge>
    )
    const el = screen.getByText("link badge")
    expect(el.tagName).toBe("A")
  })
})

describe("Card unused slots", () => {
  it("renders CardAction and CardFooter", () => {
    render(
      <Card>
        <CardHeader>
          <CardTitle>Title</CardTitle>
          <CardAction>action</CardAction>
        </CardHeader>
        <CardContent>content</CardContent>
        <CardFooter>footer</CardFooter>
      </Card>
    )
    expect(screen.getByText("action")).toBeInTheDocument()
    expect(screen.getByText("footer")).toBeInTheDocument()
  })
})

describe("Table unused parts", () => {
  it("renders TableCaption and TableFooter", () => {
    render(
      <Table>
        <TableCaption>a caption</TableCaption>
        <TableBody>
          <TableRow>
            <TableCell>cell</TableCell>
          </TableRow>
        </TableBody>
        <TableFooter>
          <TableRow>
            <TableCell>footer cell</TableCell>
          </TableRow>
        </TableFooter>
      </Table>
    )
    expect(screen.getByText("a caption")).toBeInTheDocument()
    expect(screen.getByText("footer cell")).toBeInTheDocument()
  })
})

describe("Dialog unused parts", () => {
  it("closes via an explicit DialogClose and supports DialogFooter's built-in close button", async () => {
    const user = userEvent.setup()
    render(
      <Dialog>
        <DialogTrigger>Open</DialogTrigger>
        <DialogContent>
          <DialogClose>custom close</DialogClose>
          <DialogFooter showCloseButton>footer content</DialogFooter>
        </DialogContent>
      </Dialog>
    )
    await user.click(screen.getByText("Open"))
    expect(await screen.findByText("custom close")).toBeInTheDocument()
    expect(screen.getAllByRole("button", { name: "Close" })).toHaveLength(2)

    await user.click(screen.getByText("custom close"))
    await waitFor(() => expect(screen.queryByText("custom close")).not.toBeInTheDocument())
  })
})

describe("AlertDialogMedia", () => {
  it("renders inside an alert dialog", async () => {
    const user = userEvent.setup()
    render(
      <AlertDialog>
        <AlertDialogTrigger>Open</AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogMedia>icon</AlertDialogMedia>
        </AlertDialogContent>
      </AlertDialog>
    )
    await user.click(screen.getByText("Open"))
    expect(await screen.findByText("icon")).toBeInTheDocument()
  })
})

function FullDropdownMenu() {
  const [checked, setChecked] = useState(false)
  const [radio, setRadio] = useState("a")
  return (
    <DropdownMenu>
      <DropdownMenuTrigger>Open menu</DropdownMenuTrigger>
      <DropdownMenuPortal>
        <DropdownMenuContent>
          <DropdownMenuLabel>Menu label</DropdownMenuLabel>
          <DropdownMenuGroup>
            <DropdownMenuItem>
              Plain item
              <DropdownMenuShortcut>⌘K</DropdownMenuShortcut>
            </DropdownMenuItem>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuCheckboxItem checked={checked} onCheckedChange={setChecked}>
            Toggle me
          </DropdownMenuCheckboxItem>
          <DropdownMenuRadioGroup value={radio} onValueChange={setRadio}>
            <DropdownMenuRadioItem value="a">Option A</DropdownMenuRadioItem>
            <DropdownMenuRadioItem value="b">Option B</DropdownMenuRadioItem>
          </DropdownMenuRadioGroup>
          <DropdownMenuSub>
            <DropdownMenuSubTrigger>More</DropdownMenuSubTrigger>
            <DropdownMenuSubContent>
              <DropdownMenuItem>Nested item</DropdownMenuItem>
            </DropdownMenuSubContent>
          </DropdownMenuSub>
        </DropdownMenuContent>
      </DropdownMenuPortal>
    </DropdownMenu>
  )
}

describe("DropdownMenu subcomponents not used by any page", () => {
  it("renders label, group, separator, shortcut, checkbox item, and radio group", async () => {
    const user = userEvent.setup()
    render(<FullDropdownMenu />)

    await user.click(screen.getByText("Open menu"))
    expect(await screen.findByText("Menu label")).toBeInTheDocument()
    expect(screen.getByText("Plain item")).toBeInTheDocument()
    expect(screen.getByText("⌘K")).toBeInTheDocument()
    expect(screen.getByText("Toggle me")).toBeInTheDocument()
    expect(screen.getByText("Option A")).toBeInTheDocument()
    expect(screen.getByText("Option B")).toBeInTheDocument()
  })

  it("toggles the checkbox item (which also closes the menu, like a normal item click)", async () => {
    const user = userEvent.setup()
    render(<FullDropdownMenu />)

    await user.click(screen.getByText("Open menu"))
    await user.click(await screen.findByText("Toggle me"))
    await waitFor(() => expect(screen.queryByText("Toggle me")).not.toBeInTheDocument())
  })

  it("opens the submenu on hover and shows its nested item", async () => {
    const user = userEvent.setup()
    render(<FullDropdownMenu />)

    await user.click(screen.getByText("Open menu"))
    await user.hover(await screen.findByText("More"))
    expect(await screen.findByText("Nested item")).toBeInTheDocument()
  })
})
