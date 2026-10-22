/** Hace que el navegador guarde un archivo recibido como Blob con el nombre indicado. */
export function guardarArchivo(blob: Blob, nombre: string, doc: Document = document): void {
  const url = URL.createObjectURL(blob);
  const enlace = doc.createElement('a');
  enlace.href = url;
  enlace.download = nombre;
  enlace.style.display = 'none';
  doc.body.appendChild(enlace);
  enlace.click();
  enlace.remove();
  // Safari en iOS lee el blob después de que la persona confirma la descarga: se libera al minuto
  setTimeout(() => URL.revokeObjectURL(url), 60_000);
}
