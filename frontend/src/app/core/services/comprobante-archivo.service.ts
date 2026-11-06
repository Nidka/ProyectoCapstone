import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, map } from 'rxjs';
import { GastosApiService } from './gastos-api.service';
import { ToastService } from './toast.service';

/**
 * Los comprobantes subidos exigen sesión (SEG-01): el navegador no manda el token en <img> ni en
 * <a href>, así que se descargan con HttpClient (el interceptor agrega el Bearer) y se muestran
 * desde una URL temporal de Blob.
 */
@Injectable({ providedIn: 'root' })
export class ComprobanteArchivoService {
  private readonly http = inject(HttpClient);
  private readonly api = inject(GastosApiService);
  private readonly toast = inject(ToastService);

  descargar(archivoUrl: string): Observable<Blob> {
    return this.http.get(this.api.fileUrl(archivoUrl), { responseType: 'blob' });
  }

  /** URL temporal para <img>; quien la pide debe liberarla con URL.revokeObjectURL. */
  urlTemporal(archivoUrl: string): Observable<string> {
    return this.descargar(archivoUrl).pipe(map(blob => URL.createObjectURL(blob)));
  }

  /** Abre el comprobante en otra pestaña. La pestaña se abre al instante para no chocar con el bloqueador de ventanas. */
  abrir(archivoUrl: string): void {
    const ventana = window.open('', '_blank');
    this.descargar(archivoUrl).subscribe({
      next: blob => {
        const url = URL.createObjectURL(blob);
        if (ventana) ventana.location.href = url;
        else window.open(url, '_blank');
        setTimeout(() => URL.revokeObjectURL(url), 60_000);
      },
      error: () => {
        ventana?.close();
        this.toast.show('No se pudo abrir el comprobante', 'Vuelve a intentarlo o inicia sesión de nuevo.', 'err');
      },
    });
  }
}
