import { afterEach, describe, expect, it, vi } from 'vitest';
import { Component, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { CampanaComponent } from './campana.component';
import { NotificacionesStore } from '../../core/state/notificaciones.store';
import { Notificacion } from '../../core/models/notificacion';

function aviso(id: number, extra: Partial<Notificacion> = {}): Notificacion {
  return {
    id, tipo: 'sin_comprobante', id_gasto: 10 + id, mensaje: `El gasto N° ${10 + id} lleva más de 3 días sin comprobante de pago.`,
    creada_en: '2026-10-02T19:05:00Z', leida: false, leida_en: null, ...extra,
  };
}

function storeFalso(lista: Notificacion[] = [], noLeidas = lista.filter(a => !a.leida).length) {
  return {
    lista: signal(lista),
    noLeidas: signal(noLeidas),
    cargando: signal(false),
    cargado: signal(true),
    error: signal(false),
    iniciarSondeo: vi.fn(),
    detenerSondeo: vi.fn(),
    actualizarSiViejo: vi.fn(),
    cargar: vi.fn(),
    marcarLeida: vi.fn(),
  };
}

@Component({ selector: 'app-destino', template: '' })
class DestinoComponent {}

let fixture: ComponentFixture<CampanaComponent>;

function montar(store = storeFalso()) {
  TestBed.configureTestingModule({ providers: [provideRouter([{ path: 'app/gastos/:id', component: DestinoComponent }]), { provide: NotificacionesStore, useValue: store }] });
  fixture = TestBed.createComponent(CampanaComponent);
  // Con el elemento en el documento el foco y los clics de verdad funcionan
  document.body.appendChild(fixture.nativeElement);
  fixture.detectChanges();
  const el = fixture.nativeElement as HTMLElement;
  return { store, el };
}

const boton = (el: HTMLElement) => el.querySelector('.avisos__boton') as HTMLButtonElement;
const panel = (el: HTMLElement) => el.querySelector('.avisos__panel') as HTMLElement | null;
const items = (el: HTMLElement) => [...el.querySelectorAll('.avisos__item')] as HTMLAnchorElement[];

function abrir(el: HTMLElement): void {
  boton(el).click();
  fixture.detectChanges();
}

function tecla(destino: Element, key: string): KeyboardEvent {
  const e = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true });
  destino.dispatchEvent(e);
  fixture.detectChanges();
  return e;
}

