import { Injectable, signal, computed } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Router } from '@angular/router';
import { tap, timeout } from 'rxjs';
import { LoginRequest, TokenResponse, UsuarioOut, exigeCambioDeClave } from '../models/auth.model';
import { CambiarClaveRequest } from '../models/usuario';
import { HTTP_TIMEOUT_MS } from '../utils/error-messages';
import { ApiService } from './api.service';

const TOKEN_KEY = 'lizar_token';
const USER_KEY  = 'lizar_user';

/** Una sesión guardada antes del cambio de roles todavía dice «operador»: ahora es «logistica». */
function normalizar(u: UsuarioOut | null): UsuarioOut | null {
  return u && (u.rol as string) === 'operador' ? { ...u, rol: 'logistica' } : u;
}

function usuarioGuardado(): UsuarioOut | null {
  try {
    return normalizar(JSON.parse(localStorage.getItem(USER_KEY) ?? 'null'));
  } catch {
    return null;
  }
}

@Injectable({ providedIn: 'root' })
export class AuthService {
  private _token   = signal<string | null>(localStorage.getItem(TOKEN_KEY));
  private _usuario = signal<UsuarioOut | null>(usuarioGuardado());

  readonly token   = this._token.asReadonly();
  readonly usuario = this._usuario.asReadonly();
  readonly isAuth  = computed(() => !!this._token());
  readonly isAdmin = computed(() => this._usuario()?.rol === 'admin');
  /** Solo Logística registra gastos y adjunta, revisa o configura; el administrador consulta */
  readonly esLogistica = computed(() => this._usuario()?.rol === 'logistica');
  /** Cuenta nueva o con la clave restablecida: no puede usar la app hasta cambiar la contraseña */
  readonly debeCambiarClave = computed(() => this._usuario()?.debe_cambiar_clave === true);

  constructor(private http: HttpClient, private router: Router, private api: ApiService) {}

  login(body: LoginRequest) {
    return this.http
      .post<TokenResponse>(`${this.api.base()}/auth/login`, body)
      .pipe(
        tap(res => {
          this._token.set(res.access_token);
          this.guardarUsuario({ ...res.usuario, debe_cambiar_clave: exigeCambioDeClave(res) });
          localStorage.setItem(TOKEN_KEY, res.access_token);
        })
      );
  }

  /** El rol sale de la base de datos en cada petición; esto trae el usuario real para que la pantalla lo refleje. */
  refrescar(): void {
    if (!this._token()) return;
    this.http.get<UsuarioOut>(`${this.api.base()}/auth/me`).subscribe({
      next: u => {
        this.guardarUsuario(u);
        if (u.debe_cambiar_clave) this.irACambiarClave();
      },
      error: () => {}, // un 401 lo atiende el interceptor; otro fallo deja el usuario guardado
    });
  }

  /** Con la clave actual y la nueva; el servidor responde con el usuario ya sin la marca de cambio pendiente. */
  cambiarClave(body: CambiarClaveRequest) {
    return this.http
      .post<UsuarioOut>(`${this.api.base()}/auth/cambiar-clave`, body)
      .pipe(tap(u => this.guardarUsuario(u)));
  }

  /** El servidor rechazó algo por clave pendiente (403): se marca la sesión y se lleva a la pantalla para cambiarla. */
  exigirCambioClave(): void {
    const u = this._usuario();
    if (u && !u.debe_cambiar_clave) this.guardarUsuario({ ...u, debe_cambiar_clave: true });
    this.irACambiarClave();
  }

  /**
   * Cierra la sesión aquí al instante y avisa al servidor para que quede en la bitácora.
   * El aviso no frena la salida: si falla o tarda, la sesión local ya está cerrada.
   */
  logout(): void {
    const token = this._token();
    this.limpiar();
    this.router.navigate(['/login']);
    if (!token) return;
    // El token se manda a mano porque la sesión ya se limpió y el interceptor no lo encontraría
    this.http
      .post(`${this.api.base()}/auth/logout`, null, { headers: new HttpHeaders({ Authorization: `Bearer ${token}` }) })
      .pipe(timeout(HTTP_TIMEOUT_MS))
      .subscribe({ error: () => {} });
  }

  /** El servidor ya no acepta el token: se cierra la sesión y, tras ingresar de nuevo, se vuelve a la pantalla en que estaba. */
  sesionVencida(): void {
    const volver = this.router.url;
    this.limpiar();
    this.router.navigate(['/login'], volver.startsWith('/app/') ? { queryParams: { volver } } : {});
  }

  /** Lleva a «Cambiar contraseña» recordando la pantalla en que estaba, salvo que ya esté ahí. */
  private irACambiarClave(): void {
    const volver = this.router.url;
    if (volver.startsWith('/cambiar-clave')) return;
    this.router.navigate(['/cambiar-clave'], volver.startsWith('/app/') ? { queryParams: { volver } } : {});
  }

  private guardarUsuario(u: UsuarioOut): void {
    const usuario = normalizar(u);
    this._usuario.set(usuario);
    localStorage.setItem(USER_KEY, JSON.stringify(usuario));
  }

  private limpiar(): void {
    this._token.set(null);
    this._usuario.set(null);
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(USER_KEY);
  }
}
