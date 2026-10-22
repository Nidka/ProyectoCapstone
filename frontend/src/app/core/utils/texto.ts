/** Minúsculas y sin tildes, para comparar búsquedas: «Logística» y «logistica» coinciden. */
export function normalizar(texto: string | null | undefined): string {
  return (texto ?? '').normalize('NFD').replace(/\p{M}/gu, '').toLowerCase();
}
