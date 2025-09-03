export function getCSRFToken() {
  const csrfTokenCookie = document.cookie
    .split('; ')
    .find(row => row.startsWith('nodo_csrf_token='))
  if (!csrfTokenCookie) {
    return null
  }
  return csrfTokenCookie.split('=')[1]
}
