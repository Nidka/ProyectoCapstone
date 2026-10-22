import { CrudField } from '../../core/models/crud.model';

export interface EstadoCampo {
  field: CrudField;
  value: string;
  error: string;
}

/** Valida con las reglas de cada módulo y los mismos mensajes de la vista anterior. Escribe `error` en cada campo. */
export function validarCampos(campos: EstadoCampo[]): boolean {
  let valido = true;
  for (const campo of campos) {
    campo.error = '';
    const v = String(campo.value ?? '').trim();
    if (campo.field.req && !v) {
      campo.error = 'Campo obligatorio';
      valido = false;
      continue;
    }
    if (v && campo.field.rule && !campo.field.rule(v)) {
      campo.error = campo.field.msg ?? 'Inválido';
      valido = false;
      continue;
    }
    if (campo.field.tp === 'number' && v && isNaN(Number(v))) {
      campo.error = 'Debe ser número';
      valido = false;
    }
  }
  return valido;
}

/** «asignaciones, clientes y direcciones» */
export function enumerar(partes: string[]): string {
  if (partes.length <= 1) return partes.join('');
  return `${partes.slice(0, -1).join(', ')} y ${partes[partes.length - 1]}`;
}
