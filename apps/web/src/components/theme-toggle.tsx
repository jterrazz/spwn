'use client';

import { IconMoonFilled, IconSunFilled } from '@tabler/icons-react';
import { useTheme } from 'next-themes';
import { useEffect, useState } from 'react';

export function ThemeToggle() {
    const { theme, setTheme } = useTheme();
    const [mounted, setMounted] = useState(false);
    useEffect(() => setMounted(true), []);

    if (!mounted) {
        return <div className="w-8 h-8" />;
    }

    const isDark = theme === 'dark';

    return (
        <button
            aria-label="Toggle theme"
            className="w-8 h-8 flex items-center justify-center rounded-full text-muted-foreground/30 hover:text-foreground transition-colors"
            onClick={() => setTheme(isDark ? 'light' : 'dark')}
        >
            {isDark ? <IconMoonFilled size={15} /> : <IconSunFilled size={15} />}
        </button>
    );
}
