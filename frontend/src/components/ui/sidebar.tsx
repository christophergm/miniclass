import { createContext, type ReactNode, useContext, useEffect, useRef, useState } from "react";

import { cn } from "@/lib/utils";

type SidebarContextValue = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

const SidebarContext = createContext<SidebarContextValue | null>(null);

/**
 * A local implementation of the shadcn sidebar shell. Consumers own its open
 * state, which keeps navigation state outside the component and avoids both
 * the global keyboard shortcut and persisted open-state cookie.
 */
export function SidebarProvider({
  children,
  open,
  onOpenChange,
}: {
  children: ReactNode;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <SidebarContext.Provider value={{ open, onOpenChange }}>
      <div className="flex min-w-0">{children}</div>
    </SidebarContext.Provider>
  );
}

function useSidebar() {
  const context = useContext(SidebarContext);
  if (!context) throw new Error("Sidebar must be rendered inside SidebarProvider");
  return context;
}

function useSidebarSheet() {
  const [isSheet, setIsSheet] = useState(
    () =>
      typeof window === "undefined" ||
      !window.matchMedia ||
      !window.matchMedia("(min-width: 1280px)").matches,
  );

  useEffect(() => {
    if (!window.matchMedia) return;
    const media = window.matchMedia("(min-width: 1280px)");
    const update = () => setIsSheet(!media.matches);
    update();
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, []);

  return isSheet;
}

export function Sidebar({
  "aria-label": ariaLabel,
  children,
  className,
  side = "left",
}: {
  "aria-label": string;
  children: ReactNode;
  className?: string;
  side?: "left" | "right";
}) {
  const { open, onOpenChange } = useSidebar();
  const sidebarRef = useRef<HTMLElement>(null);
  const onOpenChangeRef = useRef(onOpenChange);
  const isSheet = useSidebarSheet();

  useEffect(() => {
    onOpenChangeRef.current = onOpenChange;
  }, [onOpenChange]);

  useEffect(() => {
    if (!open) return;
    sidebarRef.current?.focus();
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        onOpenChangeRef.current(false);
      }
    };
    document.addEventListener("keydown", closeOnEscape);
    return () => document.removeEventListener("keydown", closeOnEscape);
  }, [open]);

  if (!open) return null;

  const sidebar = (
    <aside
      aria-label={ariaLabel}
      className={cn(
        "h-full w-full max-w-[400px] border-l bg-background shadow-xl min-[1280px]:w-[400px] min-[1280px]:max-w-none min-[1280px]:shadow-none",
        side === "left" && "border-r border-l-0",
        className,
      )}
      ref={sidebarRef}
      tabIndex={-1}
    >
      {children}
    </aside>
  );

  if (!isSheet) return sidebar;

  return (
    <div
      aria-label={ariaLabel}
      aria-modal="true"
      className="fixed inset-0 z-50 flex justify-end bg-black/50"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onOpenChange(false);
      }}
      role="dialog"
    >
      {sidebar}
    </div>
  );
}

export function SidebarContent({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return <div className={cn("flex h-full min-h-0 flex-col", className)}>{children}</div>;
}
