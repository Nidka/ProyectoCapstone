import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { ToastComponent } from './toast.component';
import { ToastService } from '../../../core/services/toast.service';

describe('ToastComponent', () => {
  it('anuncia el aviso con su tipo y permite cerrarlo', () => {
    const f = TestBed.createComponent(ToastComponent);
    const avisos = TestBed.inject(ToastService);
    avisos.show('Gasto registrado correctamente. Ya puedes adjuntar el comprobante de pago.', '', 'ok', 0);
    f.detectChanges();
    const region: HTMLElement = f.nativeElement.querySelector('.avisos');
    expect(region.getAttribute('aria-live')).toBe('polite');
    expect(region.querySelector('.aviso--ok .aviso__titulo')?.textContent).toBe(
      'Gasto registrado correctamente. Ya puedes adjuntar el comprobante de pago.',
    );
    (region.querySelector('.aviso__cerrar') as HTMLButtonElement).click();
    expect(avisos.toasts()).toHaveLength(0);
  });

  it('cada tipo de aviso lleva la señal vial de su clase', () => {
    const f = TestBed.createComponent(ToastComponent);
    const avisos = TestBed.inject(ToastService);
    avisos.show('Listo', '', 'ok', 0);
    avisos.show('Atención', '', 'warn', 0);
    avisos.show('Dato', '', 'info', 0);
    f.detectChanges();
    const el: HTMLElement = f.nativeElement;
    expect(el.querySelector('.aviso--ok app-senal.senal--guia')).not.toBeNull();
    expect(el.querySelector('.aviso--warn app-senal.senal--preventiva')).not.toBeNull();
    expect(el.querySelector('.aviso--info app-senal.senal--informativa')).not.toBeNull();
  });
});
