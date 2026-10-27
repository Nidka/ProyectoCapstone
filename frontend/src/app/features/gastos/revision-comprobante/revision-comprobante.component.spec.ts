import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { HttpErrorResponse } from '@angular/common/http';
import { Observable, Subject, TimeoutError, of, throwError } from 'rxjs';
import { RevisionComprobanteComponent } from './revision-comprobante.component';
import { GastosStore } from '../../../core/state/gastos.store';
import { GastosApiService } from '../../../core/services/gastos-api.service';
import { ChecklistService } from '../../../core/services/checklist.service';
import { LecturaService } from '../../../core/services/lectura.service';
import { ToastService } from '../../../core/services/toast.service';
import { ChecklistCliente } from '../../../core/models/checklist';
import { Comprobante, Gasto } from '../../../core/models/gasto.model';
import { LecturaComprobante } from '../../../core/models/lectura';
import { MSG_ERROR_RED, MSG_TIMEOUT, mensajeError } from '../../../core/utils/error-messages';

const comprobante: Comprobante = {
  id: 9, gasto_id: 7, archivo_url: '/uploads/9.jpg', tipo_archivo: 'imagen', estado: 'Pendiente', creado_en: '2026-10-01T17:00:00Z',
};
const gasto: Gasto = {
  id: 7, viaje_id: 14, cliente_id: 1, monto: 85.5, fecha: '2026-09-30T00:00:00Z', estado: 'Pagado', creado_por: 'Ana', creado_en: '2026-10-01T13:00:00Z',
};
const ESTANDAR: ChecklistCliente = {
  cliente_id: 1,
  personalizado: false,
  criterios: [
    { clave: 'legible', texto: 'El comprobante es legible y está completo.' },
    { clave: 'tipo_valido', texto: 'Es una boleta o factura válida.' },
    { clave: 'a_nombre_empresa', texto: 'Está emitido a nombre de LIZAR SRL.' },
    { clave: 'monto_coincide', texto: 'El monto coincide con el gasto.' },
    { clave: 'fecha_coincide', texto: 'La fecha corresponde al gasto.' },
  ],
};
const TODO = { legible: true, tipo_valido: true, a_nombre_empresa: true, monto_coincide: true, fecha_coincide: true };

function lecturaCon(estado: LecturaComprobante['estado'], sugerencias: LecturaComprobante['sugerencias'] = {}): LecturaComprobante {
  return {
    estado,
    datos: {
      monto: { valor: 0, texto: '', confianza: 'ninguna' },
      fecha: { valor: '', texto: '', confianza: 'ninguna' },
      rucs: [],
      tipo: { valor: 'otro', texto: '', confianza: 'ninguna' },
    },
    sugerencias,
  };
}
/** Leída: sugiere monto y empresa; la fecha leída no coincide y el tipo no se encontró */
const LEIDA = lecturaCon('leido', {
  monto_coincide: { sugerido: true, detalle: 'Leído S/ 85.50' },
  fecha_coincide: { sugerido: false, detalle: 'Leída 29/09/2026' },
  a_nombre_empresa: { sugerido: true, detalle: 'RUC 20481608750' },
  tipo_valido: { sugerido: false, detalle: '' },
});

/** Leída con valores: el monto y la fecha leídos no coinciden con el gasto (S/ 85.50 del 30/09/2026) */
const DESCUADRADA: LecturaComprobante = {
  estado: 'leido',
  datos: {
    monto: { valor: 513, texto: 'TOTAL S/ 513.00', confianza: 'alta' },
    fecha: { valor: '2019-09-26', texto: 'FECHA 26/09/2019', confianza: 'alta' },
    rucs: ['20481608750'],
    tipo: { valor: 'factura', texto: 'FACTURA', confianza: 'alta' },
  },
  sugerencias: {
    monto_coincide: { sugerido: false, detalle: 'Leído S/ 513.00' },
    fecha_coincide: { sugerido: false, detalle: 'Leída 26/09/2019' },
    a_nombre_empresa: { sugerido: true, detalle: 'RUC 20481608750' },
    tipo_valido: { sugerido: true, detalle: 'Factura' },
  },
};

