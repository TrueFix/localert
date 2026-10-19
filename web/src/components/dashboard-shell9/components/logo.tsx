export function Logo() {
  return <a href="#overview" className="flex gap-3" aria-label="Localert home"><img src="/favicon.svg" className="size-8 rounded-md" alt="Localert" /><div data-logo="description" className="grid flex-1 text-left text-sm leading-tight md:hidden"><span className="truncate font-medium">Localert</span><span className="truncate text-xs">Your notification nook</span></div></a>
}