describe('CampanaComponent', () => {
  afterEach(() => {
    fixture?.nativeElement.remove();
    document.body.innerHTML = '';
  });

  it('inicia el sondeo al aparecer y lo detiene al desaparecer', () => {
    const { store } = montar();
    expect(store.iniciarSondeo).toHaveBeenCalledTimes(1);
    expect(store.detenerSondeo).not.toHaveBeenCalled();
    fixture.destroy();
    expect(store.detenerSondeo).toHaveBeenCalledTimes(1);
  });

  it('su contenedor no usa la clase «avisos», que es la de los toasts fijos abajo a la derecha', () => {
    const { el } = montar();
    expect(el.querySelector('.avisos')).toBeNull();
    expect(el.querySelector('.campana')).not.toBeNull();
  });

  describe('el botón', () => {
    it('sin avisos pendientes no tiene insignia y lo dice en su nombre', () => {
      const { el } = montar();
      expect(el.querySelector('.avisos__insignia')).toBeNull();
      expect(boton(el).getAttribute('aria-label')).toBe('Avisos, ninguno sin leer');
      expect(boton(el).getAttribute('aria-expanded')).toBe('false');
      expect(boton(el).getAttribute('aria-haspopup')).toBe('dialog');
      expect(boton(el).getAttribute('aria-controls')).toBeNull();
      expect(panel(el)).toBeNull();
    });

    it('con pendientes muestra el número en la insignia y en su nombre', () => {
      const store = storeFalso([aviso(1), aviso(2), aviso(3, { leida: true })]);
      const { el } = montar(store);
      expect(el.querySelector('.avisos__insignia')?.textContent?.trim()).toBe('2');
      expect(boton(el).getAttribute('aria-label')).toBe('Avisos, 2 sin leer');
    });

    it('en singular dice «1 sin leer» y la insignia se corta en 99+', () => {
      const { el, store } = montar(storeFalso([aviso(1)]));
      expect(boton(el).getAttribute('aria-label')).toBe('Avisos, 1 sin leer');
      store.noLeidas.set(150);
      fixture.detectChanges();
      expect(el.querySelector('.avisos__insignia')?.textContent?.trim()).toBe('99+');
      expect(boton(el).getAttribute('aria-label')).toBe('Avisos, 150 sin leer');
    });

    it('la insignia no se lee dos veces: queda oculta a los lectores de pantalla', () => {
      const { el } = montar(storeFalso([aviso(1)]));
      expect(el.querySelector('.avisos__insignia')?.getAttribute('aria-hidden')).toBe('true');
    });
  });

  describe('la lista', () => {
    it('al pulsar el botón se abre como diálogo con nombre y actualiza los avisos', () => {
      const { el, store } = montar(storeFalso([aviso(1)]));
      abrir(el);
      expect(boton(el).getAttribute('aria-expanded')).toBe('true');
      expect(boton(el).getAttribute('aria-controls')).toBe(panel(el)?.id);
      expect(panel(el)?.getAttribute('role')).toBe('dialog');
      const titulo = el.querySelector(`#${panel(el)?.getAttribute('aria-labelledby')}`);
      expect(titulo?.textContent?.trim()).toBe('Avisos');
      expect(store.actualizarSiViejo).toHaveBeenCalledTimes(1);
    });

    it('pulsar el botón otra vez la cierra', () => {
      const { el } = montar(storeFalso([aviso(1)]));
      abrir(el);
      abrir(el);
      expect(panel(el)).toBeNull();
      expect(boton(el).getAttribute('aria-expanded')).toBe('false');
    });

    it('cada aviso lleva su título, el texto del servicio, la fecha en hora de Lima y un enlace al gasto', () => {
      const { el } = montar(storeFalso([
        aviso(1),
        aviso(2, { tipo: 'cobro_proximo', id_gasto: 7, mensaje: 'Faltan 2 días para el cobro del gasto N° 7 (24/10/2026) y su comprobante aún no está aprobado.' }),
      ]));
      abrir(el);
      const filas = items(el);
      expect(filas).toHaveLength(2);
      expect(filas[0].getAttribute('href')).toBe('/app/gastos/11');
      expect(filas[0].textContent).toContain('Falta el comprobante');
      expect(filas[0].textContent).toContain('El gasto N° 11 lleva más de 3 días sin comprobante de pago.');
      expect(filas[0].textContent).toContain('02/10/2026 14:05');
      expect(filas[0].querySelector('time')?.getAttribute('datetime')).toBe('2026-10-02T19:05:00Z');
      expect(filas[1].getAttribute('href')).toBe('/app/gastos/7');
      expect(filas[1].textContent).toContain('Cobro cerca');
      expect(filas[1].textContent).toContain('Faltan 2 días para el cobro del gasto N° 7');
    });

    it('los avisos sin leer se distinguen con texto, no solo con color', () => {
      const { el } = montar(storeFalso([aviso(1), aviso(2, { leida: true })]));
      abrir(el);
      const [nuevo, leido] = items(el);
      expect(nuevo.classList).toContain('avisos__item--nuevo');
      expect(nuevo.querySelector('.oculto-visual')?.textContent).toContain('Sin leer');
      expect(leido.classList).not.toContain('avisos__item--nuevo');
      expect(leido.textContent).not.toContain('Sin leer');
    });

    it('pulsar un aviso sin leer lo marca como leído y cierra la lista', () => {
      const { el, store } = montar(storeFalso([aviso(1), aviso(2)]));
      abrir(el);
      items(el)[1].click();
      fixture.detectChanges();
      expect(store.marcarLeida).toHaveBeenCalledWith(2);
      expect(panel(el)).toBeNull();
    });

    it('pulsar un aviso ya leído no lo vuelve a marcar, pero igual cierra la lista', () => {
      const { el, store } = montar(storeFalso([aviso(1, { leida: true })]));
      abrir(el);
      items(el)[0].click();
      fixture.detectChanges();
      expect(store.marcarLeida).not.toHaveBeenCalled();
      expect(panel(el)).toBeNull();
    });

    it('abrir la lista no marca nada como leído por sí solo', () => {
      const { el, store } = montar(storeFalso([aviso(1)]));
      abrir(el);
      expect(store.marcarLeida).not.toHaveBeenCalled();
    });
  });

  describe('teclado y foco', () => {
    it('al abrir, el foco va al primer aviso', () => {
      const { el } = montar(storeFalso([aviso(1), aviso(2)]));
      abrir(el);
      expect(document.activeElement).toBe(items(el)[0]);
    });

    it('si no hay avisos, el foco va al propio diálogo', () => {
      const { el } = montar();
      abrir(el);
      expect(document.activeElement).toBe(panel(el));
    });

    it('Escape cierra la lista y devuelve el foco al botón', () => {
      const { el } = montar(storeFalso([aviso(1)]));
      abrir(el);
      tecla(items(el)[0], 'Escape');
      expect(panel(el)).toBeNull();
      expect(document.activeElement).toBe(boton(el));
    });

    it('Escape con la lista cerrada no hace nada', () => {
      const { el } = montar(storeFalso([aviso(1)]));
      boton(el).focus();
      const e = tecla(boton(el), 'Escape');
      expect(e.defaultPrevented).toBe(false);
      expect(panel(el)).toBeNull();
    });

    it('las flechas recorren los avisos y Inicio y Fin saltan a los extremos', () => {
      const { el } = montar(storeFalso([aviso(1), aviso(2), aviso(3)]));
      abrir(el);
      const [a, b, c] = items(el);
      expect(tecla(a, 'ArrowDown').defaultPrevented).toBe(true);
      expect(document.activeElement).toBe(b);
      tecla(b, 'ArrowDown');
      expect(document.activeElement).toBe(c);
      tecla(c, 'ArrowDown');
      expect(document.activeElement).toBe(c);
      tecla(c, 'ArrowUp');
      expect(document.activeElement).toBe(b);
      tecla(b, 'Home');
      expect(document.activeElement).toBe(a);
      tecla(a, 'ArrowUp');
      expect(document.activeElement).toBe(a);
      tecla(a, 'End');
      expect(document.activeElement).toBe(c);
    });

    it('desde el diálogo (sin foco en un aviso) ArrowDown va al primero', () => {
      const { el } = montar(storeFalso([aviso(1), aviso(2)]));
      abrir(el);
      panel(el)!.focus();
      tecla(panel(el)!, 'ArrowDown');
      expect(document.activeElement).toBe(items(el)[0]);
    });

    it('un clic fuera la cierra; un clic dentro no', () => {
      const { el } = montar(storeFalso([aviso(1)]));
      const afuera = document.createElement('button');
      document.body.appendChild(afuera);
      abrir(el);
      panel(el)!.click();
      fixture.detectChanges();
      expect(panel(el)).not.toBeNull();
      afuera.click();
      fixture.detectChanges();
      expect(panel(el)).toBeNull();
    });

    it('al salir con Tab hacia otro elemento de la página se cierra', () => {
      const { el } = montar(storeFalso([aviso(1)]));
      const otro = document.createElement('button');
      document.body.appendChild(otro);
      abrir(el);
      items(el)[0].dispatchEvent(new FocusEvent('focusout', { bubbles: true, relatedTarget: otro }));
      fixture.detectChanges();
      expect(panel(el)).toBeNull();
    });

    it('mover el foco dentro de la lista no la cierra', () => {
      const { el } = montar(storeFalso([aviso(1), aviso(2)]));
      abrir(el);
      const [a, b] = items(el);
      a.dispatchEvent(new FocusEvent('focusout', { bubbles: true, relatedTarget: b }));
      fixture.detectChanges();
      expect(panel(el)).not.toBeNull();
    });
  });

  describe('estados', () => {
    it('sin avisos dice que no hay nada pendiente', () => {
      const { el } = montar();
      abrir(el);
      expect(panel(el)?.textContent).toContain('No tienes avisos por ahora.');
      expect(items(el)).toHaveLength(0);
    });

    it('mientras llegan los primeros avisos muestra un esqueleto marcado como ocupado', () => {
      const store = storeFalso();
      store.cargado.set(false);
      store.cargando.set(true);
      const { el } = montar(store);
      abrir(el);
      expect(panel(el)?.querySelector('[aria-busy="true"] .esqueleto')).not.toBeNull();
      expect(panel(el)?.textContent).not.toContain('No tienes avisos');
    });

    it('si no se pudieron cargar dice el problema y ofrece Reintentar', () => {
      const store = storeFalso();
      store.cargado.set(false);
      store.error.set(true);
      const { el } = montar(store);
      abrir(el);
      expect(panel(el)?.textContent).toContain('No se pudieron cargar los avisos.');
      const reintentar = [...panel(el)!.querySelectorAll('button')].find(b => b.textContent?.trim() === 'Reintentar')!;
      reintentar.click();
      expect(store.cargar).toHaveBeenCalled();
    });

    it('si falla una actualización conserva la lista que ya tenía y avisa', () => {
      const store = storeFalso([aviso(1)]);
      store.error.set(true);
      const { el } = montar(store);
      abrir(el);
      expect(items(el)).toHaveLength(1);
      expect(panel(el)?.querySelector('.avisos__nota')?.textContent).toContain('No se pudo actualizar.');
      [...panel(el)!.querySelectorAll('.avisos__nota button')][0].dispatchEvent(new MouseEvent('click', { bubbles: true }));
      expect(store.cargar).toHaveBeenCalled();
    });

    it('la cabecera resume cuántos avisos hay sin leer', () => {
      const store = storeFalso([aviso(1)], 140);
      const { el } = montar(store);
      abrir(el);
      expect(panel(el)?.querySelector('.avisos__resumen')?.textContent).toContain('140 sin leer');
    });
  });
});