interface Opciones {
  respuesta?: () => Observable<unknown>;
  checklist?: () => Observable<ChecklistCliente>;
  /** Por defecto, una lectura sin sugerencias (la revisión se ve como antes del OCR) */
  lectura?: (id: number) => Observable<LecturaComprobante>;
  /** Por defecto, la corrección sale bien */
  correccion?: () => Observable<unknown>;
  /** false = no esperar a que la carga termine (para probar el estado «cargando») */
  esperar?: boolean;
}

async function montar(op: Opciones = {}) {
  const store = { aprobar: vi.fn(op.respuesta ?? (() => of({}))), rechazar: vi.fn(op.respuesta ?? (() => of({}))), corregir: vi.fn(op.correccion ?? (() => of({}))), cargar: vi.fn() };
  const checklists = { deCliente: vi.fn(op.checklist ?? (() => of(ESTANDAR))) };
  const lecturas = { lectura: vi.fn(op.lectura ?? (() => of(lecturaCon('leido')))) };
  TestBed.configureTestingModule({
    providers: [
      { provide: GastosStore, useValue: store },
      { provide: ChecklistService, useValue: checklists },
      { provide: LecturaService, useValue: lecturas },
      { provide: GastosApiService, useValue: { fileUrl: (u: string) => `http://localhost:8081${u}` } },
    ],
  });
  const f = TestBed.createComponent(RevisionComprobanteComponent);
  f.componentRef.setInput('comprobante', comprobante);
  f.componentRef.setInput('gasto', gasto);
  const refrescar = async () => {
    f.detectChanges();
    if (op.esperar !== false) await f.whenStable();
    f.detectChanges();
  };
  await refrescar();
  const el = f.nativeElement as HTMLElement;
  /** El botón con ese texto exacto; si no hay, el primero que lo contenga («Rechazar» no es «Rechazar con este motivo») */
  const boton = (texto: string) => {
    const botones = [...el.querySelectorAll('button')];
    return (botones.find(b => b.textContent?.trim() === texto) ?? botones.find(b => b.textContent?.includes(texto))) as HTMLButtonElement;
  };
  const casillas = () => [...el.querySelectorAll<HTMLInputElement>('.checklist input[type="checkbox"]')];
  const marcar = async (cuantas: number) => {
    casillas().slice(0, cuantas).forEach(c => c.click());
    await refrescar();
  };
  const sugerencias = () =>
    Object.fromEntries(
      [...el.querySelectorAll('.checklist__lista li')]
        .filter(li => li.querySelector('.checklist__sugerencia'))
        .map(li => [casillas().indexOf(li.querySelector('input')!), li.querySelector('.checklist__sugerencia')!.textContent?.trim()]),
    );
  const lecturaTexto = () => el.querySelector('.checklist__lectura')?.textContent?.trim() ?? '';
  return { f, el, store, checklists, lecturas, boton, casillas, marcar, refrescar, sugerencias, lecturaTexto };
}

