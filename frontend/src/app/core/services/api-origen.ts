/**
 * URL pública de la API desplegada, sin barra final. Si el servicio de Render toma otro nombre
 * (o se despliega en otro servidor), cambia solo esta línea y vuelve a compilar el frontend.
 */
export const API_PRODUCCION = 'https://lizar-gastos-api.onrender.com';

/** En desarrollo local el frontend habla con la API del puerto 8081. */
export const API_LOCAL = 'http://localhost:8081';

/** Origen de la API según dónde corre el frontend: local en localhost, la desplegada en cualquier otro dominio. */
export function origenApi(hostname: string = window.location.hostname): string {
  return ['localhost', '127.0.0.1'].includes(hostname) ? API_LOCAL : API_PRODUCCION;
}
