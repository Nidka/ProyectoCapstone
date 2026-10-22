import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { AdjuntarComprobanteComponent } from './adjuntar-comprobante.component';
import { GastosStore } from '../../../core/state/gastos.store';
import { ToastService } from '../../../core/services/toast.service';
import { Gasto } from '../../../core/models/gasto.model';

const gasto: Gasto = {
  id: 7, viaje_id: 1, cliente_id: 1, monto: 120, fecha: '2026-10-01T00:00:00Z', estado: 'Registrado',
  creado_por: 'Ana', creado_en: '2026-10-01T15:00:00Z',
};

function montar() {
  const adjuntar = vi.fn(() => of({ id: 1 }));
  TestBed.configureTestingModule({ providers: [{ provide: GastosStore, useValue: { adjuntar } }] });
  const f = TestBed.createComponent(AdjuntarComprobanteComponent);
  f.componentRef.setInput('gasto', gasto);
  f.detectChanges();
  const el: HTMLElement = f.nativeElement;
  const subir = () => (el.querySelector('.formulario__acciones .btn') as HTMLButtonElement).click();
  const error = () => el.querySelector('.campo__error')?.textContent?.trim();
  return { f, el, adjuntar, subir, error };
}

describe('AdjuntarComprobanteComponent', () => {
  it('al soltar un archivo no permitido muestra el mensaje exacto y no sube nada', () => {
    const { f, el, adjuntar, subir, error } = montar();
    const soltar = new Event('drop', { bubbles: true, cancelable: true });
    Object.defineProperty(soltar, 'dataTransfer', { value: { files: [new File(['x'], 'notas.txt', { type: 'text/plain' })] } });
    el.querySelector('.soltar')!.dispatchEvent(soltar);
    f.detectChanges();
    expect(error()).toBe('Solo se aceptan archivos JPG, PNG o PDF.');
    subir();
    f.detectChanges();
    expect(adjuntar).not.toHaveBeenCalled();
  });

  it('rechaza archivos de más de 10 MB', () => {
    const { f, error } = montar();
    f.componentInstance.elegir(new File([new Uint8Array(10 * 1024 * 1024 + 1)], 'boleta.pdf', { type: 'application/pdf' }));
    f.detectChanges();
    expect(error()).toBe('El archivo supera el tamaño máximo permitido (10 MB).');
  });

  it('sin archivo pide elegir uno', () => {
    const { f, subir, error } = montar();
    subir();
    f.detectChanges();
    expect(error()).toBe('Selecciona al menos un comprobante antes de continuar.');
  });

  it('sube el archivo elegido al gasto y avisa con el copy exacto', () => {
    const { f, adjuntar, subir } = montar();
    const archivo = new File(['x'], 'boleta.jpg', { type: 'image/jpeg' });
    f.componentInstance.elegir(archivo);
    f.detectChanges();
    subir();
    expect(adjuntar).toHaveBeenCalledWith(7, archivo);
    expect(TestBed.inject(ToastService).toasts()[0].title).toBe('Comprobante adjuntado. El gasto cambió a estado "Pagado".');
  });
});