describe('RevisionComprobanteComponent', () => {
  it('muestra el comprobante para revisarlo', async () => {
    const { el } = await montar();
    expect(el.querySelector('img')?.getAttribute('src')).toBe('http://localhost:8081/uploads/9.jpg');
  });

  it('pide los criterios del cliente del gasto y los muestra con el monto y la fecha', async () => {
    const { el, checklists, casillas } = await montar();
    expect(checklists.deCliente).toHaveBeenCalledWith(1);
    expect(el.querySelector('.checklist legend')?.textContent?.trim()).toBe('Checklist documental');
    expect(casillas()).toHaveLength(5);
    const textos = [...el.querySelectorAll('.checklist__criterio')].map(c => c.textContent?.trim());
    expect(textos).toContain('El monto coincide con el gasto (S/ 85.50).');
    expect(textos).toContain('La fecha corresponde al gasto (30/09/2026).');
  });

  it('muestra exactamente los criterios personalizados del cliente', async () => {
    const propio: ChecklistCliente = {
      cliente_id: 1, personalizado: true,
      criterios: [{ clave: 'legible', texto: 'Se lee completo' }, { clave: 'c-9f2a11', texto: 'Lleva sello de recepción' }],
    };
    const { el, store, boton, marcar, casillas } = await montar({ checklist: () => of(propio) });
    expect(casillas()).toHaveLength(2);
    expect(el.querySelector('.checklist__conteo')?.textContent?.trim()).toBe('0 de 2 criterios');
    await marcar(2);
    expect(boton('Aprobar').disabled).toBe(false);
    boton('Aprobar').click();
    expect(store.aprobar).toHaveBeenCalledWith(9, { legible: true, 'c-9f2a11': true });
  });

  it('Aprobar queda deshabilitado hasta marcar todos los criterios', async () => {
    const { el, boton, marcar } = await montar();
    expect(boton('Aprobar').disabled).toBe(true);
    expect(el.querySelector('#checklist-aviso')?.textContent?.trim()).toBe('Faltan 5 criterios para aprobar.');
    expect(boton('Aprobar').getAttribute('aria-describedby')).toBe('checklist-aviso');
    await marcar(4);
    expect(el.querySelector('.checklist__conteo')?.textContent?.trim()).toBe('4 de 5 criterios');
    expect(el.querySelector('#checklist-aviso')?.textContent?.trim()).toBe('Falta 1 criterio para aprobar.');
    expect(boton('Aprobar').disabled).toBe(true);
  });

  it('muestra el avance del checklist como una barra', async () => {
    const { el, marcar } = await montar();
    const barra = () => el.querySelector('.checklist__avance')!;
    expect(barra().getAttribute('role')).toBe('progressbar');
    expect(barra().getAttribute('aria-valuenow')).toBe('0');
    expect(barra().getAttribute('aria-valuemax')).toBe('5');
    await marcar(2);
    expect(barra().getAttribute('aria-valuenow')).toBe('2');
    expect(el.querySelector<HTMLElement>('.checklist__avance-relleno')!.style.width).toBe('40%');
  });

  it('con todos los criterios, aprobar envía el checklist y avisa con el copy exacto', async () => {
    const { el, store, boton, marcar } = await montar();
    await marcar(5);
    expect(boton('Aprobar').disabled).toBe(false);
    expect(el.querySelector('#checklist-aviso')).toBeNull();
    boton('Aprobar').click();
    expect(store.aprobar).toHaveBeenCalledWith(9, TODO);
    expect(TestBed.inject(ToastService).toasts()[0].title).toBe('Comprobante aprobado. El gasto quedó como "Sustentado".');
  });

  it('Aprobar va en verde de guía, Rechazar es neutro y el rojo queda para confirmar el rechazo', async () => {
    const { f, boton } = await montar();
    expect(boton('Aprobar').classList).toContain('btn--guia');
    expect(boton('Rechazar').classList).toContain('btn--secundario');
    boton('Rechazar').click();
    f.detectChanges();
    expect(boton('Confirmar rechazo').classList).toContain('btn--peligro');
  });

  it('rechazar propone el motivo con los criterios sin marcar y envía el checklist', async () => {
    const { el, store, boton, marcar, refrescar } = await montar();
    await marcar(3);
    boton('Rechazar').click();
    await refrescar();
    const area = el.querySelector<HTMLTextAreaElement>('#motivo-rechazo')!;
    expect(area.value).toBe('No cumple: el monto coincide con el gasto; la fecha corresponde al gasto.');
    boton('Confirmar rechazo').click();
    expect(store.rechazar).toHaveBeenCalledWith(9, 'No cumple: el monto coincide con el gasto; la fecha corresponde al gasto.', {
      ...TODO, monto_coincide: false, fecha_coincide: false,
    });
  });

  it('rechazar exige un motivo (con todo marcado no se propone ninguno) y lo envía sin espacios sobrantes', async () => {
    const { f, el, store, boton, marcar } = await montar();
    await marcar(5);
    boton('Rechazar').click();
    f.detectChanges();
    boton('Confirmar rechazo').click();
    f.detectChanges();
    expect(el.querySelector('#motivo-error')?.textContent?.trim()).toBe('Debes indicar un motivo para rechazar el comprobante.');
    expect(store.rechazar).not.toHaveBeenCalled();
    const area = el.querySelector<HTMLTextAreaElement>('#motivo-rechazo')!;
    area.value = '  Comprobante ilegible  ';
    area.dispatchEvent(new Event('input'));
    boton('Confirmar rechazo').click();
    expect(store.rechazar).toHaveBeenCalledWith(9, 'Comprobante ilegible', TODO);
    expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe('Comprobante rechazado. El gasto volvió a estado "Registrado".');
    expect(TestBed.inject(ToastService).toasts().at(-1)?.type).toBe('info');
  });

  it('con el rechazo abierto, el motivo propuesto sigue a las casillas mientras no se edite', async () => {
    const { el, boton, marcar, casillas, refrescar } = await montar();
    await marcar(3);
    boton('Rechazar').click();
    await refrescar();
    const area = el.querySelector<HTMLTextAreaElement>('#motivo-rechazo')!;
    expect(area.value).toBe('No cumple: el monto coincide con el gasto; la fecha corresponde al gasto.');
    casillas()[3].click();
    await refrescar();
    expect(area.value).toBe('No cumple: la fecha corresponde al gasto.');
    area.value = 'La boleta está a nombre de otra empresa';
    area.dispatchEvent(new Event('input'));
    casillas()[4].click();
    await refrescar();
    expect(area.value).toBe('La boleta está a nombre de otra empresa');
  });

  it('si el comprobante ya fue revisado (409), lo avisa y recarga', async () => {
    const { store, boton, marcar } = await montar({ respuesta: () => throwError(() => new HttpErrorResponse({ status: 409 })) });
    await marcar(5);
    boton('Aprobar').click();
    expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe('Este comprobante ya fue revisado. Actualizamos la vista.');
    expect(store.cargar).toHaveBeenCalled();
  });

  it('si el servidor rechaza el checklist al aprobar (422), lo explica y vuelve a pedir los criterios', async () => {
    const { checklists, boton, marcar, refrescar } = await montar({ respuesta: () => throwError(() => new HttpErrorResponse({ status: 422 })) });
    await marcar(5);
    boton('Aprobar').click();
    await refrescar();
    expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe('Para aprobar, el comprobante debe cumplir todos los criterios del checklist.');
    // El checklist del cliente pudo cambiar mientras se revisaba
    expect(checklists.deCliente).toHaveBeenCalledTimes(2);
  });

  it('si el servidor responde 422 al rechazar, usa el mensaje general de la app', async () => {
    const err = new HttpErrorResponse({ status: 422 });
    const { f, boton, marcar } = await montar({ respuesta: () => throwError(() => err) });
    await marcar(2);
    boton('Rechazar').click();
    f.detectChanges();
    boton('Confirmar rechazo').click();
    expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe(mensajeError(err));
  });

  it('lo marcado se conserva con una recarga del mismo comprobante y se reinicia con otro', async () => {
    const { f, el, marcar, refrescar } = await montar();
    await marcar(2);
    f.componentRef.setInput('comprobante', { ...comprobante });
    await refrescar();
    expect(el.querySelector('.checklist__conteo')?.textContent?.trim()).toBe('2 de 5 criterios');
    f.componentRef.setInput('comprobante', { ...comprobante, id: 10 });
    await refrescar();
    expect(el.querySelector('.checklist__conteo')?.textContent?.trim()).toBe('0 de 5 criterios');
  });

  it('mientras carga los criterios no se puede aprobar, pero sí rechazar (sin checklist)', async () => {
    const pendiente = new Subject<ChecklistCliente>();
    const { f, el, store, boton } = await montar({ checklist: () => pendiente, esperar: false });
    expect(el.textContent).toContain('Cargando checklist…');
    expect(boton('Aprobar').disabled).toBe(true);
    boton('Rechazar').click();
    f.detectChanges();
    const area = el.querySelector<HTMLTextAreaElement>('#motivo-rechazo')!;
    area.value = 'Ilegible';
    area.dispatchEvent(new Event('input'));
    boton('Confirmar rechazo').click();
    expect(store.rechazar).toHaveBeenCalledWith(9, 'Ilegible', undefined);
  });

  it('si no se pueden cargar los criterios lo dice, ofrece reintentar y no deja aprobar', async () => {
    let fallar = true;
    const { el, checklists, boton, refrescar } = await montar({
      checklist: () => (fallar ? throwError(() => new HttpErrorResponse({ status: 0 })) : of(ESTANDAR)),
    });
    expect(el.textContent).toContain('No se pudo cargar el checklist de este cliente.');
    expect(boton('Aprobar').disabled).toBe(true);
    fallar = false;
    boton('Reintentar').click();
    await refrescar();
    expect(checklists.deCliente).toHaveBeenCalledTimes(2);
    expect(el.querySelectorAll('.checklist input[type="checkbox"]')).toHaveLength(5);
  });

  describe('lectura automática del comprobante', () => {
    it('pide la lectura del comprobante y rotula cada casilla: coincide o revisar', async () => {
      const { lecturas, sugerencias, boton } = await montar({ lectura: () => of(LEIDA) });
      expect(lecturas.lectura).toHaveBeenCalledWith(9);
      // Orden estándar: legible, tipo_valido, a_nombre_empresa, monto_coincide, fecha_coincide
      expect(sugerencias()).toEqual({
        1: '⚠ Revisar',
        2: '✓ Coincide',
        3: '✓ Coincide',
        4: '⚠ Revisar',
      });
      expect(boton('Aplicar sugerencias')).toBeDefined();
    });

    it('el panel «Lectura automática» compara el gasto con lo leído y avisa que decide la persona', async () => {
      const { el } = await montar({ lectura: () => of(LEIDA) });
      expect(el.querySelector('.lectura__titulo')?.textContent?.trim()).toBe('Lectura automática');
      expect(el.querySelector('.lectura__nota')?.textContent?.trim()).toBe('Es una ayuda: tú decides si aprobar.');
      const filas = [...el.querySelectorAll('.lectura__fila')].map(tr => [...tr.children].map(c => c.textContent?.trim()));
      expect(filas).toEqual([
        ['Monto', 'S/ 85.50', 'S/ 85.50', 'Coincide'],
        ['Fecha', '30/09/2026', '29/09/2026', 'Revisar'],
        ['Empresa', 'LIZAR SRL', 'RUC 20481608750', 'Coincide'],
        ['Documento', 'Factura o boleta', 'No es factura ni boleta', 'Revisar'],
      ]);
      expect(el.querySelectorAll('.lectura__fila--revisar')).toHaveLength(2);
    });

    it('el veredicto resume la lectura: diferencias y qué hacer si el comprobante es incorrecto', async () => {
      const { el } = await montar({ lectura: () => of(LEIDA) });
      const v = el.querySelector('.lectura__veredicto')!;
      expect(v.classList.contains('lectura__veredicto--revisar')).toBe(true);
      expect(v.textContent).toContain('2 diferencias con el gasto: fecha, documento.');
      expect(v.textContent).toContain('Si el comprobante es incorrecto, recházalo: el gasto vuelve a «Registrado» y se adjunta uno nuevo.');
    });

    it('si todo coincide, el veredicto lo dice y no ofrece rechazar', async () => {
      const todo = lecturaCon('leido', {
        monto_coincide: { sugerido: true, detalle: 'Leído S/ 85.50' },
        fecha_coincide: { sugerido: true, detalle: 'Leída 30/09/2026' },
        a_nombre_empresa: { sugerido: true, detalle: 'RUC 20481608750' },
        tipo_valido: { sugerido: true, detalle: 'Factura' },
      });
      const { el, boton } = await montar({ lectura: () => of(todo) });
      const v = el.querySelector('.lectura__veredicto')!;
      expect(v.classList.contains('lectura__veredicto--si')).toBe(true);
      expect(v.textContent?.trim()).toContain('Todo coincide con el gasto. Puedes aprobar.');
      expect(boton('Rechazar con este motivo')).toBeUndefined();
    });

    it('una sola diferencia se dice en singular', async () => {
      const una = lecturaCon('leido', {
        monto_coincide: { sugerido: false, detalle: 'Leído S/ 513.00' },
        fecha_coincide: { sugerido: true, detalle: 'Leída 30/09/2026' },
      });
      const { el } = await montar({ lectura: () => of(una) });
      expect(el.querySelector('.lectura__veredicto')?.textContent).toContain('1 diferencia con el gasto: monto.');
    });

    it('«Rechazar con este motivo» abre el rechazo con el motivo redactado por la lectura', async () => {
      const { el, store, boton, refrescar } = await montar({ lectura: () => of(LEIDA) });
      boton('Rechazar con este motivo').click();
      await refrescar();
      const motivo = el.querySelector<HTMLTextAreaElement>('#motivo-rechazo')!;
      expect(motivo.value).toBe('No cumple: la fecha leída (29/09/2026) no coincide con la del gasto (30/09/2026); no es una factura ni una boleta.');
      // El motivo es editable y marcar casillas no lo pisa
      el.querySelectorAll<HTMLInputElement>('.checklist input[type="checkbox"]')[0].click();
      await refrescar();
      expect(el.querySelector<HTMLTextAreaElement>('#motivo-rechazo')!.value).toContain('la fecha leída');
      boton('Confirmar rechazo').click();
      expect(store.rechazar).toHaveBeenCalledWith(9, expect.stringContaining('la fecha leída (29/09/2026)'), expect.any(Object));
    });

    it('«Aplicar sugerencias» dice cuántas marca y, ya aplicadas, lo confirma', async () => {
      const { el, boton, refrescar } = await montar({ lectura: () => of(LEIDA) });
      expect(boton('Aplicar sugerencias').textContent).toContain('(2)');
      expect(el.querySelector('.lectura__aplicadas')).toBeNull();
      boton('Aplicar sugerencias').click();
      await refrescar();
      expect(boton('Aplicar sugerencias')).toBeUndefined();
      expect(el.querySelector('.lectura__aplicadas')?.textContent?.trim()).toBe('Sugerencias aplicadas');
    });

    it('«Aplicar sugerencias» marca solo las sugeridas y no toca las demás', async () => {
      const { el, store, boton, casillas, refrescar } = await montar({ lectura: () => of(LEIDA) });
      casillas()[0].click();
      casillas()[4].click();
      await refrescar();
      boton('Aplicar sugerencias').click();
      await refrescar();
      expect(casillas().map(c => c.checked)).toEqual([true, false, true, true, true]);
      expect(el.querySelector('.checklist__conteo')?.textContent?.trim()).toBe('4 de 5 criterios');
      expect(boton('Aprobar').disabled).toBe(true);
      expect(store.aprobar).not.toHaveBeenCalled();
      casillas()[1].click();
      await refrescar();
      boton('Aprobar').click();
      expect(store.aprobar).toHaveBeenCalledWith(9, TODO);
    });

    it('con el rechazo abierto, aplicar sugerencias actualiza el motivo propuesto como un clic', async () => {
      const { el, boton, refrescar } = await montar({ lectura: () => of(LEIDA) });
      boton('Rechazar').click();
      await refrescar();
      boton('Aplicar sugerencias').click();
      await refrescar();
      expect(el.querySelector<HTMLTextAreaElement>('#motivo-rechazo')!.value).toBe(
        'No cumple: el comprobante es legible y está completo; es una boleta o factura válida; la fecha corresponde al gasto.',
      );
    });

    it('sin sugerencias no hay panel, «Aplicar sugerencias» ni notas junto a las casillas', async () => {
      const { el, boton, sugerencias, lecturaTexto } = await montar();
      expect(el.querySelector('.lectura')).toBeNull();
      expect(boton('Aplicar sugerencias')).toBeUndefined();
      expect(sugerencias()).toEqual({});
      expect(lecturaTexto()).toBe('');
    });

    it('las sugerencias de criterios que el cliente no revisa no se muestran ni se aplican', async () => {
      const propio: ChecklistCliente = {
        cliente_id: 1, personalizado: true,
        criterios: [{ clave: 'legible', texto: 'Se lee completo' }, { clave: 'c-9f2a11', texto: 'Lleva sello de recepción' }],
      };
      const { boton, sugerencias } = await montar({ checklist: () => of(propio), lectura: () => of(LEIDA) });
      expect(sugerencias()).toEqual({});
      expect(boton('Aplicar sugerencias')).toBeUndefined();
    });

    it('mientras lee dice «Leyendo comprobante…» y se puede marcar y aprobar sin esperar', async () => {
      const pendiente = new Subject<LecturaComprobante>();
      const { el, store, boton, casillas, lecturaTexto, refrescar } = await montar({ lectura: () => pendiente, esperar: false });
      await refrescar();
      expect(lecturaTexto()).toBe('Leyendo comprobante…');
      expect(el.querySelector('.lectura')).toBeNull();
      casillas().forEach(c => c.click());
      await refrescar();
      boton('Aprobar').click();
      expect(store.aprobar).toHaveBeenCalledWith(9, TODO);
    });

    it.each(['no_disponible', 'error', 'sin_texto'] as const)('si la lectura queda «%s», avisa con discreción y todo lo demás sigue igual', async estado => {
      const { el, store, boton, marcar, sugerencias, lecturaTexto } = await montar({ lectura: () => of(lecturaCon(estado)) });
      expect(lecturaTexto()).toBe('No se pudo leer el comprobante automáticamente');
      expect(el.querySelector('.checklist__lectura [role="alert"], .checklist__lectura .alerta')).toBeNull();
      expect(sugerencias()).toEqual({});
      expect(boton('Aplicar sugerencias')).toBeUndefined();
      await marcar(5);
      boton('Aprobar').click();
      expect(store.aprobar).toHaveBeenCalledWith(9, TODO);
    });

    it('si falla la red al leer, el aviso usa el mensaje general de la app', async () => {
      const { lecturaTexto } = await montar({ lectura: () => throwError(() => new HttpErrorResponse({ status: 0 })) });
      expect(lecturaTexto()).toBe(MSG_ERROR_RED);
      TestBed.resetTestingModule();
      const lenta = await montar({ lectura: () => throwError(() => new TimeoutError()) });
      expect(lenta.lecturaTexto()).toBe(MSG_TIMEOUT);
    });

    it('con otro comprobante no quedan las sugerencias del anterior', async () => {
      const otra = new Subject<LecturaComprobante>();
      const { f, lecturas, boton, sugerencias, lecturaTexto } = await montar({
        lectura: id => (id === 9 ? of(LEIDA) : otra),
      });
      expect(boton('Aplicar sugerencias')).toBeDefined();
      f.componentRef.setInput('comprobante', { ...comprobante, id: 10 });
      f.detectChanges();
      f.detectChanges();
      expect(lecturas.lectura).toHaveBeenLastCalledWith(10);
      expect(lecturaTexto()).toBe('Leyendo comprobante…');
      expect(sugerencias()).toEqual({});
      expect(boton('Aplicar sugerencias')).toBeUndefined();
    });
  });

  describe('corregir el gasto con lo leído', () => {
    const corregibles = (el: HTMLElement) => [...el.querySelectorAll<HTMLButtonElement>('.lectura__corregir')];

    it('ofrece corregir el monto y la fecha que no coinciden, y nada más', async () => {
      const { el } = await montar({ lectura: () => of(DESCUADRADA) });
      expect(corregibles(el).map(b => b.textContent?.trim())).toEqual(['Corregir gasto a S/ 513.00', 'Corregir gasto a 26/09/2019']);
    });

    it('sin valores leídos (o sin diferencias) no ofrece corregir', async () => {
      const { el } = await montar({ lectura: () => of(LEIDA) });
      expect(corregibles(el)).toHaveLength(0);
    });

    it('abre un formulario con el valor leído, el actual y pide el motivo', async () => {
      const { el, refrescar } = await montar({ lectura: () => of(DESCUADRADA) });
      corregibles(el)[0].click();
      await refrescar();
      const form = el.querySelector('.lectura__correccion')!;
      expect(form.querySelector('h4')?.textContent?.trim()).toBe('Corregir el monto del gasto');
      expect(form.textContent).toContain('Ahora: S/ 85.50');
      expect(form.querySelector<HTMLInputElement>('input[name="valor"]')!.value).toBe('513');
      expect(form.querySelector('input[name="motivo"]')).not.toBeNull();
    });

    it('exige el motivo y no llama al servicio sin él', async () => {
      const { el, store, boton, refrescar } = await montar({ lectura: () => of(DESCUADRADA) });
      corregibles(el)[0].click();
      await refrescar();
      boton('Confirmar corrección').click();
      await refrescar();
      expect(el.querySelector('.lectura__correccion .campo__error')?.textContent?.trim()).toBe('Indica el motivo de la corrección.');
      expect(store.corregir).not.toHaveBeenCalled();
    });

    it('confirma: manda el monto y el motivo, cierra el formulario y vuelve a leer el comprobante', async () => {
      const { el, store, lecturas, boton, refrescar } = await montar({ lectura: () => of(DESCUADRADA) });
      expect(lecturas.lectura).toHaveBeenCalledTimes(1);
      corregibles(el)[0].click();
      await refrescar();
      const motivo = el.querySelector<HTMLInputElement>('.lectura__correccion input[name="motivo"]')!;
      motivo.value = 'Error de digitación';
      motivo.dispatchEvent(new Event('input'));
      await refrescar();
      boton('Confirmar corrección').click();
      await refrescar();
      expect(store.corregir).toHaveBeenCalledWith(7, { monto: 513, motivo: 'Error de digitación' });
      expect(el.querySelector('.lectura__correccion')).toBeNull();
      expect(lecturas.lectura).toHaveBeenCalledTimes(2);
    });

    it('corrige la fecha con el formato del servicio', async () => {
      const { el, store, boton, refrescar } = await montar({ lectura: () => of(DESCUADRADA) });
      corregibles(el)[1].click();
      await refrescar();
      expect(el.querySelector('.lectura__correccion h4')?.textContent?.trim()).toBe('Corregir la fecha del gasto');
      const motivo = el.querySelector<HTMLInputElement>('.lectura__correccion input[name="motivo"]')!;
      motivo.value = 'El comprobante es del 26/09';
      motivo.dispatchEvent(new Event('input'));
      await refrescar();
      boton('Confirmar corrección').click();
      expect(store.corregir).toHaveBeenCalledWith(7, { fecha: '2019-09-26', motivo: 'El comprobante es del 26/09' });
    });

    it('un monto que no sirve se avisa y no se envía', async () => {
      const { el, store, boton, refrescar } = await montar({ lectura: () => of(DESCUADRADA) });
      corregibles(el)[0].click();
      await refrescar();
      const valor = el.querySelector<HTMLInputElement>('.lectura__correccion input[name="valor"]')!;
      valor.value = '0';
      valor.dispatchEvent(new Event('input'));
      const motivo = el.querySelector<HTMLInputElement>('.lectura__correccion input[name="motivo"]')!;
      motivo.value = 'Error';
      motivo.dispatchEvent(new Event('input'));
      await refrescar();
      boton('Confirmar corrección').click();
      await refrescar();
      expect(el.querySelector('.lectura__correccion .campo__error')?.textContent?.trim()).toBe('Indica un monto mayor a 0.');
      expect(store.corregir).not.toHaveBeenCalled();
    });

    it('Cancelar cierra el formulario sin tocar el gasto', async () => {
      const { el, store, boton, refrescar } = await montar({ lectura: () => of(DESCUADRADA) });
      corregibles(el)[0].click();
      await refrescar();
      boton('Cancelar').click();
      await refrescar();
      expect(el.querySelector('.lectura__correccion')).toBeNull();
      expect(store.corregir).not.toHaveBeenCalled();
    });

    it('si el servidor dice que el gasto ya no se puede corregir (409), lo explica y recarga', async () => {
      const toast = { show: vi.fn() };
      TestBed.overrideProvider(ToastService, { useValue: toast });
      const { el, store, boton, refrescar } = await montar({
        lectura: () => of(DESCUADRADA),
        correccion: () => throwError(() => new HttpErrorResponse({ status: 409 })),
      });
      corregibles(el)[0].click();
      await refrescar();
      const motivo = el.querySelector<HTMLInputElement>('.lectura__correccion input[name="motivo"]')!;
      motivo.value = 'Error de digitación';
      motivo.dispatchEvent(new Event('input'));
      await refrescar();
      boton('Confirmar corrección').click();
      await refrescar();
      expect(toast.show).toHaveBeenCalledWith('Este gasto ya no se puede corregir.', '', 'warn');
      expect(store.cargar).toHaveBeenCalled();
    });
  });
});
