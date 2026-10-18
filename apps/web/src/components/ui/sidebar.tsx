'use client';

import { mergeProps } from '@base-ui/react/merge-props';
import { useRender } from '@base-ui/react/use-render';
import { createContext, useContext, useEffect, useMemo, useState } from 'react';
import type { ComponentProps, CSSProperties } from 'react';

import { useIsMobile } from '../../hooks/use-mobile';
import { cn } from '../../styles/class-names';
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from './sheet';

declare module 'react' {
    interface CSSProperties {
        '--sidebar-width'?: string;
        '--sidebar-width-icon'?: string;
    }
}

const WIDTHS: CSSProperties = { '--sidebar-width': '16rem', '--sidebar-width-icon': '3rem' };
const MOBILE_WIDTH: CSSProperties = { '--sidebar-width': '18rem' };
const TOGGLE_KEY = 'b';

interface SidebarState {
    open: boolean;
    openMobile: boolean;
    setOpenMobile: (open: boolean) => void;
    isMobile: boolean;
}

const SidebarContext = createContext<null | SidebarState>(null);

function useSidebar(): SidebarState {
    const context = useContext(SidebarContext);
    if (context === null) {
        throw new Error('useSidebar must be used within a SidebarProvider.');
    }
    return context;
}

function useToggleShortcut(isMobile: boolean, toggle: (mobile: boolean) => void): void {
    useEffect(() => {
        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === TOGGLE_KEY && (event.metaKey || event.ctrlKey)) {
                event.preventDefault();
                toggle(isMobile);
            }
        };
        globalThis.addEventListener('keydown', handleKeyDown);
        return () => {
            globalThis.removeEventListener('keydown', handleKeyDown);
        };
    }, [isMobile, toggle]);
}

/** The sidebar's state (open on a desktop, a sheet on a phone), toggled by ⌘B. */
function SidebarProvider({ className, style, children, ...props }: ComponentProps<'div'>) {
    const isMobile = useIsMobile();
    const [open, setOpen] = useState(true);
    const [openMobile, setOpenMobile] = useState(false);
    const toggle = useMemo(
        () => (mobile: boolean) => {
            if (mobile) {
                setOpenMobile((previous) => !previous);
            } else {
                setOpen((previous) => !previous);
            }
        },
        [],
    );
    useToggleShortcut(isMobile, toggle);
    const state = useMemo(
        () => ({ isMobile, open, openMobile, setOpenMobile }),
        [isMobile, open, openMobile],
    );
    const widths: CSSProperties = { ...WIDTHS, ...style };

    return (
        <SidebarContext.Provider value={state}>
            <div
                className={cn(
                    'group/sidebar-wrapper has-data-[variant=inset]:bg-sidebar flex h-full w-full',
                    className,
                )}
                data-slot="sidebar-wrapper"
                style={widths}
                {...props}
            >
                {children}
            </div>
        </SidebarContext.Provider>
    );
}

function Sidebar({ className, children, dir, ...props }: ComponentProps<'div'>) {
    const { isMobile, open, openMobile, setOpenMobile } = useSidebar();

    if (isMobile) {
        return (
            <Sheet onOpenChange={setOpenMobile} open={openMobile} {...props}>
                <SheetContent
                    className="bg-sidebar text-sidebar-foreground w-(--sidebar-width) p-0 [&>button]:hidden"
                    data-mobile="true"
                    data-sidebar="sidebar"
                    data-slot="sidebar"
                    dir={dir}
                    side="left"
                    style={MOBILE_WIDTH}
                >
                    <SheetHeader className="sr-only">
                        <SheetTitle>Sidebar</SheetTitle>
                        <SheetDescription>Displays the mobile sidebar.</SheetDescription>
                    </SheetHeader>
                    <div className="flex h-full w-full flex-col">{children}</div>
                </SheetContent>
            </Sheet>
        );
    }

    return (
        <div
            className="group peer text-sidebar-foreground hidden h-full shrink-0 md:flex"
            data-collapsible={open ? '' : 'offcanvas'}
            data-side="left"
            data-slot="sidebar"
            data-state={open ? 'expanded' : 'collapsed'}
            data-variant="sidebar"
        >
            <div
                className={cn(
                    'z-10 hidden h-full w-(--sidebar-width) shrink-0 transition-[width] duration-200 ease-linear md:flex',
                    'group-data-[collapsible=offcanvas]:w-0 group-data-[collapsible=offcanvas]:overflow-hidden',
                    'group-data-[collapsible=icon]:w-(--sidebar-width-icon) group-data-[side=left]:border-r group-data-[side=right]:border-l',
                    className,
                )}
                data-side="left"
                data-slot="sidebar-container"
                {...props}
            >
                <div
                    className="bg-sidebar group-data-[variant=floating]:ring-sidebar-border flex size-full flex-col backdrop-blur-xl group-data-[variant=floating]:rounded-lg group-data-[variant=floating]:shadow-sm group-data-[variant=floating]:ring-1"
                    data-sidebar="sidebar"
                    data-slot="sidebar-inner"
                >
                    {children}
                </div>
            </div>
        </div>
    );
}

