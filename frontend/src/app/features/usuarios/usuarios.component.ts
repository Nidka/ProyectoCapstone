import { ChangeDetectionStrategy, Component, ElementRef, computed, inject, signal, viewChild } from '@angular/core';
import { rxResource } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { ROL_ETIQUETA, Rol, UsuarioOut } from '../../core/models/auth.model';
import {
  CampoUsuario, ErroresUsuario, MAX_EMAIL, MAX_NOMBRE, MAX_USERNAME, MIN_CLAVE, ValoresUsuario,
  cambiosDeUsuario, erroresDeUsuario, motivoDelServidor, validarUsuario,
} from '../../core/models/usuario';
import { AuthService } from '../../core/services/auth.service';
import { ToastService } from '../../core/services/toast.service';
import { UsuariosService } from '../../core/services/usuarios.service';
import { normalizar } from '../../core/utils/texto';
import { EstadoVacioComponent } from '../../shared/ui/estado-vacio/estado-vacio.component';
import { IconoComponent } from '../../shared/ui/icono/icono.component';

const VALORES_NUEVOS: ValoresUsuario = { username: '', email: '', nombre: '', rol: 'logistica', password: '' };

interface Confirmacion {
  id: number;
  accion: 'desactivar' | 'activar';
}

/** Gestión de usuarios (solo administrador): lista, crear, editar, desactivar y activar. Nunca se borra un usuario. */
@Component({
  selector: 'app-usuarios',
  imports: [FormsModule, IconoComponent, EstadoVacioComponent],
  templateUrl: './usuarios.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class UsuariosComponent {
  private readonly servicio = inject(UsuariosService);
  private readonly auth = inject(AuthService);
  private readonly toast = inject(ToastService);
  private readonly hoja = viewChild.required<ElementRef<HTMLDialogElement>>('hoja');

  protected readonly lista = rxResource({ stream: () => this.servicio.listar() });
  protected readonly usuarios = computed<UsuarioOut[] | undefined>(() => (this.lista.hasValue() ? this.lista.value() : undefined));
  protected readonly cargando = computed(() => !this.usuarios() && !this.lista.error());
  protected readonly falloCarga = computed(() => !!this.lista.error() && !this.usuarios());

  protected readonly termino = signal('');
  protected readonly filas = computed(() => {
    const t = normalizar(this.termino()).trim();
    return (this.usuarios() ?? []).filter(u => !t || normalizar(`${u.nombre} ${u.username} ${u.email}`).includes(t));
  });

  /** El administrador que está viendo la pantalla: no puede desactivarse ni quitarse el rol */
  protected readonly yo = computed(() => this.auth.usuario()?.id_usuario);

  // Formulario de crear / editar (hoja lateral)
  protected readonly editando = signal<UsuarioOut | null>(null);
  protected readonly valores = signal<ValoresUsuario>(VALORES_NUEVOS);
  protected readonly errores = signal<ErroresUsuario>({});
  protected readonly errorGuardado = signal('');
  protected readonly guardando = signal(false);
  protected readonly mostrarClave = signal(false);
  protected readonly modo = computed<'crear' | 'editar'>(() => (this.editando() ? 'editar' : 'crear'));
  protected readonly esPropio = computed(() => !!this.editando() && this.editando()!.id_usuario === this.yo());
  protected readonly sinCambios = computed(() => {
    const original = this.editando();
    return !!original && Object.keys(cambiosDeUsuario(original, this.valores())).length === 0;
  });

  // Confirmación en línea de desactivar / activar
  protected readonly confirmando = signal<Confirmacion | null>(null);
  protected readonly trabajando = signal(false);
  protected readonly errorAccion = signal('');

  protected readonly roles: readonly Rol[] = ['logistica', 'admin'];
  protected readonly rolEtiqueta = ROL_ETIQUETA;
  protected readonly limites = { username: MAX_USERNAME, email: MAX_EMAIL, nombre: MAX_NOMBRE, clave: MIN_CLAVE };

  protected buscar(evento: Event): void {
    this.termino.set((evento.target as HTMLInputElement).value);
  }

  protected reintentar(): void {
    this.lista.reload();
  }

  protected nuevo(): void {
    this.abrirHoja(null, VALORES_NUEVOS);
  }

  protected editar(u: UsuarioOut): void {
    this.abrirHoja(u, { username: u.username, email: u.email, nombre: u.nombre, rol: u.rol, password: '' });
  }

  private abrirHoja(usuario: UsuarioOut | null, valores: ValoresUsuario): void {
    this.confirmando.set(null);
    this.editando.set(usuario);
    this.valores.set(valores);
    this.errores.set({});
    this.errorGuardado.set('');
    this.mostrarClave.set(false);
    this.hoja().nativeElement.showModal();
  }

  protected cerrarHoja(): void {
    this.hoja().nativeElement.close();
  }

  protected cerrarSiFondo(evento: MouseEvent): void {
    if (evento.target === this.hoja().nativeElement) this.cerrarHoja();
  }

  /** Al corregir un campo se le quita su error; el resto se vuelve a revisar al guardar. */
  protected poner<C extends CampoUsuario>(campo: C, valor: ValoresUsuario[C]): void {
    this.valores.update(v => ({ ...v, [campo]: valor }));
    if (this.errores()[campo]) this.errores.update(e => ({ ...e, [campo]: undefined }));
  }

  protected guardar(): void {
    if (this.guardando()) return;
    const valores = this.valores();
    const original = this.editando();
    const errores = validarUsuario(valores, this.modo());
    this.errores.set(errores);
    this.errorGuardado.set('');
    if (Object.keys(errores).length) return;
    if (original && this.sinCambios()) return;

    this.guardando.set(true);
    const limpio: ValoresUsuario = { ...valores, username: valores.username.trim(), email: valores.email.trim(), nombre: valores.nombre.trim() };
    const peticion = original
      ? this.servicio.editar(original.id_usuario, cambiosDeUsuario(original, valores))
      : this.servicio.crear(limpio);
    peticion.subscribe({
      next: guardado => {
        this.guardando.set(false);
        this.lista.set(original ? this.usuarios()!.map(u => (u.id_usuario === guardado.id_usuario ? guardado : u)) : [...(this.usuarios() ?? []), guardado]);
        this.cerrarHoja();
        this.toast.show(original ? 'Cambios guardados.' : 'Usuario creado.', '', 'ok');
      },
      error: err => {
        this.guardando.set(false);
        const { campos, general } = erroresDeUsuario(err, valores);
        this.errores.set(campos);
        this.errorGuardado.set(general);
      },
    });
  }

  protected pedirConfirmacion(u: UsuarioOut, accion: Confirmacion['accion']): void {
    this.errorAccion.set('');
    this.confirmando.set({ id: u.id_usuario, accion });
  }

  protected cancelarConfirmacion(): void {
    this.confirmando.set(null);
    this.errorAccion.set('');
  }

  protected confirmar(u: UsuarioOut): void {
    const c = this.confirmando();
    if (!c || c.id !== u.id_usuario || this.trabajando()) return;
    this.trabajando.set(true);
    this.errorAccion.set('');
    const peticion = c.accion === 'desactivar' ? this.servicio.desactivar(u.id_usuario) : this.servicio.activar(u.id_usuario);
    peticion.subscribe({
      next: actualizado => {
        this.trabajando.set(false);
        this.confirmando.set(null);
        this.lista.set(this.usuarios()!.map(x => (x.id_usuario === actualizado.id_usuario ? actualizado : x)));
        this.toast.show(c.accion === 'desactivar' ? 'Usuario desactivado.' : 'Usuario activado.', '', 'ok');
      },
      error: err => {
        this.trabajando.set(false);
        this.errorAccion.set(motivoDelServidor(err));
      },
    });
  }
}
