---
name: "LIZAR SRL · Fleet Manager Pro"
description: "Gastos de viaje con su comprobante, de la ruta a la oficina, en el sistema visual «Hitos de carretera»."
colors:
  rojo: "#e11d48"
  rojo-hover: "#c8113d"
  rojo-presionado: "#ad0e35"
  rojo-tinta: "#b4123a"
  rojo-suave: "#fdecef"
  rojo-borde: "#f6c7d1"
  via: "#f2c200"
  via-tinta: "#6b5300"
  via-suave: "#fff5cc"
  registrado: "#8a9096"
  registrado-tinta: "#4a5056"
  registrado-suave: "#eeefec"
  pagado: "#2563eb"
  pagado-tinta: "#1e4fc2"
  pagado-suave: "#e8effe"
  sustentado: "#15803d"
  sustentado-tinta: "#166534"
  sustentado-presionado: "#14532d"
  sustentado-suave: "#e6f4ea"
  asfalto: "#1e2124"
  asfalto-2: "#2a2e33"
  asfalto-3: "#3a3f45"
  asfalto-tinta: "#c9cdd1"
  suelo: "#f7f7f5"
  panel: "#fdfdfc"
  panel-2: "#f0f1ee"
  linea: "#e2e3e0"
  linea-2: "#c9ccc8"
  borde-control: "#858a85"
  tinta: "#1e2124"
  tinta-2: "#4a5056"
  tinta-3: "#656b71"
typography:
  display:
    fontFamily: "'Overpass Variable', 'Overpass', system-ui, -apple-system, 'Segoe UI', sans-serif"
    fontSize: "clamp(2.5rem, 3.9vw, 3.5rem)"
    fontWeight: 800
    lineHeight: 1.04
    letterSpacing: "-0.03em"
  headline:
    fontFamily: "'Overpass Variable', 'Overpass', system-ui, -apple-system, 'Segoe UI', sans-serif"
    fontSize: "2rem"
    fontWeight: 700
    lineHeight: 1.1
    letterSpacing: "-0.02em"
  title:
    fontFamily: "'Overpass Variable', 'Overpass', system-ui, -apple-system, 'Segoe UI', sans-serif"
    fontSize: "1.125rem"
    fontWeight: 600
    lineHeight: 1.2
  body:
    fontFamily: "'Overpass Variable', 'Overpass', system-ui, -apple-system, 'Segoe UI', sans-serif"
    fontSize: "0.9375rem"
    fontWeight: 400
    lineHeight: 1.45
  body-sm:
    fontFamily: "'Overpass Variable', 'Overpass', system-ui, -apple-system, 'Segoe UI', sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.45
  label:
    fontFamily: "'Overpass Variable', 'Overpass', system-ui, -apple-system, 'Segoe UI', sans-serif"
    fontSize: "0.8125rem"
    fontWeight: 500
    lineHeight: 1.45
  caption:
    fontFamily: "'Overpass Variable', 'Overpass', system-ui, -apple-system, 'Segoe UI', sans-serif"
    fontSize: "0.75rem"
    fontWeight: 600
    lineHeight: 1.45
  button:
    fontFamily: "'Overpass Variable', 'Overpass', system-ui, -apple-system, 'Segoe UI', sans-serif"
    fontSize: "0.875rem"
    fontWeight: 600
    lineHeight: 1
  cifra:
    fontFamily: "'Overpass Variable', 'Overpass', system-ui, -apple-system, 'Segoe UI', sans-serif"
    fontSize: "2.5rem"
    fontWeight: 800
    lineHeight: 1
    letterSpacing: "-0.02em"
    fontFeature: "'tnum' 1"
  mono:
    fontFamily: "'Overpass Mono', ui-monospace, 'SFMono-Regular', Menlo, monospace"
    fontSize: "0.92em"
    fontWeight: 600
    fontFeature: "'tnum' 1"
rounded:
  control: "6px"
  panel: "10px"
  pastilla: "999px"
spacing:
  e-1: "4px"
  e-2: "8px"
  e-3: "12px"
  e-4: "16px"
  e-5: "20px"
  e-6: "24px"
  e-8: "32px"
  e-10: "40px"
  e-12: "48px"
  e-16: "64px"
  margen: "32px"
  margen-celular: "16px"
  contenido-max: "1240px"
