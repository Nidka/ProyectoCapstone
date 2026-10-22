---
version: 1
slug: "frontend-src-app"
primary_target: "frontend/src/app"
related_targets: ["frontend/src/app/features/landing","frontend/src/app/features/login"]
---

## Scope

Aplicación autenticada de Fleet Manager Pro (`/app/*`) más landing y acceso, en un mismo mundo. Modo Operate para `/app`; Persuade para la landing. Spec: `docs/superpowers/specs/2026-10-02-rediseno-hitos-design.md`.

## Direction contract

THESIS: La pantalla es un tramo de carretera: cada gasto avanza por tres hitos (Registrado, Pagado, Sustentado) y la interfaz siempre señala el siguiente paso a un clic. Rechaza el panel oscuro con barra lateral de 15 enlaces, tarjetas KPI con brillo rojo y texto en degradado.

OWN-WORLD: Barra asfalto #1E2124 con cinta reflectiva de 4 px en segmentos rojo #E11D48 y blanco; fondo blanco hito #F7F7F5; tinta #1E2124. Estados como la señalización vial: amarillo preventivo = pendiente, azul informativo = pagado, verde guía = sustentado/aprobado, rojo = rechazado, gris = registrado. Overpass en todo con cifras tabulares; Overpass Mono para N° y placas. Hitos: postes cuadrados con borde de 2 px, rellenos al alcanzarse; el camión del logo marca el actual. Donde una cifra pertenece a un estado va pintada en un hito kilométrico: poste claro con borde de 2 px, tope redondeado y cabezal del color del estado. Las señales se construyen completas: rombo preventivo con su placa complementaria para lo que espera aprobación, banda verde de guía y banda azul informativa con orla blanca. La foto de la carretera abre la landing a sangre y encabeza el panel de acceso; sobre ella el texto va en placas sólidas de asfalto. Radios 6/10 px, filetes de 1 px, sombras solo en lo flotante.

STORY: Logística ve en Inicio cuántos gastos esperan comprobante o aprobación, entra con un clic al gasto, revisa el comprobante y lo aprueba o rechaza en línea. El conductor, desde el celular, abre Gastos > Sin comprobante, toca su gasto y adjunta la foto.

FIRST VIEWPORT: Gastos en escritorio 1440. Barra asfalto de 56 px: marca roja y Inicio, Gastos (contador amarillo), Operación, Catálogos a la izquierda; "Buscar o ir a… ⌘K" y usuario a la derecha; cinta reflectiva debajo. Título "Gastos" 32 px con botón rojo "Registrar gasto" a la derecha; pestañas Todos / Sin comprobante / Por aprobar / Sustentados con conteos; tabla N° (mono), cliente, viaje, fecha, monto tabular a la derecha con total al pie, ruta compacta de 3 hitos y la acción del siguiente paso. Interacción firma: al avanzar un gasto, el camión se desliza al hito siguiente (420 ms).

FORM: Hitos kilométricos y señalización de la Panamericana Norte; candidata 1 de 7 (la elección de impeccable, presentada junto a la asignada); seed 1fb4b9c1; build code-led.

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

## Open decisions

- Resuelto: la foto de Unsplash `photo-1519003722824-194d4455a60c` responde 200 y su procedencia está en DESIGN.md.