function SidebarInset({ className, ...props }: ComponentProps<'main'>) {
    return (
        <main
            className={cn(
                'bg-background relative flex min-h-0 w-full flex-1 flex-col overflow-y-auto md:peer-data-[variant=inset]:m-2 md:peer-data-[variant=inset]:ml-0 md:peer-data-[variant=inset]:rounded-xl md:peer-data-[variant=inset]:shadow-sm md:peer-data-[variant=inset]:peer-data-[state=collapsed]:ml-2',
                className,
            )}
            data-slot="sidebar-inset"
            {...props}
        />
    );
}

function SidebarHeader({ className, ...props }: ComponentProps<'div'>) {
    return (
        <div
            className={cn('flex flex-col gap-2 p-2', className)}
            data-sidebar="header"
            data-slot="sidebar-header"
            {...props}
        />
    );
}

function SidebarFooter({ className, ...props }: ComponentProps<'div'>) {
    return (
        <div
            className={cn('flex flex-col gap-2 p-2', className)}
            data-sidebar="footer"
            data-slot="sidebar-footer"
            {...props}
        />
    );
}

function SidebarContent({ className, ...props }: ComponentProps<'div'>) {
    return (
        <div
            className={cn(
                'no-scrollbar flex min-h-0 flex-1 flex-col gap-0 overflow-auto group-data-[collapsible=icon]:overflow-hidden',
                className,
            )}
            data-sidebar="content"
            data-slot="sidebar-content"
            {...props}
        />
    );
}

function SidebarGroup({ className, ...props }: ComponentProps<'div'>) {
    return (
        <div
            className={cn('relative flex w-full min-w-0 flex-col p-2', className)}
            data-sidebar="group"
            data-slot="sidebar-group"
            {...props}
        />
    );
}

function SidebarGroupLabel({
    className,
    render,
    ...props
}: ComponentProps<'div'> & useRender.ComponentProps<'div'>) {
    return useRender({
        defaultTagName: 'div',
        props: mergeProps<'div'>(
            {
                className: cn(
                    'text-sidebar-foreground/70 ring-sidebar-ring flex h-8 shrink-0 items-center rounded-md px-2 text-xs font-medium outline-hidden transition-[margin,opacity] duration-200 ease-linear group-data-[collapsible=icon]:-mt-8 group-data-[collapsible=icon]:opacity-0 focus-visible:ring-2 [&>svg]:size-4 [&>svg]:shrink-0',
                    className,
                ),
            },
            props,
        ),
        render,
        state: {
            slot: 'sidebar-group-label',
            sidebar: 'group-label',
        },
    });
}

function SidebarMenu({ className, ...props }: ComponentProps<'ul'>) {
    return (
        <ul
            className={cn('flex w-full min-w-0 flex-col gap-0', className)}
            data-sidebar="menu"
            data-slot="sidebar-menu"
            {...props}
        />
    );
}

function SidebarMenuItem({ className, ...props }: ComponentProps<'li'>) {
    return (
        <li
            className={cn('group/menu-item relative', className)}
            data-sidebar="menu-item"
            data-slot="sidebar-menu-item"
            {...props}
        />
    );
}

const MENU_BUTTON =
    'peer/menu-button group/menu-button ring-sidebar-ring hover:bg-sidebar-accent hover:text-sidebar-accent-foreground active:bg-sidebar-accent active:text-sidebar-accent-foreground data-open:hover:bg-sidebar-accent data-open:hover:text-sidebar-accent-foreground data-active:bg-sidebar-accent data-active:text-sidebar-accent-foreground flex h-8 w-full items-center gap-2 overflow-hidden rounded-md p-2 text-left text-sm outline-hidden transition-[width,height,padding] group-has-data-[sidebar=menu-action]/menu-item:pr-8 group-data-[collapsible=icon]:size-8! group-data-[collapsible=icon]:p-2! focus-visible:ring-2 disabled:pointer-events-none disabled:opacity-50 aria-disabled:pointer-events-none aria-disabled:opacity-50 data-active:font-medium [&_svg]:size-4 [&_svg]:shrink-0 [&>span:last-child]:truncate';

function SidebarMenuButton({
    render,
    isActive = false,
    className,
    ...props
}: ComponentProps<'button'> & useRender.ComponentProps<'button'> & { isActive?: boolean }) {
    return useRender({
        defaultTagName: 'button',
        props: mergeProps<'button'>({ className: cn(MENU_BUTTON, className) }, props),
        render,
        state: {
            active: isActive,
            sidebar: 'menu-button',
            size: 'default',
            slot: 'sidebar-menu-button',
        },
    });
}

export {
    Sidebar,
    SidebarContent,
    SidebarFooter,
    SidebarGroup,
    SidebarGroupLabel,
    SidebarHeader,
    SidebarInset,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    SidebarProvider,
};
