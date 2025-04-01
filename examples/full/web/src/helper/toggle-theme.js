// Toggle dark theme on 'ry:toogleTheme' event, which can be send by some hyperscipt.
document.documentElement.addEventListener('ry:toggleTheme', function(event) {
    let curr = document.documentElement.dataset.theme;
    let preferDark = (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches);
    let want = '';

    if (curr == 'dark') {
        want = 'light';
    } else if (curr == 'light') {
        want = 'dark';
    } else if (!curr && preferDark) {
        want = 'light';
    } else {
        want = 'dark';
    }

    localStorage.setItem('bulma-theme', want);
    document.documentElement.dataset.theme = want;
});

// Activate theme, which was set in a previous session on startup.
{
    let want = localStorage.getItem('bulma-theme');
    if(want) {
        document.documentElement.dataset.theme = want;
    }
}
