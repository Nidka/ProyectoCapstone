import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { ChecklistService } from './checklist.service';
import { GastosApiService } from './gastos-api.service';

function montar() {
  const respuesta = { cliente_id: 3, personalizado: true, criterios: [{ clave: 'legible', texto: 'Es legible' }] };
  const api = { get: vi.fn(() => of(respuesta)), put: vi.fn(() => of(respuesta)), delete: vi.fn(() => of(respuesta)) };
  TestBed.configureTestingModule({ providers: [{ provide: GastosApiService, useValue: api }] });
  return { servicio: TestBed.inject(ChecklistService), api, respuesta };
}

describe('ChecklistService', () => {
  it('lee los criterios vigentes de un cliente', () => {
    const { servicio, api, respuesta } = montar();
    let recibido: unknown;
    servicio.deCliente(3).subscribe(r => (recibido = r));
    expect(api.get).toHaveBeenCalledWith('/clientes/3/checklist');
    expect(recibido).toEqual(respuesta);
  });

  it('guarda la lista completa de criterios, con clave solo los que ya la tenían', () => {
    const { servicio, api } = montar();
    servicio.guardar(3, [{ clave: 'legible', texto: 'Es legible' }, { texto: 'Lleva sello' }]).subscribe();
    expect(api.put).toHaveBeenCalledWith('/clientes/3/checklist', {
      criterios: [{ clave: 'legible', texto: 'Es legible' }, { texto: 'Lleva sello' }],
    });
  });

  it('restablece al estándar', () => {
    const { servicio, api } = montar();
    servicio.restablecer(3).subscribe();
    expect(api.delete).toHaveBeenCalledWith('/clientes/3/checklist');
  });
});