components:
  button-primario:
    backgroundColor: "{colors.rojo}"
    textColor: "#ffffff"
    typography: "{typography.button}"
    rounded: "{rounded.control}"
    padding: "0px 16px"
    height: "40px"
  button-primario-hover:
    backgroundColor: "{colors.rojo-hover}"
  button-primario-active:
    backgroundColor: "{colors.rojo-presionado}"
  button-guia:
    backgroundColor: "{colors.sustentado}"
    textColor: "#ffffff"
    typography: "{typography.button}"
    rounded: "{rounded.control}"
    padding: "0px 16px"
    height: "40px"
  button-guia-hover:
    backgroundColor: "{colors.sustentado-tinta}"
  button-guia-active:
    backgroundColor: "{colors.sustentado-presionado}"
  button-secundario:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.tinta}"
    typography: "{typography.button}"
    rounded: "{rounded.control}"
    padding: "0px 16px"
    height: "40px"
  button-secundario-hover:
    backgroundColor: "{colors.panel-2}"
  button-fantasma:
    backgroundColor: "transparent"
    textColor: "{colors.tinta-2}"
    typography: "{typography.button}"
    rounded: "{rounded.control}"
    padding: "0px 16px"
    height: "40px"
  button-fantasma-hover:
    backgroundColor: "{colors.panel-2}"
    textColor: "{colors.tinta}"
  button-peligro:
    backgroundColor: "{colors.rojo-suave}"
    textColor: "{colors.rojo-tinta}"
    typography: "{typography.button}"
    rounded: "{rounded.control}"
    padding: "0px 16px"
    height: "40px"
  button-peligro-hover:
    backgroundColor: "#fbdbe2"
  button-sobre-asfalto:
    backgroundColor: "{colors.asfalto-2}"
    textColor: "#ffffff"
    typography: "{typography.button}"
    rounded: "{rounded.control}"
    padding: "0px 16px"
    height: "40px"
  button-sobre-asfalto-hover:
    backgroundColor: "{colors.asfalto-3}"
  button-sm:
    padding: "0px 12px"
    height: "32px"
  button-lg:
    padding: "0px 24px"
    height: "48px"
  button-celular:
    height: "44px"
  control:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.tinta}"
    typography: "{typography.body}"
    rounded: "{rounded.control}"
    padding: "0px 12px"
    height: "40px"
  control-celular:
    height: "44px"
  insignia-registrado:
    backgroundColor: "{colors.registrado-suave}"
    textColor: "{colors.registrado-tinta}"
    typography: "{typography.caption}"
    rounded: "{rounded.control}"
    padding: "0px 8px"
    height: "24px"
  insignia-pagado:
    backgroundColor: "{colors.pagado-suave}"
    textColor: "{colors.pagado-tinta}"
    typography: "{typography.caption}"
    rounded: "{rounded.control}"
    padding: "0px 8px"
    height: "24px"
  insignia-sustentado:
    backgroundColor: "{colors.sustentado-suave}"
    textColor: "{colors.sustentado-tinta}"
    typography: "{typography.caption}"
    rounded: "{rounded.control}"
    padding: "0px 8px"
    height: "24px"
  insignia-pendiente:
    backgroundColor: "{colors.via-suave}"
    textColor: "{colors.via-tinta}"
    typography: "{typography.caption}"
    rounded: "{rounded.control}"
    padding: "0px 8px"
    height: "24px"
  insignia-rechazado:
    backgroundColor: "{colors.rojo-suave}"
    textColor: "{colors.rojo-tinta}"
    typography: "{typography.caption}"
    rounded: "{rounded.control}"
    padding: "0px 8px"
    height: "24px"
  contador:
    backgroundColor: "{colors.panel-2}"
    textColor: "{colors.tinta-2}"
    typography: "{typography.caption}"
    rounded: "{rounded.pastilla}"
    height: "20px"
  contador-via:
    backgroundColor: "{colors.via}"
    textColor: "{colors.asfalto}"
  pestana:
    textColor: "{colors.tinta-2}"
    typography: "{typography.button}"
    padding: "0px 12px"
    height: "44px"
  pestana-activa:
    textColor: "{colors.tinta}"
  panel:
    backgroundColor: "{colors.panel}"
    rounded: "{rounded.panel}"
    padding: "24px"
  tabla-cabecera:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.tinta-2}"
    typography: "{typography.caption}"
    padding: "12px 16px"
  tabla-celda:
    typography: "{typography.body-sm}"
    padding: "12px 16px"
    height: "48px"
  atender-placa:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.tinta}"
    rounded: "{rounded.control}"
    padding: "16px 20px"
    height: "84px"
  placa-complementaria:
    backgroundColor: "{colors.via}"
    textColor: "{colors.asfalto}"
    rounded: "{rounded.control}"
    padding: "8px 16px 7px"
  hito:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.tinta}"
    rounded: "{rounded.panel}"
    padding: "8px 12px 12px"
    width: "88px"
  hito-grande:
    typography: "{typography.cifra}"
    padding: "12px 20px 16px"
    width: "168px"
  hito-cabezal-registrado:
    backgroundColor: "{colors.registrado-tinta}"
    textColor: "#ffffff"
    padding: "5px 12px 4px"
  hito-cabezal-pagado:
    backgroundColor: "{colors.pagado}"
    textColor: "#ffffff"
    padding: "5px 12px 4px"
  hito-cabezal-sustentado:
    backgroundColor: "{colors.sustentado}"
    textColor: "#ffffff"
    padding: "5px 12px 4px"
  senal-preventiva:
    backgroundColor: "{colors.via}"
    textColor: "{colors.asfalto}"
    rounded: "3px"
    size: "40px"
  senal-guia:
    backgroundColor: "{colors.sustentado}"
    textColor: "#ffffff"
    rounded: "{rounded.control}"
    size: "40px"
  senal-informativa:
    backgroundColor: "{colors.pagado}"
    textColor: "#ffffff"
    rounded: "{rounded.control}"
    size: "40px"
  cartel-guia:
    backgroundColor: "{colors.sustentado}"
    textColor: "#ffffff"
    padding: "12px 20px"
  cartel-informativa:
    backgroundColor: "{colors.pagado}"
    textColor: "#ffffff"
    padding: "12px 20px"
  barra:
    backgroundColor: "{colors.asfalto}"
    textColor: "{colors.asfalto-tinta}"
    height: "56px"
  barra-enlace-activo:
    textColor: "#ffffff"
  tabbar:
    backgroundColor: "{colors.asfalto}"
    textColor: "{colors.asfalto-tinta}"
    height: "64px"
  aviso:
    backgroundColor: "{colors.asfalto}"
    textColor: "#ffffff"
    rounded: "{rounded.panel}"
    padding: "12px 12px 12px 16px"
  marca-insignia:
    backgroundColor: "{colors.rojo}"
    textColor: "#ffffff"
    rounded: "8px"
    size: "32px"
---

# Design System: LIZAR SRL · Fleet Manager Pro

## Overview

**Creative North Star: "Hitos de carretera"**

La pantalla es un tramo de carretera. Cada gasto avanza por tres hitos kilométricos (Registrado, Pagado, Sustentado) y la interfaz siempre señala el siguiente paso, a un clic. El mundo sale de la Panamericana Norte y de los camiones de LIZAR: asfalto para las barras y los paneles de carretera, blanco hito para el suelo, la cinta reflectiva roja y blanca bajo cada barra de asfalto, postes cuadrados como hitos y la señalización vial como único código de estado. Los objetos del camino se construyen completos: el hito kilométrico lleva pintada la cifra de un estado, el rombo preventivo anuncia con su placa complementaria lo que espera aprobación y una banda azul informativa o verde de guía, con orla blanca, rotula los pasos «Revisar comprobante» y «Gasto sustentado».

Es una herramienta de trabajo antes que una vitrina. La densidad es media alta y está medida con datos reales: filas de 48 a 60 px, controles de 40 px en escritorio y de 44 px en celular, cifras tabulares en todo monto. La oficina revisa en una tabla ancha; la ruta captura en tarjetas apiladas con la acción a todo el ancho. La landing (modo Persuadir) y el acceso viven en el mismo mundo: la misma carretera de asfalto, los mismos postes y la misma cinta. La única foto, un camión de carga en una carretera de montaña, abre la portada a sangre y encabeza el panel de acceso; sobre ella el texto va en placas sólidas de asfalto.

El sistema rechaza el panel oscuro con barra lateral de 15 enlaces, las tarjetas KPI con brillo rojo y el texto en degradado, y también los brillos, las partículas y los emojis como iconos. Hay un solo tema, claro (`color-scheme: light`); el oscuro queda fuera de esta iteración y los colores ya viven como tokens para sumarlo después.

**Key Characteristics:**
- Barra de asfalto de 56 px (52 px en celular) con la cinta reflectiva de 4 px debajo.
- Estados con el código de la señalización vial y su etiqueta exacta: junto a su poste cuadrado en la insignia y en el cabezal del hito kilométrico.
- La ruta del gasto como firma: tres postes sobre una línea de carretera y el camión del logo en el hito actual.
- Hitos kilométricos con la cifra de un estado y señales viales completas: el rombo preventivo con su placa, y placas y bandas de guía o informativas con orla blanca.
- Un rojo LIZAR para marca, acción principal, foco y lo activo; el amarillo vía solo para lo que espera.
- Overpass en toda la interfaz, cifras tabulares y «N°» en Overpass Mono.
- Una sola foto, a sangre en la portada y en lo alto del panel de acceso, con el texto en placas de asfalto.
- Plano en reposo: filetes de 1 px y sombras teñidas de asfalto solo en lo que flota.
- Movimiento corto y con propósito, que se apaga con `prefers-reduced-motion`.

