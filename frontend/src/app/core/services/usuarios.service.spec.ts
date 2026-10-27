import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { UsuariosService } from './usuarios.service';

const BASE = 'http://localhost:8081/api/v1/usuarios';
const ANA = { id_usuario: 2, username: 'aquispe', email: 'ana@lizar.pe', nombre: 'Ana Quispe', rol: 'logistica', estado: true, debe_cambiar_clave: false };

function montar() {
  TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
  return { servicio: TestBed.inject(UsuariosService), http: TestBed.inject(HttpTestingController) };
}

describe('UsuariosService', () => {
  it('lista los usuarios, activos e inactivos', () => {
    const { servicio, http } = montar();
    let recibido: unknown;
    servicio.listar().subscribe(r => (recibido = r));
    const req = http.expectOne(BASE);
    expect(req.request.method).toBe('GET');
    req.flush([ANA]);
    expect(recibido).toEqual([ANA]);
  });

  it('crea un usuario con POST', () => {
    const { servicio, http } = montar();
    const cuerpo = { username: 'aquispe', email: 'ana@lizar.pe', nombre: 'Ana Quispe', rol: 'logistica' as const, password: 'clave-larga-1' };
    servicio.crear(cuerpo).subscribe();
    const req = http.expectOne(BASE);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(cuerpo);
    req.flush(ANA, { status: 201, statusText: 'Created' });
  });

  it('edita con PUT enviando solo los cambios', () => {
    const { servicio, http } = montar();
    servicio.editar(2, { nombre: 'Ana M. Quispe' }).subscribe();
    const req = http.expectOne(`${BASE}/2`);
    expect(req.request.method).toBe('PUT');
    expect(req.request.body).toEqual({ nombre: 'Ana M. Quispe' });
    req.flush(ANA);
  });

  it('desactiva y activa con POST sin cuerpo', () => {
    const { servicio, http } = montar();
    servicio.desactivar(2).subscribe();
    const des = http.expectOne(`${BASE}/2/desactivar`);
    expect(des.request.method).toBe('POST');
    des.flush({ ...ANA, estado: false });
    servicio.activar(2).subscribe();
    const act = http.expectOne(`${BASE}/2/activar`);
    expect(act.request.method).toBe('POST');
    act.flush(ANA);
  });
});
