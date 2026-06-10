// Bridges non-React code (the axios interceptor) to React Router navigation.
//
// The interceptor can't call useNavigate (it isn't a component), so a component
// rendered inside the Router registers the navigate function here via
// setNavigator, and the interceptor calls redirectToLogin(). This keeps SPA
// state/history intact instead of doing a full document reload.

type NavigateFn = (path: string) => void;

let navigator: NavigateFn | null = null;

export function setNavigator(fn: NavigateFn | null): void {
  navigator = fn;
}

export function redirectToLogin(): void {
  if (navigator) {
    navigator('/login');
    return;
  }
  // Fallback if a redirect happens before the app registered a navigator
  // (e.g. an error during the very first render).
  window.location.href = '/login';
}