## Colors

Neutros de asfalto y blanco hito, un rojo de marca que significa avanzar, un amarillo que significa esperar y la señalización vial reservada a los estados. La única fuente de verdad es `frontend/src/styles/tokens.css`.

### Primary
- **Rojo LIZAR** (#e11d48): color de marca y de la acción principal. Botón primario («Registrar gasto», «Revisar», «Ingresar»), insignia de la marca, anillo de foco de 2 px, subrayado de 3 px de la pestaña y del enlace activo, segmentos de la cinta, filete superior de 3 px de la tarjeta del paso mientras el gasto está Registrado, cuadro del camión en la ruta y en la vía de la portada, y asterisco de campo obligatorio. Texto blanco encima (4.7:1).
- **Rojo hover** (#c8113d) y **Rojo presionado** (#ad0e35): estados del botón primario; el hover existe solo con puntero fino. El presionado también oscurece los enlaces al pasar el cursor.
- **Rojo tinta** (#b4123a): el rojo para leer sobre claro. Enlaces, errores de campo, motivo de rechazo, texto de la alerta de error y de la insignia Rechazado (6.0:1 sobre rojo suave).
- **Rojo suave** (#fdecef) y **Rojo borde** (#f6c7d1): fondo y borde del rojo teñido. Alerta de error, desvío de la ruta, insignia Rechazado, botón de peligro y bloque «Motivo del rechazo».

### Secondary
- **Amarillo vía** (#f2c200): la señal preventiva. Marca solo lo que espera a alguien: contadores de comprobantes por aprobar (barra superior, pestaña «Por aprobar», punto de la barra inferior), el rombo preventivo y su placa complementaria en «Para atender» de Inicio y en la vía de la portada, y el poste de la insignia Pendiente. El texto encima siempre es asfalto (9.6:1).
- **Amarillo suave** (#fff5cc) y **Tinta amarilla** (#6b5300): insignia «Pendiente de aprobación», alerta de requisitos faltantes y selección de texto. Los tokens de pendiente son alias de los de vía.

### Tertiary
Señalización de estado, fijada por el producto (RNF-1). Cada estado tiene tres tonos: pleno para el poste, suave para el fondo de la insignia y tinta para su texto.
- **Gris registrado** (#8a9096 / #eeefec / #4a5056): Registrado. El cabezal de su hito kilométrico usa la tinta y no el pleno, para que el nombre en blanco se lea (8.2:1); el pleno rellena el poste gris de «Para atender».
- **Azul informativo** (#2563eb / #e8effe / #1e4fc2): Pagado, en el poste, la insignia, el cabezal del hito, la placa informativa y la banda «Revisar comprobante» (texto blanco encima, 5.2:1); también la zona para soltar el comprobante mientras se arrastra y la franja «PE» de la placa vehicular.
- **Verde guía** (#15803d / #e6f4ea / #166534; presionado #14532d): Sustentado y Aprobado, en el poste, la insignia, el cabezal del hito, la placa de guía y la banda «Gasto sustentado» (texto blanco encima, 5.0:1); botón «Aprobar», cierre «Aprobado por…» y alerta de éxito.
- Rechazado usa el rojo LIZAR y su rojo teñido: los tokens de rechazado son alias de los rojos.

### Neutral
- **Asfalto** (#1e2124): la carretera. Barras superior e inferior, panel y cabecera de acceso, «Recorrido de los gastos» de Inicio, la vía y las placas del titular en la portada de la landing, y los avisos; también el borde de 2 px de los hitos kilométricos y de las placas amarillas, el filete y el símbolo del rombo preventivo, y el `theme-color`. **Tinta** (#1e2124) es el mismo valor: el texto principal se escribe con asfalto.
- **Asfalto 2** (#2a2e33) y **Asfalto 3** (#3a3f45): hover y campo de búsqueda sobre asfalto; filetes, bordes e iniciales del usuario sobre asfalto.
- **Tinta de asfalto** (#c9cdd1): texto secundario sobre asfalto. Enlaces inactivos, rótulos de la barra inferior, datos del recorrido y mensajes de aviso (10.1:1).
- **Blanco hito** (#f7f7f5): fondo de toda página y segmentos claros de la cinta.
- **Panel** (#fdfdfc) y **Panel 2** (#f0f1ee): superficies (paneles, tablas, controles, hojas, diálogos) y su versión hundida (hover de filas, opción activa, esqueletos, control deshabilitado).
- **Línea** (#e2e3e0) y **Línea 2** (#c9ccc8): filetes de 1 px entre filas y alrededor de paneles; borde del botón secundario y de la placa blanca de «Para atender», y el tramo pendiente, discontinuo, de la carretera sobre fondo claro.
- **Borde de control** (#858a85): borde de los campos y de la lista del selector. Da 3:1 sobre suelo, panel y panel 2 (contraste no textual, WCAG 1.4.11): el borde es lo único que dibuja un campo vacío.
- **Tinta 2** (#4a5056) y **Tinta 3** (#656b71): texto secundario (bajadas, cabeceras de tabla, metadatos) y terciario (placeholders, etiquetas de datos, N° en tablas de catálogos, iconos de campo; 5.0:1 sobre blanco hito).
- El blanco puro (#fff) no tiene token: es el texto sobre rojo, azul, verde y asfalto, la orla de las señales y el borde de los postes del panel de acceso.

### Named Rules
**The Road Sign Rule.** El color de estado es señalización vial y solo dice estado: gris es Registrado, azul informativo es Pagado, verde guía es Sustentado o Aprobado, amarillo preventivo es Pendiente de aprobación y rojo es Rechazado. Etiqueta y color salen de `gasto-estados.ts` sin reformular (RNF-1) y nunca decoran.

**The Solid Red Rule.** El rojo LIZAR sólido significa marca y avance: acción principal, insignia de la marca, foco, lo activo y la cinta. Errores, rechazos y la única acción destructiva («Confirmar rechazo») usan el rojo teñido (fondo #fdecef, texto #b4123a), nunca el sólido.

**The Yellow Means Waiting Rule.** El amarillo vía aparece solo cuando algo espera a alguien, es decir, comprobantes por aprobar; fuera de eso solo lo lleva el rombo de un aviso de advertencia. Con el conteo en cero, los contadores se ocultan o vuelven a neutro y la placa de «Para atender» pasa a blanca con su poste gris.

## Typography

**Display Font:** Overpass Variable (con Overpass, system-ui, -apple-system, Segoe UI, sans-serif)
**Body Font:** Overpass Variable (la misma familia, pesos 400 a 800)
**Label/Mono Font:** Overpass Mono, 400 y 600 (con ui-monospace, SFMono-Regular, Menlo, monospace)

**Character:** Overpass se inspira en Highway Gothic, la letra de los carteles de carretera; aquí es la letra de la carretera puesta a trabajar en una tabla. Una sola familia en toda la interfaz y su gemela mono para lo que se lee carácter por carácter: números de registro, placas, códigos y atajos de teclado.

Las dos familias van autoalojadas con @fontsource (`@fontsource-variable/overpass` y `@fontsource/overpass-mono`, licencia OFL-1.1) desde `angular.json`, sin Google Fonts. Overpass trae el punto medio (U+00B7) con avance cero, así que `base.css` declara un tramo `@font-face` de la misma familia solo para U+00B7 que toma el signo de Helvetica Neue, Segoe UI, Roboto o Arial. Sin ese tramo, «Viaje N° 15 · 01/10» se lee «15 ·01».

### Hierarchy
- **Display** (800, de 2.5rem a 3.5rem según el ancho, `clamp(2.5rem, 3.9vw, 3.5rem)`; 2rem hasta 1023 px y 1.75rem en celular; 1.04; -0.03em): solo el titular de la portada de la landing, en placas de asfalto de una línea cada una.
- **Headline** (700, 2rem; 1.1; -0.02em): título de cada página (Inicio, Gastos, módulos, Iniciar sesión, página no encontrada) y títulos de sección de la landing (1.15, hasta 22ch); baja a 1.5rem en celular en la aplicación y en la landing. También el lema del panel de acceso (1.15, hasta 20ch).
- **Title** (600, 1.125rem, 1.2): títulos de sección («Para atender», «Recorrido», «Comprobantes») y de hojas; baja a 1rem en títulos de estado vacío, diálogos y en la barra del celular. El cartel de un paso usa el mismo tamaño en 700.
- **Body** (400, 0.9375rem, 1.45): texto base; bajadas de página hasta 62ch y textos de estado vacío hasta 46ch.
- **Body small** (400, 0.875rem, 1.45): celdas de tabla, alertas, menú, metadatos de fila y la nota de desvío.
- **Label** (500, 0.8125rem): etiqueta de campo, siempre encima del control; ayuda y error del campo en el mismo tamaño.
- **Caption** (600, 0.75rem): cabeceras de tabla, insignias de estado, contadores y cabezal del hito (en 700), rótulos de la barra inferior y etiquetas de datos.
- **Button** (600, 0.875rem, 1): botones, pestañas y enlaces de la barra; 0.8125rem en el botón chico y 1rem en el grande.
- **Cifra** (800, 2.5rem, 1, -0.02em, tabular): la cifra pintada en el hito grande (el monto del detalle) y los conteos de «Para atender»; 2rem en el hito normal (recorrido de Inicio y vía de la portada) y en «Para atender» en celular.
- **Mono** (Overpass Mono 600, 0.92em, tabular): «N° 14» y placas.

### Named Rules
**The Tabular Figures Rule.** Todo monto, conteo, fecha y número de registro usa cifras tabulares. Los montos se alinean a la derecha y el total se apoya en un filete de 2 px en tinta, como en un talonario.

**The N° Rule.** Un número de registro se escribe «N° 14» en Overpass Mono, en todas las tablas y bajo la cabecera «N°».

**The Sentence Case Rule.** Títulos, pestañas, cabeceras de tabla y etiquetas van en tipo oración, sin mayúsculas sostenidas ni tracking abierto. Solo la placa vehicular, las iniciales del usuario y el nombre de la marca abren el espaciado.

## Layout

Columna única centrada de 1240 px como máximo, con márgenes laterales de 32 px (16 px en celular); el detalle de un gasto se angosta a 880 px y el formulario a 640 px. El espaciado sigue una base de 4 px (4, 8, 12, 16, 20, 24, 32, 40, 48, 64): 32 px entre bloques de página, 24 px entre el encabezado y el contenido y 24 px dentro de los paneles (16 px en celular).

- **Encabezado de página:** título a la izquierda y acciones a la derecha, alineados por la base. En celular las acciones ocupan todo el ancho y se reparten.
- **Bandeja de gastos:** rejilla de siete columnas medida con datos reales (N° 72 px, cliente, viaje, fecha 88 px, monto 104 px, estado 196 px, acción 188 px). Por debajo de 1200 px se oculta Viaje. Por debajo de 960 px la tabla ya no cabe y cada gasto es una tarjeta: N° y monto, cliente y fecha, ruta e insignia, y la acción a todo el ancho.
- **Inicio:** «Para atender» en dos columnas (una en celular); debajo, el «Recorrido de los gastos», una banda de asfalto de tres tramos que en celular se apilan con la línea de la vía en vertical, y los últimos cinco gastos en la lista compacta.
- **Shell:** la barra superior queda pegada arriba. Por debajo de 1024 px el buscador se reduce a un icono y se oculta el nombre del usuario. En celular la navegación baja a una barra inferior fija de cuatro destinos (64 px más la zona segura) y el título de la pantalla sube a la barra superior.
- **Formularios:** los de las hojas de registro usan una rejilla de dos columnas que pasa a una en celular. En celular la acción de «Registrar gasto» queda pegada sobre la barra inferior.
- **Landing:** la portada ocupa la primera vista (alto mínimo de `100dvh` menos la barra de 64 px y su cinta): la foto la cubre a sangre, el titular y la placa de la bajada se alinean a la izquierda dentro de la columna de 1240 px y la vía de asfalto queda al pie. Debajo, «Cómo avanza un gasto» con tres pasos en fila (en columna en celular) y «Desde la oficina y desde la ruta», solo texto, en 5fr/7fr con dos bloques. Por debajo de 1024 px la foto queda arriba (3:2), el titular va debajo sobre asfalto y la sección de lugares pasa a una columna.
- **Acceso:** panel de asfalto (2fr, mínimo 360 px) junto al formulario (3fr) con tarjeta de 420 px (560 px al crear usuario). El panel apila la foto (42 % del alto), la cinta y, sobre asfalto, la marca grande, la ruta de tres postes y el lema. Por debajo de 900 px el panel no se muestra y queda una cabecera de asfalto de 56 px con la marca y la cinta.
- **Puntos de corte:** 767, 899, 959, 1023 y 1199 px. El hover solo existe con puntero fino; se usan `100dvh`, `env(safe-area-inset-*)` y `viewport-fit=cover`.

### Named Rules
**The 44 px Rule.** En celular todo objetivo táctil mide al menos 44 px, también el botón chico y las opciones del selector, y el texto de los controles sube a 16 px para que iOS no haga zoom.

## Elevation & Depth

Sistema plano con dos materiales: el suelo claro y el asfalto. Los paneles se separan del suelo por tono (panel sobre blanco hito) y por un filete de 1 px, nunca por sombra. Las sombras, teñidas de asfalto, aparecen solo en lo que flota sobre la página, y detrás de hojas, diálogos y paleta un velo de asfalto al 40 % (`rgb(30 33 36 / 0.4)`) apaga el resto.

### Shadow Vocabulary
- **Sombra flotante** (`box-shadow: 0 8px 24px rgb(30 33 36 / 0.12), 0 2px 6px rgb(30 33 36 / 0.08)`): hojas, diálogos, paleta ⌘K, menú de usuario y avisos.
- **Sombra mínima** (`box-shadow: 0 1px 2px rgb(30 33 36 / 0.06)`): la lista desplegada del selector de referencias.
- **Anillo de foco del control** (`box-shadow: 0 0 0 3px rgb(225 29 72 / 0.2)`): campo enfocado, junto con el borde en tinta.
- **Halo del camión** (`box-shadow: 0 0 0 2px` en color panel): separa el camión de la línea de la ruta; no es elevación.

### Named Rules
**The Float-Only Shadow Rule.** Solo lleva sombra lo que flota sobre la página: hojas, diálogos, paleta ⌘K, menú de usuario y avisos. Paneles, tablas y tarjetas son planos y se separan con filetes de 1 px.

**The Asphalt Layer Rule.** La capa oscura es asfalto, no sombra: barras, panel de acceso, recorrido de Inicio, vía de la landing, placas del titular sobre la foto y avisos usan asfalto plano, con texto blanco o tinta de asfalto.

## Shapes

Geometría de señalética: rectángulos de esquina apenas suavizada, cuadrados como postes, un rombo, líneas de carretera sólidas o discontinuas y una franja de cortes duros.

- **Radios:** 6 px en controles, botones, insignias, alertas, campo de búsqueda, placas y señales de guía e informativa; 10 px en paneles, tablas, listas, hojas, diálogos, avisos y tarjetas; 999 px en contadores y en la píldora «Servicio no disponible». La foto y el cartel no llevan radio: van de borde a borde.
- **Postes de hito:** cuadrados de esquina casi viva: 2 px en la ruta y en la insignia; 3 px en los postes grandes de la landing (24 px), de «Para atender» en Inicio (28 px) y del acceso (32 px). El borde es de 2 px: del color del estado en la ruta, en asfalto en Inicio y blanco sobre el asfalto del acceso; los de la landing llevan un aro de 2 px del color del estado, separado por 3 px de blanco hito, y el de la insignia (8 px) va sin borde.
- **Hito kilométrico:** silueta de poste de concreto: cuerpo claro con borde de 2 px en asfalto, tope redondeado de 10 px y pie casi vivo de 2 px, con el cabezal de color a todo el ancho.
- **Carretera:** tramo recorrido sólido de 2 px en tinta 2 y tramo pendiente discontinuo de 2 px en línea 2; sobre asfalto, la línea central de la vía es discontinua, de 3 px, en blanco translúcido (35 a 45 %).
- **Subrayado activo:** 3 px con las esquinas superiores redondeadas, en pestañas y en la barra.
- **Filetes:** 1 px en toda división; borde discontinuo para invitar a actuar (estado vacío de 1 px, zona para soltar de 2 px); filete de 2 px en tinta sobre el total.
- **Rombo preventivo:** un cuadrado girado 45° que llena su cuadro (lado del 70.72 %), con esquina de 3 px y un filete negro por dentro del borde, como en la señal real; así se lee también sobre asfalto. Su cuadro mide 56 px en Inicio, 48 px en la portada y 24 px en los avisos.
- **Placa y orla:** la placa complementaria es un rectángulo de 6 px de radio con borde de 2 px en asfalto. Las señales de guía e informativa y el cartel llevan una orla blanca de 2 px dibujada por dentro (`outline-offset: -5px`).

### Named Rules
**The Two Radii Rule.** 6 px para lo que se toca y 10 px para lo que contiene; 999 px solo en contadores y píldoras. Los postes de hito (2 o 3 px), el hito kilométrico (10 px arriba y 2 px abajo), el rombo preventivo (3 px) y la insignia de la marca (8 px) son las formas propias del mundo fuera de esa regla.

**The Hard-Stop Tape Rule.** La cinta reflectiva es una franja de 4 px con segmentos de 24 px en rojo LIZAR y blanco hito, de cortes duros. Va bajo toda barra o cabecera de asfalto (aplicación, landing y acceso); también corona la vía de la portada y cierra la foto del panel de acceso.

## Components

### Buttons
Rótulos firmes: una sola línea, peso 600, icono de 16 a 18 px a la izquierda y ninguna sombra.
- **Shape:** esquina suave de 6 px; 40 px de alto (32 px el chico, 48 px el grande, cuadrado en modo icono) y 44 px en celular para todos, también el chico.
- **Primary:** rojo LIZAR con texto blanco y 16 px a los lados; para la acción principal de la pantalla y para «Revisar» en la fila.
- **Hover / Focus:** color, fondo y borde en 150 ms, solo con puntero fino (el primario pasa a #c8113d); foco con anillo rojo de 2 px separado 2 px; al presionar, `scale(0.97)` en 140 ms con ease-out (solo sin reducción de movimiento) y el primario pasa a #ad0e35; deshabilitado al 45 % de opacidad y sin cursor de mano.
- **Guía:** verde guía con texto blanco, solo para «Aprobar»: el color del estado al que lleva.
- **Secondary:** panel con borde línea 2 y texto tinta; en hover, panel 2 con borde tinta 3. Para «Adjuntar comprobante», «Rechazar», «Reintentar» y «Ver comprobante».
- **Ghost:** sin fondo y con texto tinta 2; en hover, panel 2. Para «Ver», «Cancelar», «Actualizar» y cerrar.
- **Peligro:** rojo teñido (fondo rojo suave, borde rojo borde, texto rojo tinta); solo «Confirmar rechazo».
- **Sobre asfalto:** asfalto 2 con borde asfalto 3 y texto blanco, para una acción dentro de una superficie de asfalto. La variante está definida en `componentes.css`; hoy ninguna pantalla la usa.

**The Next Step Button Rule.** El botón de cada fila dice el siguiente paso del gasto: «Adjuntar comprobante» en secundario cuando está Registrado, «Revisar» en primario cuando está Pagado y «Ver» en fantasma cuando ya está Sustentado.

### Chips
- **Style:** insignia de estado de 24 px, radio de 6 px, 12 px en 600, fondo suave y texto tinta de su estado, con un poste cuadrado de 8 px en el color pleno. La de Pendiente suma un contorno interior al poste para que no se pierda sobre el amarillo suave. Etiqueta y clase salen de `gasto-estados.ts`.
- **State:** contadores en píldora de 20 px, 12 px en 700 con cifras tabulares, en panel 2 con tinta 2; pasan a amarillo vía con texto asfalto solo cuando cuentan comprobantes por aprobar.

### Cards / Containers
- **Corner Style:** 10 px.
- **Background:** panel sobre blanco hito; asfalto para las bandas de carretera (recorrido de Inicio, vía de la landing).
- **Shadow Strategy:** ninguna en reposo (ver Elevation & Depth).
- **Border:** filete de 1 px en línea; discontinuo en el estado vacío.
- **Internal Padding:** 24 px (16 px en celular), fijado en el panel por la variable `--panel-relleno` para que una banda de cabecera pueda llegar hasta el borde; las placas de «Para atender» usan 16 px por 20 px (12 px por 16 px en celular).
- **Tabla:** marco de panel con desplazamiento horizontal propio, cabeceras de 12 px en 600 con tinta 2 y tipo oración, celdas de 48 px y hover en panel 2. El N° va en Overpass Mono y tinta 3. La celda de ruta va en dos líneas, el origen arriba y el destino debajo tras una flecha dibujada, con 220 px de ancho mínimo: con menos ancho la tabla se desplaza en vez de partir la ruta palabra por palabra.
- **Fila de gasto:** toda la fila es un enlace al gasto (con foco en contorno rojo interior) y la acción queda por encima. En «Todos», un gasto sustentado retrocede a tinta 2 y peso 500 para que lo pendiente mande. El motivo de un rechazo se lee completo en rojo tinta, hasta dos líneas bajo el cliente.
- **Para atender** (Inicio): cada ítem es un enlace con su objeto del camino a la izquierda y una placa a la derecha (radio de 6 px, 84 px de alto mínimo; 72 px en celular) con la cifra de 40 px en 800 (32 px en celular), el texto de 16 px en 600 y una flecha que avanza 3 px en hover. Si hay comprobantes por aprobar, el ítem lleva el rombo preventivo de 56 px y la placa complementaria: amarillo vía, borde de 2 px y leyenda en asfalto. Sin pendientes, y siempre en «gastos sin comprobante», lleva el poste gris de 28 px y una placa blanca (panel con borde de 1 px en línea 2, que en hover pasa a tinta 3).
- **Estado vacío:** icono en un cuadro de 48 px, título de 16 px en 600, una frase y la acción que lo resuelve.
- **Esqueleto:** bloques en panel 2 con la forma final (fila de 56 px, bloque de 120 px, franja de 72 px) y un pulso de opacidad de 1.2 s solo sin reducción de movimiento.
- **Alertas:** radio de 6 px, 14 px, icono más texto; error en rojo teñido, éxito en verde suave y requisito en amarillo suave.

### Inputs / Fields
- **Style:** 40 px de alto (44 px y texto de 16 px en celular), fondo panel, borde de 1 px en borde de control (#858a85, 3:1), radio de 6 px y 12 px a los lados; placeholder en tinta 3. La etiqueta va arriba (13 px en 500) con asterisco rojo si el campo es obligatorio y la ayuda va debajo en 13 px; el placeholder nunca reemplaza a la etiqueta.
- **Focus:** borde en tinta y anillo rojo de 3 px al 20 %.
- **Error / Disabled:** borde rojo con `aria-invalid`, etiqueta y mensaje en rojo tinta (13 px en 500) enlazados con `aria-describedby`; deshabilitado en panel 2 con texto tinta 3.
- **Monto:** prefijo «S/» dentro del control, en tinta 2 y 600, con cifras tabulares.
- **Selector de referencia:** combobox que despliega su lista dentro del flujo, debajo del input (240 px como máximo, 200 px en rejillas), con opciones de 36 px (44 px en celular), la activa en panel 2 y la elegida en 600.
- **Zona para soltar el comprobante:** borde discontinuo de 2 px sobre blanco hito; al arrastrar o en hover, borde azul informativo sobre azul suave; con archivo, borde sólido y vista previa. En pantallas táctiles ofrece cámara o archivos.

### Navigation
- **Barra superior:** asfalto, pegada arriba, de 56 px (52 px en celular). La marca a la izquierda; Inicio, Gastos (con contador amarillo), Operación y Catálogos en 14 px y 600 con tinta de asfalto; el activo pasa a blanco con subrayado rojo de 3 px. A la derecha, «Buscar o ir a… ⌘K» (campo de 36 px en asfalto 2 con borde asfalto 3) y el usuario (iniciales en un cuadro de 28 px). Debajo, la cinta reflectiva. Si el servicio cae aparece la píldora roja «Servicio no disponible».
- **Barra de la landing:** asfalto, pegada arriba, de 64 px, con la marca a la izquierda y el botón primario «Ingresar» (o «Ir a Inicio» si hay sesión) a la derecha; debajo, la cinta.
- **Pestañas:** 44 px, 14 px en 600 con tinta 2; la activa en tinta con subrayado rojo de 3 px y su contador. Se desplazan en horizontal con ajuste. En Operación y Catálogos las subpestañas llevan icono (rojo en la activa) y van bajo el título, sin antetítulo.
- **Barra inferior** (celular): asfalto fija con cuatro destinos (Inicio, Gastos, Operación, Más), icono de 22 px y rótulo de 12 px en 600; el activo en blanco con icono rojo y un punto amarillo con el conteo por aprobar. «Más» abre una hoja inferior con Catálogos y la cuenta.
- **Paleta ⌘K:** diálogo de 640 px a 12vh del borde superior, campo de 56 px, grupos «Acciones» e «Ir a» y opción activa en panel 2 con icono rojo. No tiene animación, a propósito: se usa muchas veces al día.
- **Menú de usuario:** popover de panel anclado arriba a la derecha; entra con opacidad y `scale(0.97)` en 160 ms y sale en 120 ms.
- **Volver:** enlace de 14 px en 600 con tinta 2 y una flecha, sobre el título del detalle y del formulario.

### La ruta del gasto
La firma del sistema: el recorrido de un gasto dibujado como un tramo de carretera, con `<app-ruta-gasto>`.
- Tres hitos (Registrado, Pagado, Sustentado). El tramo recorrido va sólido (2 px, tinta 2) hasta el hito actual y el resto, discontinuo (2 px, línea 2).
- Cada hito es un poste cuadrado con borde de 2 px del color de su estado: 10 px en la versión compacta y 18 px en la completa; hueco mientras está pendiente y relleno al alcanzarse.
- El camión del logo, blanco sobre un cuadro rojo (18 px en la compacta y 32 px en la completa, radio de 4 px y halo de 2 px), se para sobre el hito actual.
- **Compacta** (88 px de ancho, en las filas): solo postes y camión, expuesta como imagen con una etiqueta del recorrido. **Completa** (detalle): bajo cada hito, el nombre del estado (14 px en 600, en tinta 3 si falta), la fecha y quién (13 px); en celular se vuelve vertical, con la línea a la izquierda y 72 px por hito. Un rechazo se anota como un desvío: nota en rojo teñido con una marca cuadrada roja.
- **Movimiento:** cuando el gasto avanza en la misma vista, el camión y el tramo recorrido se deslizan al hito nuevo (`transform`, 420 ms, ease-in-out). No se anima en el primer render y con reducción de movimiento el cambio es instantáneo.
- El mismo motivo de carretera reaparece fuera de la ruta: con postes en los pasos «Cómo avanza un gasto» de la landing, en «Para atender» de Inicio y en el panel de acceso, y con hitos kilométricos en el recorrido de Inicio y en la vía de la portada.
- La spec del rediseño planteaba postes de 12 y 20 px y un camión de 16 y 28 px; los valores de arriba son los del código.

**The Milestone Rule.** En la ruta del gasto, cada hito es un poste cuadrado con borde de 2 px del color de su estado: hueco mientras falta, relleno al alcanzarse, y el camión del logo marca el actual.

### Avisos, hojas y diálogos
- **Avisos:** asfalto con texto blanco, radio de 10 px y sombra flotante; cada tipo lleva a la izquierda la señal vial de su clase, de 24 px (guía verde para confirmación, rombo preventivo para advertencia, placa azul para información); el de error conserva su icono de alerta de 18 px, en rojo claro para leerse sobre asfalto. A la derecha, un botón cerrar de 28 px. Van abajo a la derecha (centrados sobre la barra inferior en celular) dentro de una región `aria-live`. Entran desde 16 px más abajo en 240 ms y salen en 150 ms.
- **Hoja:** `<dialog>` nativo, lateral derecha de 440 px en escritorio y desde abajo en celular (hasta 92dvh; la hoja «Más», hasta 85dvh), con cabecera, cuerpo con desplazamiento y pie con las acciones. Entra en 320 ms con la curva de cajón y sale en 200 ms.
- **Diálogo:** centrado, de 640 px, solo para «Ver registro»; entra con opacidad y `scale(0.97)` en 200 ms.

**The Reduced Motion Rule.** Todo `transform` o `clip-path` animado vive dentro de `@media (prefers-reduced-motion: no-preference)`. Con reducción de movimiento solo quedan los cambios de color y un fundido breve en la salida de los avisos (120 ms).

### Hitos kilométricos y señales viales
Los objetos del camino viven en `styles/senales.css` y solo usan tokens que ya existían; la placa complementaria se define junto a cada uso, en `inicio.css` y `landing.css`.
- **Hito kilométrico** (`<app-hito>`, `.hito`): un poste claro (panel) con borde de 2 px en asfalto, tope redondeado de 10 px y pie de esquina casi viva (2 px), de 88 px de ancho mínimo. El **cabezal** lleva el nombre exacto del estado en blanco (12 px en 700) sobre el color del estado (gris tinta para Registrado, azul para Pagado, verde para Sustentado) y la **cifra** va pintada en el cuerpo, en 800 y con cifras tabulares (32 px; 40 px en el hito grande, que mide 168 px de ancho mínimo y sube el cabezal a 13 px). No lleva sombra ni estados propios: es un objeto del camino, no una tarjeta de métrica.
- **Dónde va el hito:** en tres lugares donde una cifra pertenece a un estado. En el «Recorrido de los gastos» de Inicio, un hito por estado con su conteo, plantado sobre la línea discontinua de la vía y con la suma de montos debajo; cada tramo es un enlace a su pestaña y en hover pasa a asfalto 2. En la vía de la portada, con los conteos reales, plantado en el borde entre la foto y el asfalto. En la cabecera del detalle, el hito grande del estado actual lleva el monto a la derecha del título, alineado por la base: ahí no hay insignia ni monto suelto.
- **Señal preventiva** (`<app-senal tipo="preventiva">`): rombo amarillo vía con filete negro por dentro del borde y símbolo negro del set de iconos. Anuncia lo que espera aprobación: va con su **placa complementaria** (amarilla, borde de 2 px y leyenda en asfalto) en «Para atender» de Inicio y en la vía de la portada.
- **Señal de guía e informativa** (`tipo="guia"`, `tipo="informativa"`): placa verde o azul de 6 px de radio, con orla blanca de 2 px por dentro y símbolo blanco. Hoy van en los avisos.
- **Tamaño de la señal:** ocupa un cuadro de 40 px por defecto (56 px en Inicio, 48 px en la portada y 24 px en los avisos). El símbolo mide la mitad del cuadro en el rombo y el 55 % en las placas. La señal es decorativa (`aria-hidden`): el texto vecino dice lo que anuncia.
- **Cartel** (`.cartel--guia`, `.cartel--informativa`): el título de un paso pintado como banda de señal con orla blanca (18 px en 700, con texto e icono de 20 px en blanco), de borde a borde en la cabecera de su tarjeta: azul informativo para «Revisar comprobante» (gasto Pagado) y verde de guía para «Gasto sustentado». En esos dos estados la banda reemplaza al filete rojo de la tarjeta del paso, que queda para «Adjuntar comprobante de pago» (gasto Registrado). Va a todo el ancho y sin esquinas propias para que nunca se confunda con un botón.

**The Sign Means It Rule.** Una señal dice su clase con la forma y el color, como en la carretera: el rombo amarillo previene (lo que espera aprobación, una advertencia), la placa o la banda azul informa (un gasto Pagado que toca revisar, un aviso informativo) y la placa o la banda verde guía (un gasto Sustentado, una confirmación). Nunca como decoración.

### Portada de la landing
- **Titular:** en placas de asfalto, una por línea, con el texto en blanco (Display). Cada placa mide lo que su línea; la variable `--titulo-tam` fija el tamaño del titular y `--sangria` (0.24 de ese tamaño) el relleno lateral, de modo que el titular y la bajada arrancan en la misma vertical.
- **Bajada y acciones:** juntas en una placa de asfalto de 30rem como máximo: la bajada de 18 px en tinta de asfalto, el botón primario grande («Ingresar», o «Ir a Inicio» si hay sesión) y el enlace «Crear usuario» en blanco.
- **Vía:** banda de asfalto al pie de la portada con la cinta arriba, la línea central discontinua, el camión del logo en un cuadro rojo de 36 px (radio de 8 px), los tres hitos kilométricos con los conteos reales, que suben 44 px sobre la foto, y a la derecha la señal preventiva de 48 px con su placa, que dice cuántos comprobantes hay por aprobar. Mientras cargan los conteos se lee «…»; si el servicio no responde, los hitos muestran «—» y una nota reemplaza a la señal.
- **Tableta y celular:** por debajo de 1024 px la foto queda arriba (3:2); el titular (32 px; 28 px en celular) y la bajada van sin placa, sobre el asfalto de la portada, y los hitos bajan a la vía. En celular el camión se oculta y los hitos se reparten el ancho.
- **Secciones:** «Cómo avanza un gasto» son tres pasos sobre una línea discontinua, cada uno con su poste de 24 px, un verbo (24 px en 700), una frase y la insignia del estado al que lleva. «Desde la oficina y desde la ruta» es solo texto: dos bloques, cada uno con su icono en un cuadro de asfalto de 40 px.
- **Movimiento:** una sola vez al cargar: la línea de la vía se traza en 700 ms (`clip-path`, ease-in-out, 120 ms de retraso), los hitos aparecen desde 8 px más abajo en 400 ms, escalonados cada 60 ms, y la señal entra al final. Con reducción de movimiento todo queda estático.

### Panel de acceso
- `<app-panel-acceso>`, compartido por «Iniciar sesión» y «Crear usuario»: una columna de asfalto con la foto arriba (42 % del alto, con carga diferida), la cinta reflectiva y, debajo, la marca grande, la ruta y el lema.
- **Ruta:** tres postes de 32 px (borde blanco de 2 px, relleno del color del estado, esquina de 3 px) sobre una línea discontinua blanca al 40 %, con el nombre del estado debajo en 16 px y 600, en tinta de asfalto. Es decorativa (`aria-hidden`).
- **Lema:** «Gastos de viaje con su comprobante, de la ruta a la oficina.» en 32 px y 700, hasta 20ch.
- Por debajo de 900 px el panel no se muestra: queda la cabecera de asfalto de 56 px con la marca y la cinta.

### Marca
- Insignia roja plana de 32 px con radio de 8 px y el camión del logo en blanco (geometría original, trazo de 2.2); al lado, «LIZAR» en 16 px y 800 con 0.02em de espaciado y «Fleet Manager Pro» en 12 px y 500 al 72 %. Sin degradado ni brillo. Sobre claro, el texto pasa a tinta.
- En el panel de acceso la marca va a escala de cartel: insignia de 56 px (radio de 10 px, camión de 30 px), «LIZAR» en 32 px y «Fleet Manager Pro» en 14 px.

### Iconos
- Tabler Icons v3.48.0, variante outline (MIT), copiados como trazos en `iconos.ts` y dibujados por `<app-icono>` como SVG en línea: 18 px por defecto, trazo de 1.75, `currentColor`, decorativos (`aria-hidden`) y siempre junto a un texto. Dentro de una señal vial el símbolo va con trazo de 2.5: `exclamation-mark` en el rombo, `check` en la placa de guía e `info-small` en la informativa.

### Fotografía
- Una sola foto en todo el producto: un camión de carga en una carretera de montaña. Abre la **portada de la landing** a sangre (primera vista completa, `object-fit: cover`, `fetchpriority="high"` y `srcset` de 640 a 2000 px); encima va el titular en placas de asfalto, una por línea, y al pie la vía con los hitos. En tableta y celular la foto queda arriba (3:2) y el titular debajo, sobre asfalto. La misma foto ocupa la parte alta del **panel de acceso** (login y registro), con carga diferida y `srcset` de 640 a 1400 px; por debajo de 900 px ese panel no se muestra.
- Sobre la foto nunca va texto suelto ni un degradado: el texto va en placas sólidas de asfalto.
- **Procedencia:** Unsplash, foto `https://images.unsplash.com/photo-1519003722824-194d4455a60c`, bajo la licencia de Unsplash. Está enlazada desde images.unsplash.com y no está copiada al repositorio. Aparte del favicon, es el único mapa de bits que trae el producto.

## Do's and Don'ts

### Do:
- **Do** tomar todo color, radio, espacio, sombra y curva de `tokens.css` con `var(--…)`; un valor nuevo entra primero como token.
- **Do** mostrar el estado con `<app-insignia-estado>` y el recorrido con `<app-ruta-gasto>`: etiqueta y color salen de `gasto-estados.ts`.
- **Do** construir las señales completas con `<app-senal>` y `.cartel`: el rombo preventivo con su placa complementaria para lo que espera aprobación, y las placas y bandas de guía o informativas con su orla blanca.
- **Do** dejar el siguiente paso de cada gasto a un clic: «Revisar» en primario, «Adjuntar comprobante» en secundario y «Ver» en fantasma.
- **Do** escribir «N° 14» en Overpass Mono y los montos como «S/ 1,240.50» con cifras tabulares, alineados a la derecha.
- **Do** separar superficies con filetes de 1 px (#e2e3e0) y reservar la sombra flotante para hojas, diálogos, paleta, menú y avisos.
- **Do** poner la cinta reflectiva (4 px, segmentos de 24 px) bajo toda barra o cabecera de asfalto.
- **Do** dar 44 px a todo objetivo táctil en celular y 16 px al texto de los controles.
- **Do** envolver todo `transform` o `clip-path` animado en `@media (prefers-reduced-motion: no-preference)` y hacer las salidas más rápidas que las entradas.
- **Do** dibujar los iconos con `<app-icono>` (Tabler outline, trazo de 1.75) y acompañarlos de texto.
- **Do** registrar la procedencia y la licencia de toda imagen nueva, como la de la foto de Unsplash de la portada y del panel de acceso.

### Don't:
- **Don't** volver al panel oscuro con barra lateral de 15 enlaces, a las tarjetas KPI con brillo rojo ni al texto en degradado.
- **Don't** usar brillos, partículas, emojis ni degradados decorativos; el único degradado es la franja de cortes duros de la cinta reflectiva.
- **Don't** usar un color de estado como decoración ni reformular los textos de `gasto-estados.ts` y `error-messages.ts` (RNF-1).
- **Don't** pintar de rojo sólido una acción destructiva: «Confirmar rechazo» va en rojo teñido.
- **Don't** poner antetítulos sobre los títulos ni escribir títulos en mayúsculas.
- **Don't** dar sombra a paneles, tablas o tarjetas en reposo.
- **Don't** poner texto suelto ni un degradado sobre la foto: el texto va en placas sólidas de asfalto.
- **Don't** usar una señal vial como adorno ni dar al cartel forma de botón: va de borde a borde de su tarjeta y sin esquinas propias.
- **Don't** usar caracteres tecleados ni emojis como iconos.
- **Don't** animar la paleta ⌘K ni el camión de la ruta en su primer render.
- **Don't** quitar el `@font-face` de U+00B7 de `base.css` ni cargar las fuentes desde Google Fonts.
- **Don't** usar el placeholder como etiqueta de un campo.
