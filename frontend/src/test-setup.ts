/*
 * Node 25 expone un `localStorage` global sin métodos (sin --localstorage-file) que tapa al de jsdom.
 * En las pruebas se reemplaza por un almacenamiento en memoria con la API Storage completa.
 */
if (typeof globalThis.localStorage?.setItem !== 'function') {
  const datos = new Map<string, string>();
  const almacen: Storage = {
    get length() {
      return datos.size;
    },
    clear: () => datos.clear(),
    getItem: clave => datos.get(clave) ?? null,
    key: indice => [...datos.keys()][indice] ?? null,
    removeItem: clave => {
      datos.delete(clave);
    },
    setItem: (clave, valor) => {
      datos.set(clave, String(valor));
    },
  };
  Object.defineProperty(globalThis, 'localStorage', { value: almacen, configurable: true, writable: true });
}
