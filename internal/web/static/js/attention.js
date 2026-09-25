// Delegated handlers also work after HTMX replaces the current-weather partial.
(() => {
    let active = null;
    let pinned = false;
    let closeTimer;
    let dismissed = null;

    function hide() {
        clearTimeout(closeTimer);
        if (active) {
            active.querySelector('[role="tooltip"]').hidden = true;
            active.querySelector('[data-attention-trigger]').setAttribute('aria-expanded', 'false');
        }
        active = null;
        pinned = false;
    }

    function show(signal) {
        clearTimeout(closeTimer);
        if (active !== signal) hide();
        active = signal;
        const button = signal.querySelector('[data-attention-trigger]');
        const hint = signal.querySelector('[role="tooltip"]');
        hint.hidden = false;
        button.setAttribute('aria-expanded', 'true');
        const rect = button.getBoundingClientRect();
        hint.style.maxHeight = '';
        const width = hint.offsetWidth;
        const height = hint.offsetHeight;
        hint.style.left = `${Math.max(16, Math.min(rect.left, window.innerWidth - width - 16))}px`;
        const below = Math.max(0, window.innerHeight - rect.bottom - 8);
        const above = Math.max(0, rect.top - 8);
        const useBelow = height <= below || below >= above;
        hint.style.maxHeight = `${useBelow ? below : above}px`;
        const top = useBelow ? rect.bottom : Math.max(8, rect.top - hint.offsetHeight);
        hint.style.top = `${top}px`;
    }

    document.addEventListener('pointerover', event => {
        if (event.pointerType !== 'mouse') return;
        const signal = event.target.closest('.ui-attention-signal');
        if (signal && signal !== dismissed) show(signal);
    });
    document.addEventListener('pointerout', event => {
        const signal = event.target.closest('.ui-attention-signal');
        if (!signal || signal.contains(event.relatedTarget)) return;
        if (dismissed === signal) dismissed = null;
        if (active === signal && !pinned && !signal.contains(document.activeElement)) {
            // Leave enough time to cross from the icon to its hoverable hint.
            closeTimer = setTimeout(hide, 120);
        }
    });
    document.addEventListener('focusin', event => {
        const button = event.target.closest('[data-attention-trigger]');
        if (button) {
            dismissed = null;
            show(button.closest('.ui-attention-signal'));
        } else hide();
    });
    document.addEventListener('focusout', event => {
        if (active && active.contains(event.target) && !active.contains(event.relatedTarget)) hide();
    });
    document.addEventListener('click', event => {
        const button = event.target.closest('[data-attention-trigger]');
        if (button) {
            const signal = button.closest('.ui-attention-signal');
            if (active === signal && pinned) {
                dismissed = signal;
                hide();
            } else {
                dismissed = null;
                show(signal);
                pinned = true;
            }
        } else if (!active || !active.contains(event.target)) hide();
    });
    document.addEventListener('keydown', event => {
        if (event.key === 'Escape' && active) {
            dismissed = active;
            hide();
        }
    });
    window.addEventListener('resize', hide);
    document.addEventListener('scroll', event => {
        if (!active || !active.querySelector('[role="tooltip"]').contains(event.target)) hide();
    }, true);
    document.addEventListener('htmx:beforeSwap', event => {
        if (active && event.detail.target.contains(active)) hide();
        dismissed = null;
    });
})();
