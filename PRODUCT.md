# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

- **Logística en oficina (computadora de escritorio).** Registra los gastos adicionales de cada viaje, revisa los comprobantes de pago y los aprueba o rechaza. En el sistema actúa como "Encargado de Logística".
- **Conductores u operadores en ruta (celular).** Suben el comprobante de pago de un gasto (foto o PDF) desde el teléfono y consultan en qué estado está.
- Roles de cuenta existentes: `admin` y `operador`.

## Product Purpose

LIZAR SRL · Fleet Manager Pro es el sistema interno de LIZAR SRL, empresa de transporte de carga de Trujillo, Perú. La iteración actual cubre el **Módulo de Registro de Gastos y Comprobantes**: registrar un gasto adicional asociado a un viaje y a un cliente, adjuntar su comprobante de pago y aprobarlo o rechazarlo.

Éxito: cada gasto llega a "Sustentado" con su comprobante aprobado, sin pasos perdidos entre módulos, tanto desde la oficina como desde la ruta.

## Operating Context

- Ciclo de un gasto: **Registrado** → (se adjunta un comprobante) **Pagado** → (se aprueba) **Sustentado**. Un rechazo exige un motivo y devuelve el gasto a **Registrado**.
- Estados de un comprobante: **Pendiente de aprobación**, **Aprobado**, **Rechazado**.
- Para registrar un gasto deben existir un viaje y un cliente. Cadena de datos de apoyo: Rubro → Cliente; Ubigeo → Dirección; Marca + Tipo de unidad + Subtipo → Unidad; Conductor + Unidad → Asignación C/U; Asignación + Cliente + Direcciones de origen y destino → Viaje.
- Comprobantes aceptados: JPG, PNG o PDF de hasta 5 MB. Un comprobante adjunto no se edita; si es incorrecto se adjunta uno nuevo.
- Moneda: soles (`S/`). Idioma: español de Perú.

## Capabilities and Constraints

- Frontend Angular 21 (componentes standalone y signals) con CSS propio, sin librería de componentes.
- Dos servicios: la API principal (FastAPI, `/api/v1`, autenticación JWT) para usuarios y datos maestros, y el servicio de gastos (Go) para gastos y comprobantes.
- **Copy exacto de UX writing (RNF-1):** las etiquetas y colores de estado (`gasto-estados.ts`) y los mensajes de error (`error-messages.ts`) no se reformulan. Estados de gasto: Registrado = gris, Pagado = azul, Sustentado = verde. Estados de comprobante: Pendiente = amarillo, Aprobado = verde, Rechazado = rojo.
- Fuera del alcance de esta iteración, y ocultos: tipos de gasto, categorías de producto, guías de transportista, viajes realizados, productos transportados, facturas y presupuestos.
- Todavía no existe gestión de usuarios (RF-19 a RF-21): la sesión actual actúa como Encargado de Logística para aprobar o rechazar.

## Brand Commitments

- Nombre: **LIZAR SRL · Fleet Manager Pro**.
- Logo: el pictograma del camión de carga.
- Color principal: **rojo LIZAR `#E11D48`**.

## Evidence on Hand

- No hay fotos propias de la flota ni testimonios; la landing actual usa fotos de Unsplash.
- Los únicos indicadores reales son los conteos del servicio de gastos (gastos por estado y comprobantes pendientes). No se inventan clientes, cifras ni testimonios.

## Product Principles

1. **El flujo del gasto manda.** Cada pantalla deja a un clic el siguiente paso de ese gasto.
2. **Un dato relacionado se elige, no se teclea.** Los registros de otros módulos se seleccionan por su nombre, no por su ID.
3. **La oficina revisa, la ruta captura.** El escritorio está optimizado para revisar y aprobar; el celular, para registrar y adjuntar.
4. **El estado siempre a la vista.** Gastos y comprobantes muestran su estado con el mismo vocabulario y color en toda la aplicación.
